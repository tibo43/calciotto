// Admin editing with the design-v2 feature on. The same route renders
// MatchDetails.vue when the feature is off, which is what match-details.spec.js
// still photographs.

const { test, expect } = require('@playwright/test');
const data = require('./fixtures/data');
const { gotoApp, expectEverythingStubbed, expectNoHorizontalOverflow } = require('./fixtures/app');

const ID = data.scheduledClosedMatch.ID;
const EDIT = `/matches/${ID}/edit`;

const registrations = data.registrations.slice(0, 13).map((entry, index) => ({
  ...entry,
  Position: index + 1,
  IsWaiting: index >= 11,
}));

function closedMatch(extra = {}) {
  return {
    ...data.scheduledClosedMatch,
    MaxPlayers: 11,
    Teams: data.scheduledClosedMatch.Teams.map((team) => ({ ...team, Players: [], Score: 0 })),
    ...extra,
  };
}

function scoredMatch() {
  const base = closedMatch({ ResultRecordedAt: '2026-09-06T22:00:00+02:00' });
  return {
    ...base,
    Teams: base.Teams.map((team, index) => ({
      ...team,
      Players: data.playedMatch.Teams[index].Players.map((player) => ({
        ...player,
        OwnGoals: index === 0 && player.Name === 'Giovanni Esposito' ? 2 : (player.OwnGoals || 0),
      })),
    })),
  };
}

async function openEdit(page, path, options = {}) {
  await page.addInitScript(() => {
    localStorage.setItem('calciotto-design-v2', '1');
  });
  const { match, ...rest } = options;
  return gotoApp(page, path, {
    features: ['design-v2'],
    registrations,
    matchDetails: { [ID]: match || closedMatch() },
    ...rest,
  });
}

async function splitTeams(page) {
  await page.getByRole('button', { name: 'Split automatically' }).click();
  await expect(page.locator('.size-count')).toHaveText('6 v 5');
}

// Moves the first player of the second team onto the first, then picks them
// again. The assign sheet is hidden at desktop width (the design only shows
// it on a phone), so that click is forced there.
async function pickUneven(page) {
  await splitTeams(page);
  const row = page.locator('.compose-col').nth(2).locator('.p-row').first();
  const name = (await row.locator('.nm').innerText()).trim();
  await row.click();
  // The sheet is display:none on a wide screen, so a pointer click has no box.
  await page.locator('.sheet button', { hasText: 'Les Bleus' }).first().evaluate((button) => button.click());
  await page.locator('.compose-col').nth(1).locator('.p-row', { hasText: name }).click();
  await expect(page.locator('.size-count')).toHaveText('7 v 4');
}

test.describe('design v2 admin editing', () => {
  test('sign-ups', async ({ page }) => {
    const { unstubbed } = await openEdit(page, EDIT);
    await expect(page.getByRole('button', { name: 'Reopen sign-ups' })).toBeVisible();
    await expect(page.getByText('Share on WhatsApp')).toBeVisible();
    await expect(page.locator('.tog.is-on').first()).toHaveText('Confirmed');
    await expect(page.getByRole('button', { name: 'Promote' })).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-edit-signups.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('sign-ups, dark', async ({ page }) => {
    const { unstubbed } = await openEdit(page, EDIT, { theme: 'dark' });
    await expect(page.locator('.design-v2.dark-mode')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Reopen sign-ups' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-signups-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('sign-ups @mobile', async ({ page }) => {
    const { unstubbed } = await openEdit(page, EDIT);
    await expect(page.getByRole('button', { name: 'Reopen sign-ups' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-signups-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('teams, nobody placed', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=teams`);
    await expect(page.locator('.size-count')).toHaveText('0 v 0');
    await expect(page.locator('.compose-col input')).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Promote' })).toHaveCount(0);
    await expect(page.getByRole('heading', { name: 'Les Bleus' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-teams.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('teams, split automatically', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=teams`);
    await splitTeams(page);
    await expect(page.getByText('The teams are uneven')).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-edit-teams-auto.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('teams, a player picked', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=teams`);
    await pickUneven(page);
    await expect(page.getByText('The teams are uneven')).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-edit-teams-choice.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('teams, a player picked @mobile', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=teams`);
    await pickUneven(page);
    await expect(page.locator('.sheet')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Reserve' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-teams-choice-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('score', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=score`, { match: scoredMatch() });
    await expect(page.getByRole('button', { name: '+ Own goal' }).first()).toBeVisible();
    await expect(page.locator('.og-plain', { hasText: '2 OG' })).toBeVisible();
    await expect(page.locator('input[type="number"]')).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-edit-score.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('score, dark', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=score`, { theme: 'dark', match: scoredMatch() });
    await expect(page.locator('.og-plain', { hasText: '2 OG' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-score-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('score @mobile', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=score`, { match: scoredMatch() });
    await expect(page.locator('.og-plain', { hasText: '2 OG' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-score-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('own goal picker', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=score`, { match: scoredMatch() });
    await page.getByRole('button', { name: '+ Own goal' }).nth(1).click();
    await expect(page.locator('.csc-pop')).toBeVisible();
    await expect(page.locator('.csc-pop .csc-pick', { hasText: 'Giovanni Esposito' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-own-goal.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('own goal picker @mobile', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=score`, { match: scoredMatch() });
    await page.getByRole('button', { name: '+ Own goal' }).nth(1).click();
    await expect(page.locator('.csc-sheet')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-own-goal-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });
});
