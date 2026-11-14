// Putting a page of the real app in a known state: a session, a frozen clock,
// and a stubbed backend.
//
// The backend is stubbed rather than run for real (Postgres + Go + seed) because
// a screenshot has to be byte-identical between runs, and the seeded data is
// not: `cmd/seed` places its matches on "the last N Sundays" with random goals.
// Stubbing also makes states that are awkward to reach live — a closed sign-up
// list, a player who left the group — a matter of a different fixture.

const { expect } = require('@playwright/test');
const data = require('./data');

// Same key as api.js's TOKEN_KEY / activeGroup.js's storage key. Duplicated
// deliberately: a test that reads them from the source would keep passing if the
// app silently changed where it stores a session, which is exactly the kind of
// break worth failing on.
const TOKEN_KEY = 'calciotto-token';
const ACTIVE_GROUP_KEY = 'calciotto-active-group';
const THEME_KEY = 'calciotto-theme';

// The app never verifies the signature client-side — the router only checks that
// a token exists, and MatchDetails/Profile decode the payload for the player id
// (`player_id`). So an unsigned, syntactically valid JWT is enough, and inventing
// one here beats depending on a real /auth/login round-trip.
const fakeJWT = (playerID) => {
  const b64 = (obj) => Buffer.from(JSON.stringify(obj))
    .toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
  return [
    b64({ alg: 'HS256', typ: 'JWT' }),
    b64({ player_id: playerID, exp: Math.floor(data.FIXED_NOW.getTime() / 1000) + 604800 }),
    'not-a-real-signature',
  ].join('.');
};

const json = (route, body) => route.fulfill({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify(body),
});

// The API origin api.js falls back to when VUE_APP_API_BASE_URL is unset, which
// is how the bundle under test is built. Scoped to that origin so a stub can
// never accidentally intercept the static server serving the app itself.
const API = 'http://127.0.0.1:8080';

/**
 * Installs the stubs and the session, then navigates. `overrides` replaces
 * individual fixtures (`matchDetails` keyed by match id, `registrations`, …).
 *
 * Returns the list of API requests that hit no stub — asserted empty by
 * `gotoApp` below, because an unstubbed call renders an error state into the
 * screenshot instead of failing the test.
 */
