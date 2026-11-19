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

// Noon UTC, so the day is the same in every time zone the tests could run in.
const PASSKEYS = [
  { id: 'pk-1', name: 'iPhone', created_at: '2026-09-02T12:00:00Z', last_used_at: '2026-09-03T12:00:00Z' },
  { id: 'pk-2', name: 'MacBook Air', created_at: '2026-08-28T12:00:00Z', last_used_at: null },
];

// Invitation and Passkeys are collapsible cards, closed by default.
async function openCard(page, title) {
  const header = page.locator('.cc-head', { hasText: title });
  await header.click();
  await expect(header).toHaveAttribute('aria-expanded', 'true');
  await page.evaluate(() => window.scrollTo(0, 0));
}

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

  test('group, not admin', async ({ page }) => {
    const member = groupsWithDetails.map((group) => ({ ...group, role: 'member' }));
    const { unstubbed } = await openProfile(page, GROUP, { groups: member });
    await expect(page.getByRole('link', { name: WEBSITE_LABEL })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Group settings' })).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-profile-group-member.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('profile, long name @mobile', async ({ page }) => {
    const long = JSON.parse(JSON.stringify(stats));
    long.Overall.Name = 'Maximilien Dumoulin-Castellanos';
    const { unstubbed } = await openProfile(page, '/profile', { playerStats: long });
    await expect(page.getByRole('heading', { name: 'Maximilien Dumoulin-Castellanos' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Manage account' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-profile-long-name-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
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
    await expect(page.getByRole('button', { name: /Invite players/ })).toHaveAttribute('aria-expanded', 'false');
    await expect(page.getByText('Code active')).toBeVisible();
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
    await expect(page.getByText('Passkeys', { exact: true })).toHaveCount(0);
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

  test('invitation open', async ({ page }) => {
    const { unstubbed } = await openProfile(page, `${GROUP}/settings`, { groups: groupsWithDetails });
    await openCard(page, 'Invite players');
    await expect(page.getByText('ABC23XYZ')).toBeVisible();
    await expect(page.locator('.invite-tile')).toHaveCount(3);
    await expect(page.getByText('Revoke')).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-profile-settings-invite-open.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('invitation open @mobile', async ({ page }) => {
    const { unstubbed } = await openProfile(page, `${GROUP}/settings`, { groups: groupsWithDetails });
    await openCard(page, 'Invite players');
    await expect(page.getByText('ABC23XYZ')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-profile-settings-invite-open-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('group settings without an invite @mobile', async ({ page }) => {
    const { unstubbed } = await openProfile(page, `${GROUP}/settings`, { groups: groupsWithDetails, inviteCode: '' });
    await expect(page.getByText('No active code')).toBeVisible();
    await openCard(page, 'Invite players');
    await expect(page.getByRole('button', { name: 'Make a code' })).toBeVisible();
    await expect(page.locator('.invite-tiles')).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-profile-settings-no-invite-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('renew code confirmation @mobile', async ({ page }) => {
    const { unstubbed } = await openProfile(page, `${GROUP}/settings`, { groups: groupsWithDetails });
    await openCard(page, 'Invite players');
    await page.getByRole('button', { name: 'Renew', exact: true }).click();
    const sheet = page.locator('.confirm-sheet');
    await expect(sheet.getByRole('heading', { name: 'Renew the code?' })).toBeVisible();
    await expect(page.locator('.confirm-pop')).toBeHidden();
    await expect(page).toHaveScreenshot('v2-profile-settings-renew-mobile.png');
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  // A failed renewal lands in the shared toast (above the tab bar on phones).
  async function failRenewal(page) {
    await page.route('**/groups/*/invite-code', (route) => (route.request().method() === 'POST'
      ? route.fulfill({ status: 500, contentType: 'application/json', body: '{}' })
      : route.fallback()));
    await openCard(page, 'Invite players');
    await page.getByRole('button', { name: 'Renew', exact: true }).click();
    await page.locator('.confirm-pop, .confirm-sheet').filter({ visible: true }).getByRole('button', { name: 'Renew', exact: true }).click();
    const toast = page.locator('.toast-message.error');
    await expect(toast).toContainText('Could not renew the code.');
    return toast;
  }

  test('renew failure toast', async ({ page }) => {
    const { unstubbed } = await openProfile(page, `${GROUP}/settings`, { groups: groupsWithDetails });
    await failRenewal(page);
    await expect(page).toHaveScreenshot('v2-profile-settings-renew-error.png');
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('renew failure toast @mobile', async ({ page }) => {
    const { unstubbed } = await openProfile(page, `${GROUP}/settings`, { groups: groupsWithDetails, theme: 'dark' });
    const toast = await failRenewal(page);
    const toastBox = await toast.boundingBox();
    const tabBox = await page.locator('nav.tabbar').boundingBox();
    expect(toastBox.y + toastBox.height).toBeLessThanOrEqual(tabBox.y);
    await expect(page).toHaveScreenshot('v2-profile-settings-renew-error-mobile.png');
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('account with passkeys, closed', async ({ page }) => {
    const { unstubbed } = await openProfile(page, '/profile/account', { features: [...V2.features, 'passkeys'], passkeys: PASSKEYS, theme: 'dark' });
    await expect(page.getByText('2 devices')).toBeVisible();
    await expect(page.locator('.pk-row')).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-profile-account-passkeys-closed.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('account with passkeys', async ({ page }) => {
    const { unstubbed } = await openProfile(page, '/profile/account', { features: [...V2.features, 'passkeys'], passkeys: PASSKEYS });
    await openCard(page, 'Passkeys');
    await expect(page.getByText('Last used 3 Sep 2026')).toBeVisible();
    await expect(page.getByText('Added 28 Aug 2026')).toBeVisible();
    await page.getByRole('button', { name: 'Remove iPhone' }).click();
    await expect(page.locator('.confirm-pop').getByRole('heading', { name: 'Remove this passkey?' })).toBeVisible();
    await expect(page.locator('.confirm-sheet')).toBeHidden();
    await expect(page).toHaveScreenshot('v2-profile-account-passkeys-remove.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('account with passkeys @mobile', async ({ page }) => {
    const { unstubbed } = await openProfile(page, '/profile/account', { features: [...V2.features, 'passkeys'], passkeys: PASSKEYS, theme: 'dark' });
    await openCard(page, 'Passkeys');
    await expect(page.locator('.pk-row')).toHaveCount(2);
    await page.evaluate(() => window.scrollTo(0, 0));
    await expect(page).toHaveScreenshot('v2-profile-account-passkeys-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('add passkey form @mobile', async ({ page }) => {
    const { unstubbed } = await openProfile(page, '/profile/account', { features: [...V2.features, 'passkeys'] });
    await expect(page.getByText('None yet')).toBeVisible();
    await openCard(page, 'Passkeys');
    await expect(page.getByText('Sign in with Face ID, your fingerprint or your screen lock.')).toBeVisible();
    await page.getByRole('button', { name: 'Add a passkey' }).click();
    await expect(page.getByLabel('Name (optional)')).toHaveAttribute('placeholder', 'My phone');
    // The click scrolled the form into view. A full-page capture from the
    // top keeps the sticky top bar where it belongs.
    await page.evaluate(() => window.scrollTo(0, 0));
    await expect(page).toHaveScreenshot('v2-profile-account-passkey-form-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('passkeys appear after sign-in with both flags', async ({ page }) => {
    const passkeyLogin = page.getByRole('button', { name: 'Log in with a passkey', exact: true });
    await page.goto('/login');
    await expect(page.getByRole('button', { name: 'Log in', exact: true })).toBeVisible();
    await expect(passkeyLogin).toHaveCount(0);

    const { unstubbed } = await openProfile(page, '/profile/account', { features: [...V2.features, 'passkeys'] });
    await expect(page.getByRole('heading', { name: 'Passkeys' })).toBeVisible();
    await expect(page.getByText('None yet')).toBeVisible();
    await page.goto('/login');
    await expect(passkeyLogin).toBeVisible();
    expectEverythingStubbed(unstubbed);
  });
});
