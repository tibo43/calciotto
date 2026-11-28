// The public sign up, forgot password and reset password pages
// (SignupV2.vue, ForgotPasswordV2.vue, ResetPasswordV2.vue). No session, so
// the only API calls are the ones a test makes on purpose. Each state is
// shot full page at 1280 and 360, light and dark.

const { test, expect } = require('@playwright/test');
const { expectEverythingStubbed, expectNoHorizontalOverflow } = require('./fixtures/app');

// api.js's two clients: the API origin, and same-origin /api for the
// session calls (signup, forgot and reset password).
const API = 'http://127.0.0.1:8080';
const SESSION_API = '**/api';
const INVITE = 'A7FFYDMN';
const EXPIRED = 'This link has expired. Request a new one.';

async function openPage(page, path, { dark = false, width, stubs = {} } = {}) {
  const unstubbed = [];
  for (const origin of [API, SESSION_API]) {
    await page.route(`${origin}/**`, async (route) => {
      unstubbed.push(`${route.request().method()} ${route.request().url()}`);
      await route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"not stubbed"}' });
    });
  }
  for (const [endpoint, { status, body }] of Object.entries(stubs)) {
    await page.route(`${SESSION_API}${endpoint}`, (route) => route.fulfill({
      status, contentType: 'application/json', body: JSON.stringify(body),
    }));
  }
  if (width) await page.setViewportSize({ width, height: page.viewportSize().height });
  await page.emulateMedia({ colorScheme: dark ? 'dark' : 'light' });
  await page.goto(path);
  await page.addStyleTag({ content: '* { -webkit-font-smoothing: antialiased !important; }' });
  await expect(page.locator('#app.design-v2')).toHaveCount(1);
  await expect(page.locator('.theme-toggle')).toHaveCount(0);
  await expect(page.locator('.tabbar')).toHaveCount(0);
  return { unstubbed };
}

async function settle(page) {
  await page.evaluate(() => document.activeElement.blur());
  await page.evaluate(() => document.fonts.ready);
}

async function expectToast(page, text) {
  const toast = page.locator('.toast-message.error');
  await expect(toast).toContainText(text);
  // Measured once its slide-in is over.
  await toast.evaluate((el) => Promise.all(el.getAnimations().map((animation) => animation.finished)));
  return toast;
}

async function fillSignup(page) {
  await page.fill('#name', 'Damien Rossi');
  await page.fill('#email', 'damien@calciotto.test');
  await page.fill('#password', 'password');
}

// Each state: how to reach it, and what must be true before the shot.
const STATES = {
  'signup-empty': async (page, options) => {
    const opened = await openPage(page, '/signup', options);
    await expect(page.getByRole('heading', { name: 'Sign up' })).toBeVisible();
    await expect(page.locator('.hint')).toHaveText('Ask your group admin for it.');
    return opened;
  },
  signup: async (page, options) => {
    const opened = await openPage(page, `/signup?invite=${INVITE}`, options);
    await expect(page.locator('#invite-code')).toHaveValue(INVITE);
    await expect(page.locator('.hint')).toHaveText('Filled in from your invite link.');
    await fillSignup(page);
    return opened;
  },
  'signup-error': async (page, options) => {
    const opened = await openPage(page, `/signup?invite=${INVITE}`, {
      ...options,
      stubs: { '/auth/signup': { status: 400, body: { error: 'no group matches this invite code' } } },
    });
    await fillSignup(page);
    await page.getByRole('button', { name: 'Sign up', exact: true }).click();
    await expectToast(page, 'No group matches this invite code.');
    await expect(page).toHaveURL(/\/signup\?invite=/);
    return opened;
  },
  forgot: async (page, options) => {
    const opened = await openPage(page, '/forgot-password', options);
    await expect(page.getByRole('heading', { name: 'Forgot password' })).toBeVisible();
    await page.fill('#email', 'thibaut@calciotto.test');
    await expect(page.getByRole('button', { name: 'Send the link' })).toBeVisible();
    return opened;
  },
  'forgot-sent': async (page, options) => {
    const opened = await openPage(page, '/forgot-password', {
      ...options,
      stubs: { '/auth/forgot-password': { status: 200, body: { message: 'if an account exists for this email, a reset link has been sent' } } },
    });
    await page.fill('#email', 'thibaut@calciotto.test');
    await page.getByRole('button', { name: 'Send the link' }).click();
    await expect(page.locator('.auth-notice[role="status"]')).toHaveText('If an account exists for this email, a reset link has been sent.');
    await expect(page.getByRole('button', { name: 'Send again' })).toBeEnabled();
    await expect(page.locator('.toast-message')).toHaveCount(0);
    return opened;
  },
  reset: async (page, options) => {
    const opened = await openPage(page, '/reset-password?token=visual', options);
    await expect(page.getByRole('heading', { name: 'New password' })).toBeVisible();
    await expect(page.locator('main input')).toHaveCount(1);
    await page.fill('#password', 'password');
    return opened;
  },
  'reset-expired': async (page, options) => {
    const opened = await openPage(page, '/reset-password?token=visual', {
      ...options,
      stubs: { '/auth/reset-password': { status: 400, body: { error: 'invalid or expired reset token' } } },
    });
    await page.fill('#password', 'password');
    await page.getByRole('button', { name: 'Save password' }).click();
    await expectToast(page, EXPIRED);
    await expect(page).toHaveURL(/\/reset-password\?token=visual$/);
    return opened;
  },
};

