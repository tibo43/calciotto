// The match detail page (`/matches/:id/edit`), which is where this app has the
// most state-dependent UI: the same route renders an editable scoresheet, a
// read-only one, or a sign-up panel in one of five states, depending on the
// match and on the caller's role.
//
// These are the cases where a CSS or markup regression would be both invisible
// to the unit tests (jsdom renders no pixels) and expensive in production.

const { test, expect } = require('@playwright/test');
const { gotoApp, expectEverythingStubbed, data } = require('./fixtures/app');

const url = (match) => `/matches/${match.ID}/edit`;

test.describe('a played match', () => {
  test('as an admin, fully editable', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, url(data.playedMatch));
    // Save Changes is admin-only, so its presence is what proves the role
    // landed before the screenshot.
    await expect(page.getByRole('button', { name: 'Save Changes' })).toBeVisible();
    await expect(page).toHaveScreenshot('played-admin.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('as a plain member, read-only', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, url(data.playedMatch), {
      groups: [{ id: data.GROUP_ID, name: 'Calciotto Milano', role: 'member', is_favorite: true }],
    });
    // Names, avatars and goal counts stay visible; every control that writes is
    // gone. Asserted as well as photographed, so the reason this baseline looks
    // the way it does is written down.
    await expect(page.locator('.player-card').first()).toBeVisible();
    await expect(page.getByRole('button', { name: 'Save Changes' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Delete Match' })).toHaveCount(0);
    await expect(page).toHaveScreenshot('played-member.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });
});

test.describe('a scheduled match', () => {
  test('sign-ups open, with a waiting list', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, url(data.scheduledOpenMatch), {
      groups: [{ id: data.GROUP_ID, name: 'Calciotto Milano', role: 'member', is_favorite: true }],
    });
    await expect(page.locator('.signup-panel')).toBeVisible();
    // 18 sign-ups against a cap of 16: the two extra are the waiting list, which
    // exists only as a consequence of the ordering — worth having in a baseline.
    await expect(page.locator('.signup-list-waiting')).toBeVisible();
    await expect(page).toHaveScreenshot('scheduled-open-member.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('sign-ups open, in dark mode', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, url(data.scheduledOpenMatch), { theme: 'dark' });
    await expect(page.locator('.dark-mode')).toBeVisible();
    await expect(page.locator('.signup-panel')).toBeVisible();
    await expect(page).toHaveScreenshot('scheduled-open-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('sign-ups closed, as the admin composing the teams', async ({ page }) => {
    const closed = data.scheduledClosedMatch;
    const { unstubbed } = await gotoApp(page, url(closed), {
      matchDetails: { [closed.ID]: closed },
    });
    await expect(page.locator('.signup-panel')).toBeVisible();
    // The whole point of this state: the list is frozen and the mechanical
    // team split is offered. It is admin-only and closed-only, so both halves
    // of the gate are covered by this one baseline.
    await expect(page.getByRole('button', { name: /Fill teams from sign-ups/ })).toBeVisible();
    await expect(page).toHaveScreenshot('scheduled-closed-admin.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('sign-ups not open yet', async ({ page }) => {
    // The one state built by overriding a timestamp rather than by its own
    // fixture: identical to the open match except that the window opens after
    // the frozen clock, which is exactly what deriveRegistrationState reads.
    const notOpenYet = {
      ...data.scheduledOpenMatch,
      ID: 'cccccccc-0000-4000-8000-000000000005',
      RegistrationOpensAt: '2026-09-05T18:00:00+02:00',
      RegistrationCount: 0,
    };
    const { unstubbed } = await gotoApp(page, url(notOpenYet), {
      matchDetails: { [notOpenYet.ID]: notOpenYet },
      registrations: [],
    });
    await expect(page.locator('.signup-panel')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Participate' })).toHaveCount(0);
    await expect(page).toHaveScreenshot('scheduled-not-open-yet.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });
});
