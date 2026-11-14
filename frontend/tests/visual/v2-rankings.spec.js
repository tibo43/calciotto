// Rankings with the design-v2 feature on. /classements still redirects home
// when the cache is unset, which is what a v1 player hits.

const { test, expect } = require('@playwright/test');
const { gotoApp, expectEverythingStubbed, expectNoHorizontalOverflow } = require('./fixtures/app');

const V2 = { features: ['design-v2'] };

async function openRankings(page, path, options = {}) {
  await page.addInitScript(() => {
    localStorage.setItem('calciotto-design-v2', '1');
  });
  return gotoApp(page, path, { ...V2, ...options });
}

test.describe('design v2 rankings', () => {
  test('points', async ({ page }) => {
    const { unstubbed } = await openRankings(page, '/classements');
    await expect(page.locator('table.stand')).toBeVisible();
    await expect(page.locator('table.stand').getByText('(left the group)')).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'OG' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-rankings-points.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('points, dark', async ({ page }) => {
    const { unstubbed } = await openRankings(page, '/classements', { theme: 'dark' });
    await expect(page.locator('.design-v2.dark-mode')).toBeVisible();
    await expect(page.locator('table.stand')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-rankings-points-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('points @mobile', async ({ page }) => {
    const { unstubbed } = await openRankings(page, '/classements');
    await expect(page.locator('table.stand')).toBeHidden();
    await expect(page.locator('.pcard').first()).toBeVisible();
    await expect(page.getByText('P 9 · W 6 · D 2 · L 1 · GF 14')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-rankings-points-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('scorers', async ({ page }) => {
    const { unstubbed } = await openRankings(page, '/classements?tab=scorers');
    await expect(page.locator('table.stand')).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'Goals' })).toBeVisible();
    await expect(page.getByText('9 matches').first()).toBeVisible();
    await expect(page).toHaveScreenshot('v2-rankings-scorers.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('scorers @mobile', async ({ page }) => {
    const { unstubbed } = await openRankings(page, '/classements?tab=scorers');
    await expect(page.locator('table.stand')).toBeHidden();
    await expect(page.getByText('14 goals · 9 matches')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-rankings-scorers-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('motm', async ({ page }) => {
    const { unstubbed } = await openRankings(page, '/classements?tab=motm');
    await expect(page.locator('table.stand')).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'Awards' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-rankings-motm.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('motm, dark', async ({ page }) => {
    const { unstubbed } = await openRankings(page, '/classements?tab=motm', { theme: 'dark' });
    await expect(page.locator('table.stand')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-rankings-motm-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('motm @mobile', async ({ page }) => {
    const { unstubbed } = await openRankings(page, '/classements?tab=motm');
    await expect(page.locator('table.stand')).toBeHidden();
    await expect(page.getByText('5 awards')).toBeVisible();
    await expect(page.getByText('1 award')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-rankings-motm-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });
});
