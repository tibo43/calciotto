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