async function stubApi(page, overrides = {}) {
  const unstubbed = [];

  // Registered FIRST on purpose: Playwright checks route handlers in reverse
  // order of registration, so this catch-all is only reached when none of the
  // specific handlers below matched.
  await page.route(`${API}/**`, async (route) => {
    unstubbed.push(`${route.request().method()} ${route.request().url()}`);
    await route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"not stubbed"}' });
  });

  await page.route(`${API}/groups/me`, (route) => json(route, overrides.groups ?? data.groups));
  // Profile.vue's own cross-group stats call. Deliberately carries no
  // group_id — GET /players/me/stats is cross-group by design.
  await page.route(`${API}/players/me/stats*`, (route) => json(route, overrides.playerStats ?? data.playerStats));
  // Hidden features. Profile.vue reads this for its cards; App.vue reads it
  // once per session for the design-v2 class. None by default, like most
  // players — so the redesign class stays off and today's screenshots hold.
  await page.route(`${API}/players/me/features`, (route) => json(route, { features: overrides.features ?? data.features }));
  // The Notifications card reads these only once push is on for the device.
  // The off-state screenshot never calls them; stubbing them keeps an on-state
  // page from falling through to the catch-all. A PUT merges into the copy
  // this page is holding, so a test can flip one switch and read it back.
  let notificationPreferences = {
    ...(overrides.notificationPreferences ?? data.notificationPreferences),
  };
  await page.route(`${API}/players/me/notification-preferences`, async (route) => {
    const method = route.request().method();
    if (method === 'PUT') {
      const body = JSON.parse(route.request().postData() || '{}');
      notificationPreferences = { ...notificationPreferences, ...(body.preferences || {}) };
    } else if (method !== 'GET') {
      unstubbed.push(`${method} ${route.request().url()}`);
      return route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"unexpected write"}' });
    }
    return json(route, { preferences: notificationPreferences });
  });
  await page.route(`${API}/push/public-key`, (route) => json(route, { public_key: 'test-public-key' }));
  await page.route(`${API}/players/me/push-subscription`, (route) => {
    const method = route.request().method();
    if (method === 'POST') return json(route, { subscribed: true });
    if (method === 'DELETE') return json(route, { unsubscribed: true });
    unstubbed.push(`${method} ${route.request().url()}`);
    return route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"unexpected write"}' });
  });
  await page.route(`${API}/groups/*/players`, (route) => json(route, overrides.groupMembers ?? data.groupMembers));
  // Only an admin viewing a scheduled match fetches this (MatchDetails.vue,
  // for "Share on WhatsApp") — stubbed unconditionally anyway since every
  // match-details scenario shares this one route.
  await page.route(`${API}/groups/*/invite-code`, (route) => json(route, { invite_code: overrides.inviteCode ?? data.inviteCode }));
  await page.route(`${API}/standings/seasons*`, (route) => json(route, overrides.seasons ?? data.seasons));
  await page.route(`${API}/standings/points*`, (route) => json(route, overrides.pointsStandings ?? data.pointsStandings));
  await page.route(`${API}/standings/scorers*`, (route) => json(route, overrides.topScorers ?? data.topScorers));
  await page.route(`${API}/standings/motm*`, (route) => json(route, overrides.motmStandings ?? data.motmStandings));
  await page.route(`${API}/standings/predictions*`, (route) => json(route, overrides.predictionStandings ?? []));
  await page.route(`${API}/matches/details*`, (route) => json(route, overrides.matches ?? data.matches));
  await page.route(`${API}/matches/*/predictions`, (route) => {
    if (route.request().method() !== 'GET') {
      unstubbed.push(`${route.request().method()} ${route.request().url()}`);
      return route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"unexpected write"}' });
    }
    const id = new URL(route.request().url()).pathname.split('/')[2];
    const summary = (overrides.predictionSummaries ?? {})[id];
    return json(route, summary ?? { Entries: [], Count: 0, Revealed: false });
  });

  // Keyed by id so one test can put a match in a state another test doesn't
  // have — the same URL shape serves every match.
  const byID = overrides.matchDetails ?? {};
  await page.route(`${API}/matches/*/details*`, (route) => {
    const id = new URL(route.request().url()).pathname.split('/')[2];
    const match = byID[id] ?? (overrides.matches ?? data.matches).find((m) => m.ID === id);
    if (!match) {
      return route.fulfill({ status: 404, contentType: 'application/json', body: '{"error":"match not found"}' });
    }
    return json(route, match);
  });

  await page.route(`${API}/matches/*/registrations`, (route) => {
    // Only GET is exercised by these tests — they capture states, they don't
    // mutate. A POST/DELETE getting here means a click landed somewhere
    // unintended, and answering 500 makes that visible rather than plausible.
    if (route.request().method() !== 'GET') {
      unstubbed.push(`${route.request().method()} ${route.request().url()}`);
      return route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"unexpected write"}' });
    }
    return json(route, overrides.registrations ?? data.registrations);
  });

  // Fetched for any match with a composed roster (scheduled or not — see
  // CLAUDE.md's Man of the Match section), same "GET only" simplification as
  // registrations above.
  await page.route(`${API}/matches/*/votes`, (route) => {
    if (route.request().method() !== 'GET') {
      unstubbed.push(`${route.request().method()} ${route.request().url()}`);
      return route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"unexpected write"}' });
    }
    return json(route, overrides.motmVotes ?? data.motmVotes);
  });

  return unstubbed;
}

/**
 * The one entry point the specs use: stub, seed the session, freeze the clock,
 * navigate, and wait until the page has settled enough to be photographed.
 */
