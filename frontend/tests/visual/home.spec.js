// The unified home page (`/`): one season selector over three sub-tabs. Each
// tab is captured separately because they are three different components
// (MatchesPanel, PointsStandingsTable, ScorersTable) sharing one shell — a diff
// in one of them says which.

const { test, expect } = require('@playwright/test');
const { gotoApp, expectEverythingStubbed } = require('./fixtures/app');

test.describe('home page', () => {
  test('matches tab', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/');
    // The trailing dashed card is admin-only, so waiting on it also proves the
    // role resolved before the screenshot — otherwise this could photograph the
    // pre-role state and pass, then diff once the role arrives a tick earlier.
    await expect(page.locator('.add-match-card')).toBeVisible();
    await expect(page).toHaveScreenshot('matches-tab.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('points tab', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/');
    await page.getByRole('button', { name: 'Points' }).click();
    // One fixture row carries IsMember: false, so the "(left the group)" tag is
    // part of this baseline rather than an untested branch.
    await expect(page.locator('.left-group-tag').first()).toBeVisible();
    await expect(page).toHaveScreenshot('points-tab.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('scorers tab', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/');
    await page.getByRole('button', { name: 'Scorers' }).click();
    await expect(page.locator('.left-group-tag').first()).toBeVisible();
    await expect(page).toHaveScreenshot('scorers-tab.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('matches tab in dark mode', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', { theme: 'dark' });
    await expect(page.locator('.dark-mode')).toBeVisible();
    await expect(page.locator('.add-match-card')).toBeVisible();
    await expect(page).toHaveScreenshot('matches-tab-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  // The narrow layout is a different layout, not a squeezed one: the controls
  // bar stacks and the match list is a horizontal carousel. Tagged so it runs
  // only in the `mobile` project.
  test('matches tab on a narrow viewport @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/');
    await expect(page.locator('.add-match-card')).toBeVisible();
    await expect(page).toHaveScreenshot('matches-tab-mobile.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  // A member (not an admin) must not be offered the create-match affordance.
  // The backend refuses it anyway (requireGroupAdmin), so this pins the UI half
  // of that: what a non-admin is shown.
  test('matches tab as a plain member', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', {
      groups: [{ id: '11111111-1111-4111-8111-111111111111', name: 'Calciotto Milano', role: 'member', is_favorite: true }],
    });
    await expect(page.locator('.match-card-horizontal').first()).toBeVisible();
    await expect(page.locator('.add-match-card')).toHaveCount(0);
    await expect(page).toHaveScreenshot('matches-tab-member.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });
});
