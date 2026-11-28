// The translated languages on a 360 px phone, the narrowest width the design
// supports. German, the longest of the four, gets full-page baselines in both
// themes; every language gets the layout invariant that matters most for
// translations: no screen may ever scroll sideways. English keeps its own
// baselines in the other specs, untouched.

const { test, expect } = require('@playwright/test');
const data = require('./fixtures/data');
const { gotoApp, expectEverythingStubbed } = require('./fixtures/app');
const { SCREENS, PUBLIC_SCREENS, expectNoOverflow, fitViewportToPage } = require('./fixtures/screens');

const PHONE = { width: 360, height: 800 };

async function openScreen(page, screen, theme = 'light') {
  if (screen.options) {
    const { unstubbed } = await gotoApp(page, screen.path, { ...screen.options, theme });
    await expect(page.locator('.design-v2').first()).toBeVisible();
    return unstubbed;
  }
  // Signed out: no token, and an API that answers nothing.
  await page.route('http://127.0.0.1:8080/**', (route) => route.fulfill({ status: 404, body: '{}' }));
  await page.clock.setFixedTime(data.FIXED_NOW);
  await page.addInitScript((value) => localStorage.setItem('calciotto-theme', value), theme);
  await page.goto(screen.path);
  await expect(page.locator('main, form').first()).toBeVisible();
  return [];
}

test.describe('german at 360 px', () => {
  test.use({ locale: 'de', viewport: PHONE });

  for (const theme of ['light', 'dark']) {
    for (const screen of SCREENS) {
      test(`de ${screen.name}, ${theme} @mobile`, async ({ page }) => {
        const unstubbed = await openScreen(page, screen, theme);
        await expect(page.locator('html')).toHaveAttribute('lang', 'de');
        await expectNoOverflow(page, `de ${screen.name}`);
        // A viewport as tall as the page, not fullPage: the fixed bars stay at the bottom.
        await fitViewportToPage(page);
        await expect(page).toHaveScreenshot(`de-${screen.name}-${theme}.png`);
        expectEverythingStubbed(unstubbed);
      });
    }
  }
});

// Colours never move a line break, so one theme is enough here; German dark
// is checked by the baselines above, which run the same check.
for (const lang of ['fr', 'it', 'de', 'es']) {
  test.describe(`${lang} fits 360 px`, () => {
    test.use({ locale: lang, viewport: PHONE });

    for (const screen of [...SCREENS, ...PUBLIC_SCREENS]) {
      test(`${lang} ${screen.name}: no sideways scroll @mobile`, async ({ page }) => {
        await openScreen(page, screen);
        await expect(page.locator('html')).toHaveAttribute('lang', lang);
        await expectNoOverflow(page, `${lang} ${screen.name}`);
      });
    }
  });
}
