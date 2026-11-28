// An API error is written in the player's language from its code (apiError.*
// in the bundles), in the same toast as in English. The English texts of these
// screens are covered by v2-profile.spec.js and auth.spec.js.

const { test, expect } = require('@playwright/test');
const { gotoApp, expectEverythingStubbed } = require('./fixtures/app');
const { expectNoOverflow, fitViewportToPage } = require('./fixtures/screens');

const V2 = { features: ['design-v2'] };
const PHONE = { width: 360, height: 800 };
const REFUSED = { error: 'invalid position "X": use GK, DEF, MID or FWD', code: 'invalid_position' };

test.describe('api errors on a german phone', () => {
  test.use({ viewport: PHONE, locale: 'de' });

  for (const theme of ['light', 'dark']) {
    test(`a refused position save, ${theme} @mobile`, async ({ page }) => {
      const { unstubbed } = await gotoApp(page, '/profile/account', { ...V2, theme, myPositions: ['GK'], myPositionsError: REFUSED });
      const group = page.getByRole('group', { name: 'Wo du gern spielst' });
      await expect(group.getByRole('button', { name: 'TW' })).toHaveAttribute('aria-pressed', 'true');
      await group.getByRole('button', { name: 'ST' }).click();
      const toast = page.locator('.toast-message.error');
      await expect(toast).toHaveText(/Ungültige Position\./);
      await expect(toast).not.toContainText('invalid position');
      await toast.evaluate((el) => Promise.all(el.getAnimations().map((animation) => animation.finished)));
      await expectNoOverflow(page, `de api error toast ${theme}`);
      await fitViewportToPage(page);
      await expect(page).toHaveScreenshot(`v2-api-errors-de-position-${theme}.png`);
      expectEverythingStubbed(unstubbed);
    });
  }

  test('an error with no code keeps its English text @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/profile/account', { ...V2, myPositions: ['GK'], myPositionsError: 'something the app has no code for' });
    const group = page.getByRole('group', { name: 'Wo du gern spielst' });
    await group.getByRole('button', { name: 'ST' }).click();
    await expect(page.locator('.toast-message.error')).toContainText('something the app has no code for');
    expectEverythingStubbed(unstubbed);
  });
});
