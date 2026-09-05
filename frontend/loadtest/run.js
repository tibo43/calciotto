#!/usr/bin/env node
// One-off load-test driver: NOT part of `npm run test:unit` / `test:visual`.
// It drives the real production build through the browser (not the API
// directly) for N virtual users in parallel: sign up with an invite code,
// select the scheduled match, click Participate, open the group roster on
// the profile page, then log out. Timing per user and a pass/fail summary
// are printed at the end.
//
// Prerequisites: run `go run ./cmd/perfsetup` first (see app/cmd/perfsetup)
// to create a throwaway group + admin + open-signups match, and pass its
// invite code here.
//
// Usage:
//   BASE_URL=https://calciotto.example.com \
//   INVITE_CODE=AB2N7TQR \
//   USERS=50 \
//   node frontend/loadtest/run.js
//
// Env vars (all but INVITE_CODE have a default):
//   BASE_URL     frontend origin to hit               (default http://localhost:4000 — almost
//                                                       certainly wrong for a real load test,
//                                                       set it to the Vercel prod URL)
//   INVITE_CODE  the group's invite code               (required, no default — from perfsetup's output)
//   USERS        how many accounts to create           (default 50)
//   EMAIL_DOMAIN domain for the throwaway accounts      (default perfload.test — reserved TLD, never emailed)
//   HEADLESS     "false" to watch the browsers          (default true)

const { chromium } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://localhost:4000';
const INVITE_CODE = process.env.INVITE_CODE;
const USER_COUNT = parseInt(process.env.USERS || '50', 10);
const EMAIL_DOMAIN = process.env.EMAIL_DOMAIN || 'perfload.test';
const HEADLESS = process.env.HEADLESS !== 'false';
const PASSWORD = 'PerfLoad!2026';

if (!INVITE_CODE) {
  console.error('INVITE_CODE is required — copy it from `go run ./cmd/perfsetup`\'s output.');
  process.exit(1);
}

async function runOneUser(browser, index) {
  const name = `Perf User ${index}`;
  const email = `perfload+u${index}@${EMAIL_DOMAIN}`;
  const startedAt = Date.now();
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    // --- Sign up (auto-logs in and redirects to `/` on success) ---
    await page.goto(`${BASE_URL}/signup?invite=${INVITE_CODE}`);
    await page.locator('#name').fill(name);
    await page.locator('#email').fill(email);
    await page.locator('#password').fill(PASSWORD);
    // Prefilled from the ?invite= query param already — filled again
    // defensively in case that behavior ever regresses.
    await page.locator('#invite-code').fill(INVITE_CODE);
    await page.getByRole('button', { name: 'Sign up' }).click();
    await page.waitForURL((url) => url.pathname === '/', { timeout: 30_000 });
    await page.locator('.nav-menu').waitFor({ state: 'visible', timeout: 30_000 });

    // --- Sign up for the match ---
    // The group has exactly one match (created by cmd/perfsetup), and
    // MatchesPanel auto-selects the newest one on load, so it should already
    // be showing the inline sign-up panel; click the card defensively in
    // case more than one match exists in this group by the time this runs.
    const signupPanel = page.locator('.signup-inline');
    if (!(await signupPanel.isVisible().catch(() => false))) {
      await page.locator('.match-card-horizontal.scheduled').first().click();
      await signupPanel.waitFor({ state: 'visible', timeout: 15_000 });
    }
    await signupPanel.getByRole('button', { name: 'Participate' }).click();
    // No confirmation toast to key off reliably — the count badge or state
    // label updating is the observable effect; give it a moment to settle.
    await page.waitForTimeout(1000);

    // --- Profile: open the group card to view the roster ---
    await page.goto(`${BASE_URL}/profile`);
    await page.locator('.group-card-horizontal').first().click();
    await page.locator('.roster-panel-container').waitFor({ state: 'visible', timeout: 15_000 });
    await page.locator('.member-row').first().waitFor({ state: 'visible', timeout: 15_000 });

    // --- Log out ---
    await page.getByRole('button', { name: 'Log out' }).click();
    await page.waitForURL((url) => url.pathname === '/login', { timeout: 15_000 });

    return { index, ok: true, ms: Date.now() - startedAt };
  } catch (err) {
    return { index, ok: false, ms: Date.now() - startedAt, error: err.message };
  } finally {
    await context.close();
  }
}

function summarize(results) {
  const durations = results.map((r) => r.ms).sort((a, b) => a - b);
  const pct = (p) => durations[Math.min(durations.length - 1, Math.floor((p / 100) * durations.length))];
  const failures = results.filter((r) => !r.ok);

  console.log('\n=== Load test summary ===');
  console.log(`Users:      ${results.length}`);
  console.log(`Succeeded:  ${results.length - failures.length}`);
  console.log(`Failed:     ${failures.length}`);
  console.log(`Duration:   p50=${pct(50)}ms  p95=${pct(95)}ms  max=${durations[durations.length - 1]}ms`);
  if (failures.length > 0) {
    console.log('\nFailures:');
    for (const f of failures) {
      console.log(`  user ${f.index}: ${f.error}`);
    }
  }
}

async function main() {
  console.log(`Running ${USER_COUNT} virtual users in parallel against ${BASE_URL} ...`);
  const browser = await chromium.launch({ headless: HEADLESS });
  try {
    const results = await Promise.all(
      Array.from({ length: USER_COUNT }, (_, i) => runOneUser(browser, i + 1))
    );
    summarize(results);
    process.exitCode = results.some((r) => !r.ok) ? 1 : 0;
  } finally {
    await browser.close();
  }
}

main();