async function gotoApp(page, path, options = {}) {
  const {
    theme = 'light',
    playerID = data.CURRENT_PLAYER_ID,
    activeGroupID = data.GROUP_ID,
    ...overrides
  } = options;

  const unstubbed = await stubApi(page, overrides);

  // Before goto, so the app's own scripts never observe the real clock. A fixed
  // time rather than fake timers: transitions and timeouts keep working, which
  // is what lets the page finish rendering at all.
  await page.clock.setFixedTime(data.FIXED_NOW);

  await page.addInitScript(({ token, tokenKey, groupKey, groupID, themeKey, themeValue }) => {
    localStorage.setItem(tokenKey, token);
    localStorage.setItem(groupKey, groupID);
    localStorage.setItem(themeKey, themeValue);
  }, {
    token: fakeJWT(playerID),
    tokenKey: TOKEN_KEY,
    groupKey: ACTIVE_GROUP_KEY,
    groupID: activeGroupID,
    themeKey: THEME_KEY,
    themeValue: theme,
  });

  await page.goto(path);

  // The horizontal match carousel is `scroll-behavior: smooth` (global-styles),
  // so Playwright's own auto-scroll-into-view before a `.click()` (e.g.
  // selecting a card off-screen) animates instead of jumping — and unlike CSS
  // transitions/animations, that isn't neutralized by toHaveScreenshot's own
  // animation-disabling, which only applies once the screenshot call itself
  // starts, well after the click already happened. Without this, a test that
  // clicks a card can capture mid-scroll, blurred text — a real, if small and
  // intermittent, diff. Forcing `auto` here, once, up front, is cheaper and
  // more robust than adding a manual settle-wait to every test that clicks
  // anything in the carousel.
  //
  // -webkit-font-smoothing: antialiased forces grayscale text rendering
  // instead of Chromium's default subpixel (LCD) rendering. It's here for an
  // unrelated reason: subpixel rendering is what makes text inside a
  // scrolled carousel card rasterize slightly differently between otherwise
  // identical runs (see pinCarouselScroll's own comment) — grayscale AA is
  // far less sensitive to that sub-pixel positioning. It measurably reduced
  // the resulting flake rate but did not eliminate it (see
  // playwright.config.js's maxDiffPixels).
  await page.addStyleTag({ content: '* { scroll-behavior: auto !important; -webkit-font-smoothing: antialiased !important; }' });

  // The app renders a spinner while it resolves the active group; every page
  // under test is past that once the navbar and the main content are both up.
  // `.main-content` rather than anything in the top bar: on a phone-width
  // viewport (the @mobile project) App.vue hides the whole top bar on an
  // authenticated page, so no locator inside it would ever become visible
  // there. The main content renders on every viewport and route.
  await expect(page.locator('.main-content')).toBeVisible();
  await expect(page.locator('.loading-spinner')).toHaveCount(0);

  return { unstubbed };
}

/** Fails with the offending URLs rather than a bare count, so a missing stub is self-diagnosing. */
function expectEverythingStubbed(unstubbed) {
  expect(unstubbed, `API calls reached no stub:\n  ${unstubbed.join('\n  ')}`).toEqual([]);
}

/**
 * Fails the test if the page itself scrolls horizontally — a real bug found
 * on the @mobile project: four equal-flex sub-tab buttons (Matches/Points/
 * Scorers/MOTM) defaulted to min-width: auto, so on a narrow phone the row
 * didn't shrink, it overflowed, and the MOTM tab stuck out past the right
 * edge needing a sideways scroll to reach. A screenshot diff alone wouldn't
 * necessarily catch a regression here — a *narrower* overflow could still
 * look plausible at a glance — so this checks the actual layout invariant
 * ("nothing should ever force this page to scroll sideways") directly,
 * independent of any one baseline.
 */
async function expectNoHorizontalOverflow(page) {
  const { scrollWidth, clientWidth } = await page.evaluate(() => ({
    scrollWidth: document.documentElement.scrollWidth,
    clientWidth: document.documentElement.clientWidth,
  }));
  expect(scrollWidth, 'page.documentElement.scrollWidth should never exceed clientWidth').toBeLessThanOrEqual(clientWidth);
}

/**
 * Waits for a horizontal carousel's own scroll position to stop moving, so the
 * screenshot that follows can't catch it mid-scroll.
 *
 * Playwright auto-scrolls a click target into view before clicking, and
 * `gotoApp` already forces `scroll-behavior: auto` globally so that scroll
 * jumps rather than animates — but "jumped" is not "settled": the compositor
 * can be a frame behind under the parallel load `run-in-docker.sh` and CI both
 * use. Two consecutive equal reads of `scrollLeft` are what prove it stopped,
 * whatever the reason it hadn't.
 *
 * `carouselSelector` is the scrolling container (`.matches-bar` on the home
 * page, `.groups-bar` on the profile), not the card — the container's
 * scrollLeft is what has to be stable.
 */
