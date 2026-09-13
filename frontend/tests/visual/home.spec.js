// The unified home page (`/`): one season selector over three sub-tabs. Each
// tab is captured separately because they are three different components
// (MatchesPanel, PointsStandingsTable, ScorersTable) sharing one shell — a diff
// in one of them says which.

const { test, expect } = require('@playwright/test');
const { gotoApp, expectEverythingStubbed, expectNoHorizontalOverflow, pinCarouselScroll } = require('./fixtures/app');

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

  test('motm tab', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/');
    await page.getByRole('button', { name: 'MOTM' }).click();
    await expect(page.locator('.left-group-tag').first()).toBeVisible();
    await expect(page).toHaveScreenshot('motm-tab.png', { fullPage: true });
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
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  // The MOTM sub-tab specifically: real feedback found it sticking out past
  // the right edge of a phone screen, needing a sideways scroll to reach —
  // four equal-flex tab buttons that didn't shrink (see
  // expectNoHorizontalOverflow's own comment). This tab had no @mobile
  // coverage at all before that, on any baseline — it's the newest of the
  // four sub-tabs and was added after the "matches tab" mobile baseline
  // above already existed.
  test('motm tab on a narrow viewport @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/');
    await page.getByRole('button', { name: 'MOTM' }).click();
    await expect(page.locator('.left-group-tag').first()).toBeVisible();
    await expect(page).toHaveScreenshot('motm-tab-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  // The inline sign-up panel added to the "Selected Match Details" preview —
  // reported missing from real testing, since the only way to sign up used to
  // be leaving this tab for the match's own page. Selecting the scheduled
  // fixture match is what makes the panel render at all (isScheduledMatch),
  // so this is its own baseline rather than folded into "matches tab" above,
  // whose default selection is the unscheduled olderPlayedMatch.
  test('matches tab, scheduled match selected, inline sign-up panel', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/');
    await page.locator('.match-card-horizontal.scheduled').first().click();
    // Parked at the carousel's end stop, not merely waited on. An explicit
    // instant scrollIntoView plus a settle-wait used to stand here, and it was
    // not enough: CI failed this baseline by 652 pixels — all of them inside
    // this carousel's selected card — while reporting "captured a stable
    // screenshot", i.e. the scroll had finished somewhere a pixel or so from
    // where the baseline was recorded. See pinCarouselScroll.
    await pinCarouselScroll(page, '.matches-bar', 'end');
    await expect(page.locator('.signup-inline')).toBeVisible();
    await expect(page).toHaveScreenshot('matches-tab-signup-inline.png', { fullPage: true });
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

  // The confirmed/waiting split and Participate/Withdraw, as a plain member —
  // this used to be its own baseline on the match detail page, but that page
  // is admin-only now (see the router guard in router/index.js) and a member
  // can no longer reach it at all. The inline panel here is what replaced it,
  // and it's what a member actually uses to sign up in practice, so this is
  // the more representative baseline of the two roles, not a downgrade of it.
  test('matches tab, scheduled match selected, inline sign-up panel, as a plain member', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', {
      groups: [{ id: '11111111-1111-4111-8111-111111111111', name: 'Calciotto Milano', role: 'member', is_favorite: true }],
    });
    await page.locator('.match-card-horizontal.scheduled').first().click();
    // See the admin-role test above for why the carousel is pinned to an end
    // stop rather than left wherever the click-time auto-scroll landed it.
    // This is the baseline that actually failed in CI.
    await pinCarouselScroll(page, '.matches-bar', 'end');
    await expect(page.locator('.signup-inline')).toBeVisible();
    // 18 sign-ups against a cap of 16: the two extra are the waiting list,
    // which exists only as a consequence of the ordering — worth having in a
    // baseline, and specifically absent from the admin-role baseline above.
    await expect(page.locator('.signup-list-waiting')).toBeVisible();
    await expect(page.locator('.edit-match-btn')).toHaveCount(0);
    await expect(page).toHaveScreenshot('matches-tab-signup-inline-member.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });
});
