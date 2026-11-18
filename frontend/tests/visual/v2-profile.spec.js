// Profile, group, notifications, group settings, and account with design-v2
// on. The flag stays off in every other spec.

const { test, expect } = require('@playwright/test');
const data = require('./fixtures/data');
const { gotoApp, expectEverythingStubbed, expectNoHorizontalOverflow } = require('./fixtures/app');

const stats = JSON.parse(JSON.stringify(data.playerStats));
stats.Seasons = ['2025-2026', '2026-2027'];
stats.Overall.PredictionPoints = 4.2;
stats.PerGroup[0].PredictionPoints = 5.1;
stats.PerGroup[1].PredictionPoints = -0.9;
stats.Email = 'thibaut@example.com';

// push-notifications too: the Notifications row and page only exist for
// players with that feature, as on today's profile.
const V2 = { features: ['design-v2', 'push-notifications'], playerStats: stats };
const GROUP = `/profile/groups/${data.GROUP_ID}`;
const WEBSITE_LABEL = 'Request to join our group chat';
const groupsWithDetails = data.groups.map((group) => group.id === data.GROUP_ID ? {
  ...group,
  description: 'We play on Thursday evenings. Use the group chat to coordinate arrivals.',
  website_url: 'https://example.test/football-group',
  website_label: WEBSITE_LABEL,
} : group);

async function openProfile(page, path, options = {}) {
  return gotoApp(page, path, { ...V2, ...options });
}

test.describe('design v2 profile', () => {
  test('profile', async ({ page }) => {
    const { unstubbed } = await openProfile(page, '/profile');
    await expect(page.getByRole('heading', { name: 'Thibaut Fabre' })).toBeVisible();
    await expect(page.locator('label.ctx')).toHaveCount(0);
    await expect(page.getByText('2 groups combined')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-profile.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('profile, dark', async ({ page }) => {
    const { unstubbed } = await openProfile(page, '/profile', { theme: 'dark' });
    await expect(page.locator('.design-v2.dark-mode')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Thibaut Fabre' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-profile-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('profile @mobile', async ({ page }) => {
    const { unstubbed } = await openProfile(page, '/profile');
    await expect(page.getByText('2 groups combined')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-profile-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('group', async ({ page }) => {
    const { unstubbed } = await openProfile(page, GROUP, { groups: groupsWithDetails });
    await expect(page.getByRole('link', { name: 'Group settings' })).toBeVisible();
    await expect(page.getByRole('link', { name: WEBSITE_LABEL })).toHaveAttribute('href', 'https://example.test/football-group');
    await expect(page.locator('label.ctx')).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-profile-group.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('group @mobile', async ({ page }) => {
    const { unstubbed } = await openProfile(page, GROUP, { groups: groupsWithDetails });
    await expect(page.getByRole('link', { name: 'Group settings' })).toBeVisible();
    await expect(page.getByRole('link', { name: WEBSITE_LABEL })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-profile-group-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('notifications', async ({ page }) => {
    const { unstubbed } = await openProfile(page, '/profile/notifications');
    await expect(page.getByRole('heading', { name: 'Types' })).toBeVisible();
    await expect(page.locator('.pref-group.is-locked').first()).toBeVisible();
    await expect(page).toHaveScreenshot('v2-profile-notifications.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('notifications, dark', async ({ page }) => {
    const { unstubbed } = await openProfile(page, '/profile/notifications', { theme: 'dark' });
    await expect(page.locator('.pref-group.is-locked').first()).toBeVisible();
    await expect(page).toHaveScreenshot('v2-profile-notifications-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('notifications @mobile', async ({ page }) => {
    const { unstubbed } = await openProfile(page, '/profile/notifications');
    await expect(page.getByRole('heading', { name: 'Types' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-profile-notifications-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('group settings', async ({ page }) => {
    const { unstubbed } = await openProfile(page, `${GROUP}/settings`, { groups: groupsWithDetails });
    await expect(page.getByRole('heading', { name: 'Teams' })).toBeVisible();
    await expect(page.getByLabel('Link label')).toHaveValue(WEBSITE_LABEL);
    await expect(page.getByText('ABC23XYZ')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-profile-settings.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('group settings @mobile', async ({ page }) => {
    const { unstubbed } = await openProfile(page, `${GROUP}/settings`, { groups: groupsWithDetails });
    await expect(page.getByRole('heading', { name: 'Teams' })).toBeVisible();
    await expect(page.getByLabel('Link label')).toHaveValue(WEBSITE_LABEL);
    await expect(page).toHaveScreenshot('v2-profile-settings-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('account', async ({ page }) => {
    const { unstubbed } = await openProfile(page, '/profile/account');
    await expect(page.getByRole('heading', { name: 'Identity' })).toBeVisible();
    await expect(page.locator('.account-email-value')).toHaveText('thibaut@example.com');
    await expect(page.locator('#account-email')).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-profile-account.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('account @mobile', async ({ page }) => {
    const { unstubbed } = await openProfile(page, '/profile/account');
    await expect(page.getByRole('heading', { name: 'Identity' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-profile-account-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });
});
