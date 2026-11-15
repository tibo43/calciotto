// Home and match detail with the design-v2 feature on. The flag stays off in
// every other spec, so those baselines keep photographing today's screens.

const { test, expect } = require('@playwright/test');
const data = require('./fixtures/data');
const { gotoApp, expectEverythingStubbed, expectNoHorizontalOverflow } = require('./fixtures/app');

const V2 = { features: ['design-v2'] };
const OPEN = data.scheduledOpenMatch.ID;
const PLAYED = data.playedMatch.ID;

test.describe('design v2 home and match', () => {
  test('home, signed up', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', V2);
    await expect(page.getByText("You're signed up")).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('home, signed up, dark', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', { ...V2, theme: 'dark' });
    await expect(page.locator('.design-v2.dark-mode')).toBeVisible();
    await expect(page.getByText("You're signed up")).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('home, signed up @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', V2);
    await expect(page.getByText("You're signed up")).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('home, signed up, dark @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', { ...V2, theme: 'dark' });
    await expect(page.getByText("You're signed up")).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home-dark-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('home, withdraw confirmation', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', V2);
    await page.getByRole('button', { name: 'Withdraw', exact: true }).click();
    await expect(page.getByRole('dialog').getByText('Withdraw from this match?')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home-withdraw.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('upcoming match, waiting list', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${OPEN}`, V2);
    await expect(page.getByRole('heading', { name: 'Waiting list' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Manage match' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-upcoming.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('upcoming match, waiting list, dark', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${OPEN}`, { ...V2, theme: 'dark' });
    await expect(page.getByRole('heading', { name: 'Waiting list' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-upcoming-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('upcoming match, waiting list @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${OPEN}`, V2);
    await expect(page.getByRole('heading', { name: 'Waiting list' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-upcoming-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('played match, vote closed', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${PLAYED}`, V2);
    await expect(page.getByText('MOTM vote closed', { exact: true })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-played.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('played match, vote closed, dark', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${PLAYED}`, { ...V2, theme: 'dark' });
    await expect(page.getByText('MOTM vote closed', { exact: true })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-played-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });
});

// The same match once the admin has set the teams, with the caller on one of
// them: their side instead of the places, both line-ups, and the prediction
// bar pinned above the tab bar.
const ME = '33333333-3333-4333-8333-333333333333';
const member = (n, name) => ({ ID: n === 'me' ? ME : `bbbbbbbb-0000-4000-8000-0000000000${n}`, Name: name, GoalNumber: 0 });
const teamsSet = {
  ...data.scheduledOpenMatch,
  RegistrationsClosedAt: '2026-09-04T09:00:00+02:00',
  Teams: [
    { ...data.playedMatch.Teams[0], Score: 0, Players: [member('01', 'Marco Rossi'), member('me', 'Thibaut Fabre'), member('02', 'Luca Bianchi'), member('04', 'Giovanni Esposito')] },
    { ...data.playedMatch.Teams[1], Score: 0, Players: [member('05', 'Andrea Conti'), member('06', 'Matteo Ricci'), member('07', 'Federico Greco'), member('08', 'Alessandro Moretti')] },
  ],
};
const TEAMS_SET = {
  ...V2,
  matches: data.matches.map((match) => (match.ID === teamsSet.ID ? teamsSet : match)),
  predictionSummaries: { [teamsSet.ID]: { Entries: [], Count: 5, Revealed: false, Odds: { TeamAOdds: 1.85, DrawOdds: 3.4, TeamBOdds: 2.1 } } },
};

test.describe('design v2 once the teams are set', () => {
  test('upcoming match, teams set, prediction bar @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${OPEN}`, TEAMS_SET);
    await expect(page.getByText('Your prediction')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-teams-set-mobile.png');
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('upcoming match, teams set, prediction bar', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${OPEN}`, TEAMS_SET);
    await expect(page.getByText('Your prediction')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-teams-set.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('home, teams set @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', TEAMS_SET);
    await expect(page.getByText('You play for')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home-teams-set-mobile.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  // Every place taken and the caller not signed up: the waiting list.
  test('home, list full @mobile', async ({ page }) => {
    const notMe = data.registrations.map((entry) => (entry.PlayerID === ME ? { ...entry, PlayerID: 'eeeeeeee-0000-4000-8000-000000000001' } : entry));
    const { unstubbed } = await gotoApp(page, '/', { ...V2, registrations: notMe });
    await expect(page.getByRole('button', { name: 'Join the waiting list' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home-full-mobile.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });
});

// A match is over as soon as its score is recorded, here on its own day
// (kick-off 08:00, the clock at 10:00): no longer the next match, and its
// page shows the result with nothing about sign-ups.
const recorded = {
  ...data.playedMatch, ID: 'cccccccc-0000-4000-8000-0000000000f1', Date: '2026-09-04', ScheduledAt: '2026-09-04T08:00:00+02:00',
  RegistrationOpensAt: '2026-09-01T18:00:00+02:00', RegistrationsClosedAt: '2026-09-04T07:00:00+02:00',
  ResultRecordedAt: '2026-09-04T09:30:00+02:00', MaxPlayers: 8,
};
const NAV = '.nav-text, .nav-ico';

test.describe('design v2 once the score is recorded', () => {
  test('home, no upcoming match, as an admin', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', { ...V2, matches: [...data.matches.slice(0, 2), recorded] });
    await expect(page.locator('.hero-none')).toContainText('No upcoming match');
    await expect(page.getByText('Sign-ups closed')).toHaveCount(0);
    const first = page.locator('.list .row').first();
    await expect(first).toHaveAttribute('href', `/matches/${recorded.ID}`);
    await expect(first.locator('.score')).toHaveText('4–6');
    await expect(page.getByRole('link', { name: 'New match' })).toBeVisible();
    expectEverythingStubbed(unstubbed);
  });

  test('home, the next scheduled match after it', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', { ...V2, matches: [...data.matches, recorded] });
    await expect(page.locator('.hero .hero-date')).toHaveText('Sun 6 Sep');
    await expect(page.locator('.list .row').first()).toHaveAttribute('href', `/matches/${recorded.ID}`);
    expectEverythingStubbed(unstubbed);
  });

  test('match page, the result and no sign-ups', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${recorded.ID}`, { ...V2, matches: [...data.matches, recorded] });
    await expect(page.locator('.board .board-score')).toHaveText('4–6');
    await expect(page.locator('.motm-line')).toBeVisible();
    await expect(page.locator('main.page')).not.toContainText(/Sign up|Withdraw|Sign-ups|places left/);
    expectEverythingStubbed(unstubbed);
  });
});

// Only a tab's own page highlights its entry in the menu.
test.describe('design v2 menu', () => {
  for (const [name, path] of [['match page', `/matches/${PLAYED}`], ['edit page', `/matches/${data.scheduledClosedMatch.ID}/edit`], ['new match', '/matches/new']]) {
    test(`${name}, no tab highlighted`, async ({ page }) => {
      const closed = data.scheduledClosedMatch;
      const { unstubbed } = await gotoApp(page, path, { ...V2, matchDetails: { [closed.ID]: closed } });
      await expect(page).toHaveURL(new RegExp(`${path}$`));
      await expect(page.locator('main.page')).toBeVisible();
      await expect(page.locator(`${NAV}`).first()).toBeAttached();
      await expect(page.locator(`:is(${NAV}).is-on`)).toHaveCount(0);
      await expect(page.locator(`:is(${NAV})[aria-current="page"]`)).toHaveCount(0);
      expectEverythingStubbed(unstubbed);
    });
  }

  test('home, Home highlighted', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', V2);
    await expect(page.locator('.hero')).toBeVisible();
    const on = page.locator(`:is(${NAV}).is-on`);
    await expect(on.first()).toHaveAttribute('aria-current', 'page');
    for (const label of await on.evaluateAll((items) => items.map((item) => item.getAttribute('aria-label') || item.textContent.trim()))) {
      expect(label).toBe('Home');
    }
    expectEverythingStubbed(unstubbed);
  });
});

test.describe('design v2 home spacing', () => {
  test('16px between the MOTM line and Latest matches @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', { ...V2, matches: [...data.matches.slice(0, 2), recorded] });
    const line = await page.locator('.motm-row').boundingBox();
    const heading = await page.locator('.sec').filter({ hasText: 'Latest matches' }).boundingBox();
    expect(Math.round(heading.y - (line.y + line.height))).toBe(16);
    expectEverythingStubbed(unstubbed);
  });
});