for (const [state, open] of Object.entries(STATES)) {
  for (const theme of ['light', 'dark']) {
    const dark = theme === 'dark';

    test(`${state} ${theme} 1280`, async ({ page }) => {
      const { unstubbed } = await open(page, { dark });
      await settle(page);
      await expect(page).toHaveScreenshot(`${state}-${theme}-1280.png`, { fullPage: true });
      await expectNoHorizontalOverflow(page);
      expectEverythingStubbed(unstubbed);
    });

    test(`${state} ${theme} 360 @mobile`, async ({ page }) => {
      const { unstubbed } = await open(page, { dark, width: 360 });
      await settle(page);
      await expect(page).toHaveScreenshot(`${state}-${theme}-360.png`, { fullPage: true });
      await expectNoHorizontalOverflow(page);
      expectEverythingStubbed(unstubbed);
    });
  }
}

test.describe('auth pages', () => {
  for (const path of [`/signup?invite=${INVITE}`, '/forgot-password', '/reset-password?token=visual']) {
    test(`${path} never scrolls sideways at 360 and 768 @mobile`, async ({ page }) => {
      for (const width of [360, 768]) {
        const { unstubbed } = await openPage(page, path, { width });
        await expectNoHorizontalOverflow(page);
        expectEverythingStubbed(unstubbed);
        await page.unrouteAll({ behavior: 'ignoreErrors' });
      }
    });
  }

  test('the web toast sits in the bottom right corner', async ({ page }) => {
    const { unstubbed } = await STATES['reset-expired'](page, {});
    const box = await page.locator('.toast-message.error').boundingBox();
    expect(1280 - (box.x + box.width)).toBeLessThanOrEqual(32);
    expectEverythingStubbed(unstubbed);
  });

  test('the phone toast sits 12px from the bottom, with a 44px close button and room under the last link @mobile', async ({ page }) => {
    const { unstubbed } = await STATES['signup-error'](page, { width: 360 });
    const toast = page.locator('.toast-message.error');
    const box = await toast.boundingBox();
    const height = page.viewportSize().height;
    expect(Math.round(height - (box.y + box.height))).toBe(12);
    expect(Math.round(box.x)).toBe(12);
    const close = await toast.locator('.toast-close').boundingBox();
    expect(Math.round(close.width)).toBeGreaterThanOrEqual(44);
    expect(Math.round(close.height)).toBeGreaterThanOrEqual(44);
    const padding = await page.locator('main.auth-v2').evaluate((el) => parseFloat(getComputedStyle(el).paddingBottom));
    expect(padding).toBeGreaterThanOrEqual(88);
    // Scrolled to the end, the last link clears the toast.
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    const link = await page.getByRole('link', { name: 'Already have an account? Log in' }).boundingBox();
    const toastAfter = await toast.boundingBox();
    expect(link.y + link.height).toBeLessThanOrEqual(toastAfter.y);
    expectEverythingStubbed(unstubbed);
  });

  test('a reset link with no token says it has expired, without a request', async ({ page }) => {
    const { unstubbed } = await openPage(page, '/reset-password');
    await page.fill('#password', 'password');
    await page.getByRole('button', { name: 'Save password' }).click();
    await expectToast(page, EXPIRED);
    expectEverythingStubbed(unstubbed);
  });

  test('an empty invite code is refused in the toast, without a request', async ({ page }) => {
    const { unstubbed } = await openPage(page, '/signup');
    await fillSignup(page);
    await page.getByRole('button', { name: 'Sign up', exact: true }).click();
    await expectToast(page, 'Invite code is required.');
    expectEverythingStubbed(unstubbed);
  });
});

// The public pages follow the browser's language: a refused sign up reads in
// German, from the error's code.
test.describe('auth pages on a german phone', () => {
  test.use({ locale: 'de', viewport: { width: 360, height: 800 } });

  for (const theme of ['light', 'dark']) {
    test(`a refused sign up, ${theme} @mobile`, async ({ page }) => {
      const { unstubbed } = await openPage(page, `/signup?invite=${INVITE}`, {
        dark: theme === 'dark',
        width: 360,
        stubs: { '/auth/signup': { status: 400, body: { error: 'no group matches this invite code', code: 'invite_code_not_found' } } },
      });
      await fillSignup(page);
      await page.locator('form button[type="submit"]').click();
      const toast = await expectToast(page, 'Zu diesem Einladungscode gibt es keine Gruppe.');
      await expect(toast).not.toContainText('invite code');
      await settle(page);
      await expect(page).toHaveScreenshot(`signup-error-de-${theme}-360.png`, { fullPage: true });
      await expectNoHorizontalOverflow(page);
      expectEverythingStubbed(unstubbed);
    });
  }
});