async function waitForCarouselToSettle(page, carouselSelector) {
  const scrollLeft = () => page.locator(carouselSelector).evaluate((el) => el.scrollLeft);
  let previous = await scrollLeft();
  for (let i = 0; i < 20; i += 1) {
    await page.waitForTimeout(50);
    const current = await scrollLeft();
    if (current === previous) return;
    previous = current;
  }
}

/**
 * Parks a horizontal carousel at one of its two end stops and waits for that
 * to hold, so a baseline covering the carousel cannot depend on where a
 * click-time auto-scroll happened to leave it.
 *
 * This is stronger than waiting for the scroll to settle, and the difference
 * matters: a CI run failed `profile-roster-member-mobile.png` by ~610 pixels,
 * all of them inside the carousel, while Playwright reported "captured a
 * stable screenshot" — i.e. the scroll had finished, it had simply finished
 * somewhere a pixel or two away from where the baseline was recorded. No
 * amount of waiting fixes a resting position that isn't reproducible; an end
 * stop is exact (0, or the layout-derived maximum the browser clamps to) and
 * therefore is.
 *
 * Deliberately *not* `scrollIntoView` on the card: that lands the carousel
 * somewhere different again from Playwright's own auto-scroll, which is what
 * recorded the baselines — trying it failed the profile baseline 6 runs out
 * of 6.
 *
 * Every visual test that clicks a card in a carousel the screenshot then
 * covers goes through this. `home.spec.js`'s two were briefly an exception,
 * kept on an instant `scrollIntoView` plus a settle-wait because their
 * baselines had been recorded that way; CI then failed one of them by 652
 * pixels on a branch whose entire diff was a workflow comment, which settled
 * the question. Neither carousel needed its baselines regenerated to adopt
 * this, so the end stop is where the auto-scroll was aiming in both cases.
 *
 * Pinning this way still does not fully eliminate the flake, and a direct CI
 * diff image is what finally showed why: every differing pixel sits inside
 * the text of whichever card the carousel is scrolled to — nowhere else on
 * the page — even though scrollLeft/scrollWidth/clientWidth read back
 * bit-identical across failing and passing runs (confirmed by instrumenting
 * this function directly). That rules out the scroll position itself; what's
 * left is Chromium's own sub-pixel text rendering for *scrolled* content,
 * which can rasterize a hair differently between otherwise-identical runs
 * independent of any value this function or the test can read. Forcing
 * grayscale antialiasing (`gotoApp`'s own style tag) measurably reduced how
 * often this happens but did not remove it, which is why `maxDiffPixels` in
 * playwright.config.js carries headroom for it specifically rather than the
 * flake being chased further here.
 */
async function pinCarouselScroll(page, carouselSelector, position) {
  const pin = () => page.locator(carouselSelector).evaluate((el, where) => {
    // Force a layout read so scrollWidth/clientWidth aren't a frame behind
    // under the parallel load CI uses. Assigning scrollWidth itself (and
    // relying on the browser to clamp) is what the first version did; the
    // explicit max is the same number, just not dependent on clamp rounding
    // a too-large value onto a fractional scrollLeft.
    void el.offsetWidth;
    const max = Math.max(0, el.scrollWidth - el.clientWidth);
    el.scrollLeft = where === 'end' ? max : 0;
  }, position);

  await pin();
  await waitForCarouselToSettle(page, carouselSelector);
  // A sibling expanding after the first pin (the inline sign-up panel
  // finishing its fetch, a flex reflow) can change the max by a pixel or
  // two. Pinning again against the *settled* layout is what makes the end
  // stop the resting position of the screenshot, not of the moment we
  // first assigned. The 650-pixel `matches-tab-signup-inline.png` failure
  // on a feature that never touched this page is the reproduction: pin
  // had already run, Playwright reported a stable screenshot, and every
  // differing pixel was inside the selected card.
  await pin();
  await waitForCarouselToSettle(page, carouselSelector);
}

module.exports = {
  gotoApp,
  expectEverythingStubbed,
  expectNoHorizontalOverflow,
  waitForCarouselToSettle,
  pinCarouselScroll,
  data,
};
