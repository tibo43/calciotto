const { test, expect } = require('@playwright/test');
const { gotoApp, expectEverythingStubbed, expectNoHorizontalOverflow } = require('./fixtures/app');

const V2 = { features: ['design-v2'] };

async function expectStep(page, title) {
  const guide = page.locator('.player-guide');
  await expect(guide.getByRole('heading', { name: title, exact: true })).toBeVisible();
  await expect.poll(() => guide.locator('img').evaluate((img) => img.naturalWidth)).toBe(390);
  return guide;
}

test('first-run guide saves completion before opening the requested page @mobile', async ({ page }) => {
  const destination = '/rankings?tab=scorers';
  const { unstubbed } = await gotoApp(page, destination, { ...V2, v2GuideCompleted: false });
  const guide = await expectStep(page, 'Welcome to Calciotto');
  await expect(page.locator('.main-content')).toHaveCount(0);
  await expect(guide.getByRole('button', { name: 'Close guide' })).toHaveCount(0);
  await page.keyboard.press('Escape');
  await expect(guide).toBeVisible();
  await expect(page).toHaveScreenshot('v2-guide-required-mobile.png');
  await expectNoHorizontalOverflow(page);

  for (const title of ['Matches', 'Rankings', 'Predictions', 'Profile']) {
    await guide.getByRole('button', { name: 'Next', exact: true }).click();
    await expectStep(page, title);
  }
  await guide.getByRole('button', { name: 'Back', exact: true }).click();
  await expectStep(page, 'Predictions');
  await guide.getByRole('button', { name: 'Next', exact: true }).click();

  await page.route('**/players/me/v2-guide', (route) => route.fulfill({ status: 503 }), { times: 1 });
  await guide.getByRole('button', { name: 'Get started', exact: true }).click();
  await expect(guide.getByRole('alert')).toHaveText('Could not save your progress. Try again.');
  await expect(page.locator('.main-content')).toHaveCount(0);

  await guide.getByRole('button', { name: 'Get started', exact: true }).click();
  await expect(page.locator('.main-content')).toBeVisible();
  await expect(page).toHaveURL(/\/rankings\?tab=scorers$/);
  await page.reload();
  await expect(page.locator('.main-content')).toBeVisible();
  await expect(guide).toHaveCount(0);
  expectEverythingStubbed(unstubbed);
});

test('the profile reopens the optional guide after completion', async ({ page }) => {
  const { unstubbed } = await gotoApp(page, '/profile', V2);
  await page.getByRole('button', { name: 'Replay app guide' }).click();
  const guide = await expectStep(page, 'Welcome to Calciotto');
  await expect(page).toHaveScreenshot('v2-guide-replay.png');
  await guide.getByRole('button', { name: 'Close guide', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Replay app guide' })).toBeVisible();
  await expect(page).toHaveURL(/\/profile$/);
  expectEverythingStubbed(unstubbed);
});
