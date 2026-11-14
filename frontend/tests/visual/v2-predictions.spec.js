// Predictions with the design-v2 feature on. The flag stays off in every
// other spec, so those baselines keep photographing today's screens.

const { test, expect } = require('@playwright/test');
const data = require('./fixtures/data');
const { gotoApp, expectEverythingStubbed, expectNoHorizontalOverflow } = require('./fixtures/app');

const V2 = {
  features: ['design-v2'],
  matches: data.predictionMatches,
  predictionSummaries: data.predictionSummaries,
  predictionStandings: data.predictionStandings,
};

async function openPredictions(page, path, options = {}) {
  await page.addInitScript(() => {
    localStorage.setItem('calciotto-design-v2', '1');
  });
  return gotoApp(page, path, { ...V2, ...options });
}

test.describe('design v2 predictions', () => {
  test('matches', async ({ page }) => {
    const { unstubbed } = await openPredictions(page, '/predictions');
    await expect(page.locator('.pred-state.ok').first()).toHaveText(/your pick/i);
    await expect(page.locator('.pred-score').first()).toHaveText('4–2');
    const score = await page.locator('.pred-score').first().textContent();
    expect(score).toBe(' 4–2 ');
    await expect(page).toHaveScreenshot('v2-predictions-matches.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('matches, dark', async ({ page }) => {
    const { unstubbed } = await openPredictions(page, '/predictions', { theme: 'dark' });
    await expect(page.locator('.design-v2.dark-mode')).toBeVisible();
    await expect(page.locator('.pred-state.ok').first()).toHaveText(/your pick/i);
    await expect(page).toHaveScreenshot('v2-predictions-matches-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('matches @mobile', async ({ page }) => {
    const { unstubbed } = await openPredictions(page, '/predictions');
    await expect(page.locator('.pred-state.ok').first()).toBeVisible();
    const widths = await page.locator('label.ctx').evaluateAll((els) => els.map((el) => el.getBoundingClientRect().width));
    expect(widths[0]).toBeGreaterThan(widths[1]);
    await expect(page).toHaveScreenshot('v2-predictions-matches-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('leaderboard', async ({ page }) => {
    const { unstubbed } = await openPredictions(page, '/predictions?tab=leaderboard');
    await expect(page.locator('table.stand')).toBeVisible();
    await expect(page.locator('table.stand .form-k:visible')).toHaveCount(0);
    await expect(page.locator('table.stand .form-cell.is-pad')).toHaveCount(2);
    const pad = await page.locator('table.stand tr.is-me td.rk').evaluate((el) => getComputedStyle(el).paddingLeft);
    expect(pad).toBe('8px');
    await expect(page).toHaveScreenshot('v2-predictions-leaderboard.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('leaderboard, dark', async ({ page }) => {
    const { unstubbed } = await openPredictions(page, '/predictions?tab=leaderboard', { theme: 'dark' });
    await expect(page.locator('table.stand')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-predictions-leaderboard-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('leaderboard @mobile', async ({ page }) => {
    const { unstubbed } = await openPredictions(page, '/predictions?tab=leaderboard');
    await expect(page.locator('table.stand')).toBeHidden();
    await expect(page.locator('.pcard .form-k').first()).toBeVisible();
    await expect(page.locator('.pcard .form-cell.is-pad')).toHaveCount(2);
    await expect(page).toHaveScreenshot('v2-predictions-leaderboard-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });
});
