// Creating a match with design v2 on: the home page's floating "New match"
// button (admins) and the page it opens.

const { test, expect } = require('@playwright/test');
const data = require('./fixtures/data');
const { gotoApp, expectEverythingStubbed, expectNoHorizontalOverflow } = require('./fixtures/app');

const V2 = { features: ['design-v2'] };
const NOTHING_SCHEDULED = { ...V2, matches: data.matches.filter((match) => match.ID !== data.scheduledOpenMatch.ID) };

test.describe('design v2 new match', () => {
  test('home, nothing scheduled, as an admin @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', NOTHING_SCHEDULED);
    await expect(page.getByRole('link', { name: 'New match' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home-new-match-mobile.png');
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('new match page', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/matches/new', V2);
    await expect(page.getByRole('button', { name: 'Create match' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-new-match.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('new match page, dark', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/matches/new', { ...V2, theme: 'dark' });
    await expect(page.getByRole('button', { name: 'Create match' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-new-match-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('new match page, a missing field @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/matches/new', V2);
    await page.getByRole('button', { name: 'Create match' }).click();
    await expect(page.getByText('Pick the match day.')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-new-match-error-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });
});
