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
  await page.route(`${API}/groups/*/players`, (route) => json(route, overrides.groupMembers ?? data.groupMembers));
  await page.route(`${API}/standings/seasons*`, (route) => json(route, overrides.seasons ?? data.seasons));
  await page.route(`${API}/standings/points*`, (route) => json(route, overrides.pointsStandings ?? data.pointsStandings));
  await page.route(`${API}/standings/scorers*`, (route) => json(route, overrides.topScorers ?? data.topScorers));
  await page.route(`${API}/standings/motm*`, (route) => json(route, overrides.motmStandings ?? data.motmStandings));
  await page.route(`${API}/matches/details*`, (route) => json(route, overrides.matches ?? data.matches));

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
  await page.addStyleTag({ content: '* { scroll-behavior: auto !important; }' });

  // The app renders a spinner while it resolves the active group; every page
  // under test is past that once the navbar and the main content are both up.
  await expect(page.locator('.nav-menu')).toBeVisible();
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

module.exports = { gotoApp, expectEverythingStubbed, expectNoHorizontalOverflow, data };
