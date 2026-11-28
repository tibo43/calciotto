// The Account language selector (design v2): the phone's language first, a
// rule, then the others. The language is saved on the server (stubbed here),
// never on the device.

const { test, expect } = require('@playwright/test');
const { gotoApp, expectEverythingStubbed } = require('./fixtures/app');
const { expectNoOverflow, fitViewportToPage } = require('./fixtures/screens');

const V2 = { features: ['design-v2'] };
const ACCOUNT = '/profile/account';
const PHONE = { width: 360, height: 800 };

async function openList(page) {
  await page.getByRole('button', { name: /^(Language|Sprache|Langue|Idioma|Lingua)/ }).click();
  await expect(page.getByRole('radiogroup')).toBeVisible();
}

test.describe('account language selector', () => {
  test('list open', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, ACCOUNT, V2);
    await openList(page);
    await expect(page.getByRole('radio', { name: 'English' })).toHaveAttribute('aria-checked', 'true');
    await expect(page).toHaveScreenshot('v2-language-open.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('list open, dark', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, ACCOUNT, { ...V2, theme: 'dark' });
    await openList(page);
    await expect(page).toHaveScreenshot('v2-language-open-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('picking a language applies it at once and saves it; the first line goes back to the phone', async ({ page }) => {
    const localeWrites = [];
    const { unstubbed } = await gotoApp(page, ACCOUNT, { ...V2, localeWrites });
    await openList(page);

    await page.getByRole('radio', { name: 'Français' }).click();
    await expect(page.locator('html')).toHaveAttribute('lang', 'fr');
    await expect(page.getByRole('radio', { name: 'Français' })).toHaveAttribute('aria-checked', 'true');
    await expect(page.getByRole('button', { name: /^Langue/ })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Apparence' })).toBeVisible();
    await expect.poll(() => localeWrites).toEqual(['fr']);

    await page.getByRole('radio', { name: 'English' }).click();
    await expect(page.locator('html')).toHaveAttribute('lang', 'en');
    await expect.poll(() => localeWrites).toEqual(['fr', null]);
    expectEverythingStubbed(unstubbed);
  });

  test('a failed save puts the previous language back and says so, in that language', async ({ page }) => {
    const localeWrites = [];
    const { unstubbed } = await gotoApp(page, ACCOUNT, { ...V2, localeWrites, localeSaveError: true });
    await openList(page);

    await page.getByRole('radio', { name: 'Français' }).click();

    const toast = page.locator('.toast-message');
    await expect(toast).toContainText("Couldn't save your language. Try again.");
    await expect(toast).toHaveClass(/error/);
    await expect(page.locator('html')).toHaveAttribute('lang', 'en');
    await expect(page.getByRole('radio', { name: 'English' })).toHaveAttribute('aria-checked', 'true');
    await expect(page.getByRole('radio', { name: 'Français' })).toHaveAttribute('aria-checked', 'false');
    await expect(page.getByRole('heading', { name: 'Appearance' })).toBeVisible();
    expect(localeWrites).toEqual([]);
    expectEverythingStubbed(unstubbed);
  });

  test('a saved language wins over the phone\'s at start-up', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, ACCOUNT, { ...V2, locale: 'it' });
    await expect(page.locator('html')).toHaveAttribute('lang', 'it');
    await expect(page.getByRole('button', { name: /^Lingua/ })).toBeVisible();
    await openList(page);
    // The phone speaks English: it stays the first line, Italian is checked below the rule.
    await expect(page.getByRole('radio').first()).toHaveText('English');
    await expect(page.getByRole('radio', { name: 'Italiano' })).toHaveAttribute('aria-checked', 'true');
    expectEverythingStubbed(unstubbed);
  });

  test('keeps nothing in localStorage', async ({ page }) => {
    await gotoApp(page, ACCOUNT, V2);
    await openList(page);
    await page.getByRole('radio', { name: 'Deutsch' }).click();
    const keys = await page.evaluate(() => Object.keys(localStorage).sort());
    expect(keys.filter((key) => /lang|locale/i.test(key))).toEqual([]);
  });
});

test.describe('account language selector on a german phone', () => {
  test.use({ viewport: PHONE, locale: 'de' });

  for (const theme of ['light', 'dark']) {
    test(`german phone, list open, ${theme} @mobile`, async ({ page }) => {
      const { unstubbed } = await gotoApp(page, ACCOUNT, { ...V2, theme });
      await openList(page);
      await expect(page.getByRole('radio').first()).toHaveText('Deutsch');
      await expectNoOverflow(page, `de account ${theme}`);
      await fitViewportToPage(page);
      await expect(page).toHaveScreenshot(`v2-language-de-${theme}.png`);
      expectEverythingStubbed(unstubbed);
    });

    // The save fails: German is put back, and the toast says so in German.
    test(`german phone, save error toast, ${theme} @mobile`, async ({ page }) => {
      const { unstubbed } = await gotoApp(page, ACCOUNT, { ...V2, theme, localeSaveError: true });
      await openList(page);
      await page.getByRole('radio', { name: 'English' }).click();
      await expect(page.locator('.toast-message')).toContainText('Sprache konnte nicht gespeichert werden. Versuch es noch mal.');
      await expect(page.locator('html')).toHaveAttribute('lang', 'de');
      await expect(page.getByRole('radio', { name: 'Deutsch' })).toHaveAttribute('aria-checked', 'true');
      await expectNoOverflow(page, `de save error toast ${theme}`);
      await fitViewportToPage(page);
      await expect(page).toHaveScreenshot(`v2-language-de-save-error-toast-${theme}.png`);
      expectEverythingStubbed(unstubbed);
    });
  }
});
