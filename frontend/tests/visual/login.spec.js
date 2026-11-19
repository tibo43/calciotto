// The login page: today's Login.vue, and LoginV2.vue on a browser that
// carries the v2 login mark (services/loginV2.js). No session, so the only
// API calls are the ones a test makes on purpose.

const { test, expect } = require('@playwright/test');
const { expectEverythingStubbed, expectNoHorizontalOverflow } = require('./fixtures/app');

// api.js's two clients: the API origin, and same-origin /api for the
// session calls (login, passkey login).
const API = 'http://127.0.0.1:8080';
const SESSION_API = '**/api';
const PASSKEY_ERROR = 'Could not log in with a passkey. Try again or use your password.';

// mark: the v2 login mark; passkey: the passkey login hint; device: what
// navigator.credentials.get does ('wait' never answers, 'cancel' rejects
// like a dismissed prompt). The theme follows the emulated device scheme.
async function openLogin(page, { mark = true, passkey = true, dark = false, device = null, width } = {}) {
  const unstubbed = [];
  for (const origin of [API, SESSION_API]) {
    await page.route(`${origin}/**`, async (route) => {
      unstubbed.push(`${route.request().method()} ${route.request().url()}`);
      await route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"not stubbed"}' });
    });
  }
  await page.route(`${SESSION_API}/auth/passkeys/options`, (route) => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ challenge_id: 'visual', publicKey: { challenge: 'dmlzdWFs', rpId: '127.0.0.1', allowCredentials: [], timeout: 60000, userVerification: 'preferred' } }),
  }));
  await page.route(`${SESSION_API}/auth/login`, (route) => route.fulfill({
    status: 401, contentType: 'application/json', body: '{"error":"invalid email or password"}',
  }));
  if (width) await page.setViewportSize({ width, height: page.viewportSize().height });
  await page.emulateMedia({ colorScheme: dark ? 'dark' : 'light' });
  await page.addInitScript(({ mark, passkey, device }) => {
    if (mark) localStorage.setItem('calciotto-login-v2', '1');
    if (passkey) localStorage.setItem('calciotto-passkey-login', '1');
    if (device === 'wait') navigator.credentials.get = () => new Promise(() => {});
    if (device === 'cancel') {
      navigator.credentials.get = () => Promise.reject(new DOMException('The operation was not allowed.', 'NotAllowedError'));
    }
  }, { mark, passkey, device });
  await page.goto('/login');
  await page.addStyleTag({ content: '* { -webkit-font-smoothing: antialiased !important; }' });
  await page.fill('#email', 'thibaut@calciotto.test');
  await page.fill('#password', 'password');
  await page.evaluate(() => document.activeElement.blur());
  await page.evaluate(() => document.fonts.ready);
  return { unstubbed };
}

async function expectV2(page, { passkey = true } = {}) {
  await expect(page.locator('#app.design-v2')).toHaveCount(1);
  await expect(page.getByRole('heading', { name: 'Log in' })).toBeVisible();
  await expect(page.locator('.theme-toggle')).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Log in with a passkey' })).toHaveCount(passkey ? 1 : 0);
  await expect(page.locator('.or')).toHaveCount(passkey ? 1 : 0);
}

async function failPasskey(page) {
  await page.getByRole('button', { name: 'Log in with a passkey' }).click();
  const toast = page.locator('.toast-message.error');
  await expect(toast).toContainText(PASSKEY_ERROR);
  // Measured once its slide-in is over.
  await toast.evaluate((el) => Promise.all(el.getAnimations().map((animation) => animation.finished)));
  await expect(page.locator('main [role="alert"]')).toHaveCount(0);
  return toast;
}

test.describe('login', () => {
  test('login v2', async ({ page }) => {
    const { unstubbed } = await openLogin(page);
    await expectV2(page);
    await expect(page).toHaveScreenshot('login-v2.png');
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('login v2 dark', async ({ page }) => {
    const { unstubbed } = await openLogin(page, { dark: true });
    await expectV2(page);
    await expect(page).toHaveScreenshot('login-v2-dark.png');
    expectEverythingStubbed(unstubbed);
  });

  test('login v2 without a passkey', async ({ page }) => {
    const { unstubbed } = await openLogin(page, { passkey: false });
    await expectV2(page, { passkey: false });
    await expect(page).toHaveScreenshot('login-v2-no-passkey.png');
    expectEverythingStubbed(unstubbed);
  });

  test('login v2 passkey error toast', async ({ page }) => {
    const { unstubbed } = await openLogin(page, { device: 'cancel' });
    const toast = await failPasskey(page);
    const box = await toast.boundingBox();
    expect(1280 - (box.x + box.width)).toBeLessThanOrEqual(32);
    await expect(page).toHaveScreenshot('login-v2-passkey-error.png');
    expectEverythingStubbed(unstubbed);
  });

  test('login v2 wrong password goes to the toast', async ({ page }) => {
    const { unstubbed } = await openLogin(page);
    await page.getByRole('button', { name: 'Log in', exact: true }).click();
    await expect(page.locator('.toast-message.error')).toContainText('invalid email or password');
    await expect(page.locator('main [role="alert"]')).toHaveCount(0);
    await expect(page).toHaveURL(/\/login$/);
    expectEverythingStubbed(unstubbed);
  });

  test('login v1 without the mark', async ({ page }) => {
    const { unstubbed } = await openLogin(page, { mark: false });
    await expect(page.locator('#app.design-v2')).toHaveCount(0);
    await expect(page.locator('.theme-toggle')).toHaveCount(1);
    await expect(page).toHaveScreenshot('login-v1.png');
    expectEverythingStubbed(unstubbed);
  });

  test('login v2 @mobile', async ({ page }) => {
    const { unstubbed } = await openLogin(page, { dark: true, width: 360 });
    await expectV2(page);
    await expect(page).toHaveScreenshot('login-v2-mobile.png');
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('login v2 light without a passkey @mobile', async ({ page }) => {
    const { unstubbed } = await openLogin(page, { passkey: false, width: 360 });
    await expectV2(page, { passkey: false });
    await expect(page).toHaveScreenshot('login-v2-no-passkey-mobile.png');
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('login v2 waiting for the device @mobile', async ({ page }) => {
    const { unstubbed } = await openLogin(page, { device: 'wait' });
    await page.getByRole('button', { name: 'Log in with a passkey' }).click();
    await expect(page.getByRole('button', { name: 'Waiting for your device…' })).toBeDisabled();
    await expect(page.getByRole('button', { name: 'Log in', exact: true })).toBeDisabled();
    await expect(page).toHaveScreenshot('login-v2-passkey-waiting-mobile.png');
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('login v2 passkey error toast @mobile', async ({ page }) => {
    const { unstubbed } = await openLogin(page, { dark: true, device: 'cancel', width: 360 });
    const toast = await failPasskey(page);
    const box = await toast.boundingBox();
    const height = page.viewportSize().height;
    expect(Math.round(height - (box.y + box.height))).toBe(12);
    expect(Math.round(box.x)).toBe(12);
    await expect(page).toHaveScreenshot('login-v2-passkey-error-mobile.png');
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('login v1 without the mark @mobile', async ({ page }) => {
    const { unstubbed } = await openLogin(page, { mark: false, dark: true, width: 360 });
    await expect(page.locator('#app.design-v2')).toHaveCount(0);
    await expect(page).toHaveScreenshot('login-v1-mobile.png');
    expectEverythingStubbed(unstubbed);
  });
});
