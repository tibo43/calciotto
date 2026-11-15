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
