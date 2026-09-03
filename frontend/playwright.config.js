// Visual regression tests. Deliberately separate from `npm run test:unit`
// (Jest/jsdom, which renders no pixels at all) and never run on the host — see
// the `test:visual` script in package.json.
//
// **Baselines are platform-specific.** Font rasterisation and antialiasing
// differ between macOS and the Ubuntu CI runner, so a screenshot taken on a Mac
// will never match one taken in CI, no matter how correct the page is. The fix
// is not a fuzzy threshold — that just hides real regressions — but generating
// and verifying baselines in the *same* environment: the pinned
// mcr.microsoft.com/playwright image, used both by `npm run test:visual`
// locally (through Docker) and by the `visual` job in CI (as the job's
// container). That is also why there is a single baseline set below rather than
// Playwright's default per-platform naming: there is only ever one platform
// producing them.
//
// The image tag must be kept in step with the @playwright/test version in
// package.json — the image ships the browser build that its own version expects.

const { defineConfig, devices } = require('@playwright/test');

const PORT = Number(process.env.VISUAL_SERVER_PORT || 4173);

module.exports = defineConfig({
  testDir: './tests/visual',
  // One baseline set, no {platform} segment: see the note above.
  // {projectName} is not decoration: without it the desktop and mobile projects
  // write the same file for a shared test and silently overwrite each other's
  // baseline. The platform is still absent, because only one ever produces them.
  snapshotPathTemplate: '{testDir}/__screenshots__/{testFileName}/{projectName}/{arg}{ext}',
  // Fail rather than silently write a new baseline when one is missing. Without
  // this, a renamed screenshot quietly creates a fresh baseline in CI and the
  // test passes having compared nothing.
  ignoreSnapshots: false,
  forbidOnly: !!process.env.CI,
  // No retries: a visual test that only passes on the second run is telling us
  // the page isn't deterministic yet, and retrying would hide exactly that.
  retries: 0,
  workers: process.env.CI ? 2 : undefined,
  reporter: process.env.CI ? [['html', { open: 'never' }], ['github']] : [['list']],

  expect: {
    toHaveScreenshot: {
      // Antialiasing still differs by a hair between runs of the same browser
      // build (GPU-less rendering isn't bit-exact), so a handful of pixels are
      // tolerated — but as an absolute count, not a ratio: a ratio scales with
      // the screenshot and would let a small broken component through on a tall
      // page.
      maxDiffPixels: 120,
      // `animations: 'disabled'` and `caret: 'hide'` are Playwright's defaults
      // here; stated explicitly because they are load-bearing for stability.
      animations: 'disabled',
      caret: 'hide',
      scale: 'css',
    },
  },

  use: {
    baseURL: `http://127.0.0.1:${PORT}`,
    // Pinned so a baseline can never shift because of a device-pixel-ratio or
    // locale difference between the machine that generated it and CI.
    deviceScaleFactor: 1,
    locale: 'en-US',
    timezoneId: 'Europe/Paris',
    colorScheme: 'light',
    // 'on' rather than 'retain-on-failure' for both: on a passing run the HTML
    // report otherwise shows only a bare list of step titles, with no way to
    // see what the page actually looked like — `toHaveScreenshot` itself only
    // attaches its expected/actual/diff triptych when the comparison fails, so
    // a green run leaves nothing to inspect either. Trace gives a full
    // timeline (DOM snapshot, console, network at every action) and video
    // gives a literal recording — together, "what did this test actually do"
    // is answerable from the artifact alone, not only from a diff. The suite
    // is 12 short tests (a few seconds total), so the extra recording
    // overhead and artifact size are negligible; this would be worth revisiting
    // to 'retain-on-failure' if the suite grows enough for that to change.
    trace: 'on',
    video: 'on',
  },

  projects: [
    {
      name: 'desktop',
      use: { ...devices['Desktop Chrome'], viewport: { width: 1280, height: 900 } },
      // Without this the desktop project also picks up the @mobile tests and
      // photographs them at 1280px wide, which is the opposite of their point.
      grepInvert: /@mobile/,
    },
    {
      // The app is mobile-first (horizontal card carousels, bars that stack), so
      // the narrow layout is worth pinning too — but only for the pages whose
      // layout actually changes, hence the tag rather than a second full run.
      name: 'mobile',
      use: { ...devices['Desktop Chrome'], viewport: { width: 390, height: 844 } },
      grep: /@mobile/,
    },
  ],

  webServer: {
    command: 'node tests/visual/serve-dist.js',
    url: `http://127.0.0.1:${PORT}`,
    reuseExistingServer: !process.env.CI,
    timeout: 30_000,
  },
});
