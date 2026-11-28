// The translated languages on a 360 px phone, the narrowest width the design
// supports. German, the longest of the four, gets baselines in both themes for
// every signed-in screen and every public page; every language gets the layout
// invariant that matters most for translations: no screen may ever scroll
// sideways, at 360 px and at 320 px (the smallest phones still in use, where
// a label may wrap but nothing may overlap or leave the page). English keeps
// its own baselines in the other specs, untouched.

const { test, expect } = require('@playwright/test');
const data = require('./fixtures/data');
const { gotoApp, expectEverythingStubbed } = require('./fixtures/app');
const { SCREENS, PUBLIC_SCREENS, expectNoOverflow, fitViewportToPage } = require('./fixtures/screens');

const PHONE = { width: 360, height: 800 };
const SMALL_PHONE = { width: 320, height: 800 };

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

// The public pages, signed out, in German.
test.describe('german public pages at 360 px', () => {
  test.use({ locale: 'de', viewport: PHONE });

  for (const theme of ['light', 'dark']) {
    for (const screen of PUBLIC_SCREENS) {
      test(`de ${screen.name}, ${theme} @mobile`, async ({ page }) => {
        const unstubbed = await openScreen(page, screen, theme);
        await expect(page.locator('html')).toHaveAttribute('lang', 'de');
        await expectNoOverflow(page, `de ${screen.name}`);
        await fitViewportToPage(page);
        await page.evaluate(() => document.activeElement.blur());
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

// The same invariant on the narrowest phones: 320 px.
for (const lang of ['fr', 'it', 'de', 'es']) {
  test.describe(`${lang} fits 320 px`, () => {
    test.use({ locale: lang, viewport: SMALL_PHONE });

    for (const screen of [...SCREENS, ...PUBLIC_SCREENS]) {
      test(`${lang} ${screen.name}: no sideways scroll at 320 @mobile`, async ({ page }) => {
        await openScreen(page, screen);
        await expect(page.locator('html')).toHaveAttribute('lang', lang);
        await expectNoOverflow(page, `${lang} ${screen.name} at 320`);
      });
    }
  });
}
