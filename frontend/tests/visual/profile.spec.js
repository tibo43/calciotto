// The profile page (`/profile`): cross-group stats, the groups carousel, and
// the roster panel a group's admin manages members from.
//
// Only the roster panel is photographed here, and only at phone width — that
// is the layout that actually had a problem. Its per-member admin actions
// (promote/demote, remove) used to be text buttons, which did not fit on the
// name's own line on a phone: the row wrapped and every member cost two
// lines. They are icon-only now (an up/down arrow and an X), with the row
// held to `flex-wrap: nowrap` so a long name truncates instead of pushing
// them off. A pixel diff is the only thing that can catch that regressing —
// Jest renders nothing, and the markup would look perfectly correct either
// way.

const { test, expect } = require('@playwright/test');
const { gotoApp, expectEverythingStubbed, expectNoHorizontalOverflow } = require('./fixtures/app');
const data = require('./fixtures/data');

test.describe('profile page', () => {
  // The fixture's first group has the caller as `admin`, so this photographs
  // the branch with the action buttons present. The roster override carries a
  // deliberately over-long name, which is what proves the truncation rather
  // than the actions being pushed out of the row.
  test('group roster as an admin on a narrow viewport @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/profile', { groupMembers: data.groupMembersLongName });
    await page.locator('.group-card-horizontal').first().click();
    // Waiting on an action button rather than the panel itself: the panel
    // renders before its members have loaded, so this is what proves the
    // roster (and the caller's admin role on this specific group) resolved
    // before the screenshot.
    await expect(page.locator('.member-action-btn').first()).toBeVisible();
    await expect(page).toHaveScreenshot('profile-roster-admin-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  // The same panel for a group the caller is only a member of: no action
  // buttons at all. Its own baseline because the gate is per-group (via
  // groupMeta), not "is admin of some group" — a regression there would show
  // up as buttons appearing here, which the admin baseline above cannot see.
  test('group roster as a plain member on a narrow viewport @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/profile');
    await page.locator('.group-card-horizontal').nth(1).click();
    await expect(page.locator('.member-list')).toBeVisible();
    await expect(page.locator('.member-action-btn')).toHaveCount(0);
    await expect(page).toHaveScreenshot('profile-roster-member-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });
});
