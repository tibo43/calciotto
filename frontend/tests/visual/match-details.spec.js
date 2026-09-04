// The match detail page (`/matches/:id/edit`), admin-only (see the router
// guard in router/index.js — a plain member is redirected to `/` before this
// page ever mounts, so there is no "as a plain member" scenario left to
// photograph here at all; that content now lives in home.spec.js instead,
// on the page a member actually reaches). What's left is still the most
// state-dependent UI an admin sees: an editable scoresheet or a sign-up
// panel in one of several states, depending on the match.
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
});

test.describe('a scheduled match', () => {
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
