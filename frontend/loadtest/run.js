#!/usr/bin/env node
// One-off load-test driver: NOT part of `npm run test:unit` / `test:visual`.
// It drives the real production build through the browser (not the API
// directly) for N virtual users in parallel: sign up with an invite code,
// select the scheduled match, click Participate, open the group roster on
// the profile page, then log out. Per-step timing for every user is written
// to a JSON report (plus a plain-text summary) under loadtest/reports/, and
// a pass/fail summary prints to the console.
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
//   REPORT_DIR   where to write the report files        (default loadtest/reports, relative to this file)

const fs = require('fs');
const path = require('path');
const { chromium } = require('@playwright/test');

const BASE_URL = process.env.BASE_URL || 'http://localhost:4000';
const INVITE_CODE = process.env.INVITE_CODE;
const USER_COUNT = parseInt(process.env.USERS || '50', 10);
const EMAIL_DOMAIN = process.env.EMAIL_DOMAIN || 'perfload.test';
const HEADLESS = process.env.HEADLESS !== 'false';
const REPORT_DIR = process.env.REPORT_DIR || path.join(__dirname, 'reports');
const PASSWORD = 'PerfLoad!2026';

if (!INVITE_CODE) {
  console.error('INVITE_CODE is required — copy it from `go run ./cmd/perfsetup`\'s output.');
  process.exit(1);
}

// Records the wall-clock duration of each named step for one virtual user,
// so a slow run can be attributed to a specific step (e.g. signup taking
// long under load because of bcrypt hashing) rather than only to a single
// opaque total.
function stepTimer() {
  const steps = {};
  let last = Date.now();
  return {
    mark(name) {
      const now = Date.now();
      steps[name] = now - last;
      last = now;
    },
    steps,
  };
}

async function runOneUser(browser, index) {
  const name = `Perf User ${index}`;
  const email = `perfload+u${index}@${EMAIL_DOMAIN}`;
  const startedAt = Date.now();
  const timer = stepTimer();
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
    timer.mark('signup');

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
    timer.mark('participate');

    // --- Profile: open the group card to view the roster ---
    await page.goto(`${BASE_URL}/profile`);
    await page.locator('.group-card-horizontal').first().click();
    await page.locator('.roster-panel-container').waitFor({ state: 'visible', timeout: 15_000 });
    await page.locator('.member-row').first().waitFor({ state: 'visible', timeout: 15_000 });
    timer.mark('view_roster');

    // --- Log out ---
    await page.getByRole('button', { name: 'Log out' }).click();
    await page.waitForURL((url) => url.pathname === '/login', { timeout: 15_000 });
    timer.mark('logout');

    return { index, email, ok: true, ms: Date.now() - startedAt, steps: timer.steps };
  } catch (err) {
    return { index, email, ok: false, ms: Date.now() - startedAt, steps: timer.steps, error: err.message };
  } finally {
    await context.close();
  }
}

function percentile(sortedDurations, p) {
  return sortedDurations[Math.min(sortedDurations.length - 1, Math.floor((p / 100) * sortedDurations.length))];
}

function buildSummary(results) {
  const durations = results.map((r) => r.ms).sort((a, b) => a - b);
  const failures = results.filter((r) => !r.ok);

  const stepNames = ['signup', 'participate', 'view_roster', 'logout'];
  const perStep = {};
  for (const stepName of stepNames) {
    const values = results.map((r) => r.steps[stepName]).filter((v) => v !== undefined).sort((a, b) => a - b);
    if (values.length > 0) {
      perStep[stepName] = { p50: percentile(values, 50), p95: percentile(values, 95), max: values[values.length - 1] };
    }
  }

  return {
    users: results.length,
    succeeded: results.length - failures.length,
    failed: failures.length,
    total: durations.length > 0 ? { p50: percentile(durations, 50), p95: percentile(durations, 95), max: durations[durations.length - 1] } : null,
    perStep,
    failures: failures.map((f) => ({ index: f.index, email: f.email, error: f.error })),
  };
}

function printSummary(summary) {
  console.log('\n=== Load test summary ===');
  console.log(`Users:      ${summary.users}`);
  console.log(`Succeeded:  ${summary.succeeded}`);
  console.log(`Failed:     ${summary.failed}`);
  if (summary.total) {
    console.log(`Total time: p50=${summary.total.p50}ms  p95=${summary.total.p95}ms  max=${summary.total.max}ms`);
  }
  for (const [stepName, s] of Object.entries(summary.perStep)) {
    console.log(`  ${stepName.padEnd(12)} p50=${s.p50}ms  p95=${s.p95}ms  max=${s.max}ms`);
  }
  if (summary.failures.length > 0) {
    console.log('\nFailures:');
    for (const f of summary.failures) {
      console.log(`  user ${f.index} (${f.email}): ${f.error}`);
    }
  }
}

function writeReport(config, results, summary) {
  fs.mkdirSync(REPORT_DIR, { recursive: true });
  const stamp = new Date().toISOString().replace(/[:.]/g, '-');
  const jsonPath = path.join(REPORT_DIR, `report-${stamp}.json`);
  const txtPath = path.join(REPORT_DIR, `report-${stamp}.txt`);

  fs.writeFileSync(jsonPath, JSON.stringify({ config, summary, results }, null, 2));

  const lines = [
    `Perf load test — ${config.startedAt}`,
    `Target: ${config.baseUrl}`,
    `Users:  ${config.users}`,
    '',
    `Succeeded: ${summary.succeeded} / ${summary.users}`,
    `Failed:    ${summary.failed}`,
    '',
  ];
  if (summary.total) {
    lines.push(`Total duration — p50=${summary.total.p50}ms p95=${summary.total.p95}ms max=${summary.total.max}ms`);
  }
  for (const [stepName, s] of Object.entries(summary.perStep)) {
    lines.push(`  ${stepName}: p50=${s.p50}ms p95=${s.p95}ms max=${s.max}ms`);
  }
  if (summary.failures.length > 0) {
    lines.push('', 'Failures:');
    for (const f of summary.failures) {
      lines.push(`  user ${f.index} (${f.email}): ${f.error}`);
    }
  }
  fs.writeFileSync(txtPath, lines.join('\n') + '\n');

  return { jsonPath, txtPath };
}

async function main() {
  const config = { startedAt: new Date().toISOString(), baseUrl: BASE_URL, users: USER_COUNT };
  console.log(`Running ${USER_COUNT} virtual users in parallel against ${BASE_URL} ...`);
  const browser = await chromium.launch({ headless: HEADLESS });
  try {
    const results = await Promise.all(
      Array.from({ length: USER_COUNT }, (_, i) => runOneUser(browser, i + 1))
    );
    const summary = buildSummary(results);
    printSummary(summary);
    const { jsonPath, txtPath } = writeReport(config, results, summary);
    console.log(`\nReport written to:\n  ${jsonPath}\n  ${txtPath}`);
    process.exitCode = results.some((r) => !r.ok) ? 1 : 0;
  } finally {
    await browser.close();
  }
}

main();
