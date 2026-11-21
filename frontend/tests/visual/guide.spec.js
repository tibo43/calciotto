const { test, expect } = require('@playwright/test');
const { gotoApp, expectEverythingStubbed, expectNoHorizontalOverflow } = require('./fixtures/app');

const guideOptions = { features: ['design-v2'] };

async function expectStep(guide, title, theme, image) {
  await expect(guide.getByRole('heading', { name: title, exact: true })).toBeVisible();
  const src = `/guide/${theme}/${image}.jpg`;
  const preview = guide.locator('img');
  await expect(preview).toHaveAttribute('src', src);
  await expect.poll(() => preview.evaluate((img, url) =>
    img.complete && img.currentSrc.endsWith(url) && img.naturalWidth === 390, src)).toBe(true);
}

async function reachLastStep(guide, theme) {
  for (const [title, image] of [
    ['Matches', 'matches'], ['Rankings', 'rankings'], ['Predictions', 'predictions'], ['Profile', 'profile'],
  ]) {
    await guide.getByRole('button', { name: 'Next', exact: true }).click();
    await expectStep(guide, title, theme, image);
  }
}

test('first-run guide saves completion before opening the requested page @mobile', async ({ page }) => {
  await page.emulateMedia({ colorScheme: 'dark' });
  const destination = '/rankings?tab=scorers';
  const { unstubbed } = await gotoApp(page, destination, { ...guideOptions, guideCompleted: false, theme: 'light' });
  const guide = page.locator('.player-guide');
  await expectStep(guide, 'Welcome to Agoran', 'light', 'welcome');
  await expect(page.locator('.main-content')).toHaveCount(0);
  await expect(guide.getByRole('button', { name: 'Close guide' })).toHaveCount(0);
  await page.keyboard.press('Escape');
  await expect(guide).toBeVisible();
  await expect(page).toHaveScreenshot('guide-required-light.png');
  await expectNoHorizontalOverflow(page);

  await reachLastStep(guide, 'light');
  await guide.getByRole('button', { name: 'Back', exact: true }).click();
  await expectStep(guide, 'Predictions', 'light', 'predictions');
  await guide.getByRole('button', { name: 'Next', exact: true }).click();

  await page.route('**/players/me/guide', (route) => route.fulfill({ status: 503 }), { times: 1 });
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

test('the profile reopens the guide in the system theme without a saved preference', async ({ page }) => {
  await page.emulateMedia({ colorScheme: 'dark' });
  const { unstubbed } = await gotoApp(page, '/profile', { ...guideOptions, theme: null });
  await page.getByRole('button', { name: 'Replay app guide' }).click();
  const guide = page.locator('.player-guide');
  await expectStep(guide, 'Welcome to Agoran', 'dark', 'welcome');
  await expect(page).toHaveScreenshot('guide-replay-dark.png');
  await reachLastStep(guide, 'dark');
  await guide.getByRole('button', { name: 'Close guide', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Replay app guide' })).toBeVisible();
  await expect(page).toHaveURL(/\/profile$/);
  expectEverythingStubbed(unstubbed);
});
