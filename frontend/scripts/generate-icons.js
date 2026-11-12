// Renders the installable-app icons in public/icons/ from one source image
// (SVG or PNG). Run through generate-icons.sh, not directly: it needs the
// Chromium that ships in the pinned Playwright image.
//
// Chromium does the rasterising rather than an image library, so there is no
// new dependency for something run once per logo change — @playwright/test is
// already in the tree for the visual tests.
//
// Usage: node scripts/generate-icons.js <source> [--background <css colour>] [--inset <0..1>]
//   --background  fills the transparent parts of the opaque icons (maskable,
//                 apple-touch-icon). Default: the app's primary green.
//   --inset       how much of each opaque icon the source covers (the
//                 transparent ones always use the whole canvas). 1 for a
//                 full-bleed square logo; less for a round or transparent one,
//                 so that Android's maskable crop (a circle of 80% of the
//                 icon) and iOS's rounded corners don't cut into it. Default: 1.

const fs = require('fs');
const path = require('path');
const { chromium } = require('@playwright/test');

const OUT_DIR = path.join(__dirname, '..', 'public', 'icons');

// `opaque` icons get the background colour: iOS shows transparency as black,
// and a maskable icon is by definition cropped out of a full-bleed square.
const ICONS = [
  { file: 'icon-192.png', size: 192, opaque: false },
  { file: 'icon-512.png', size: 512, opaque: false },
  { file: 'icon-maskable-512.png', size: 512, opaque: true },
  { file: 'apple-touch-icon.png', size: 180, opaque: true },
];

function parseArgs(argv) {
  const args = { source: null, background: '#10b981', inset: 1 };
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === '--background') {
      args.background = argv[++i];
    } else if (argv[i] === '--inset') {
      args.inset = Number(argv[++i]);
    } else {
      args.source = argv[i];
    }
  }
  if (!args.source || !(args.inset > 0 && args.inset <= 1)) {
    throw new Error('usage: generate-icons.js <source> [--background <colour>] [--inset <0..1>]');
  }
  return args;
}

function dataUrl(file) {
  const mime = path.extname(file).toLowerCase() === '.svg' ? 'image/svg+xml' : 'image/png';
  return `data:${mime};base64,${fs.readFileSync(file).toString('base64')}`;
}

async function main() {
  const { source, background, inset } = parseArgs(process.argv.slice(2));
  const src = dataUrl(source);
  const browser = await chromium.launch();
  try {
    for (const icon of ICONS) {
      const page = await browser.newPage({ viewport: { width: icon.size, height: icon.size } });
      const imageSize = Math.round(icon.size * (icon.opaque ? inset : 1));
      await page.setContent(`
        <html><body style="margin:0;width:${icon.size}px;height:${icon.size}px;display:flex;
          align-items:center;justify-content:center;background:${icon.opaque ? background : 'transparent'}">
          <img src="${src}" style="width:${imageSize}px;height:${imageSize}px;object-fit:contain">
        </body></html>`);
      await page.locator('img').evaluate((img) => img.decode());
      await page.screenshot({ path: path.join(OUT_DIR, icon.file), omitBackground: !icon.opaque });
      await page.close();
      console.log(`wrote public/icons/${icon.file}`);
    }
  } finally {
    await browser.close();
  }
}

main().catch((err) => {
  console.error(err.message);
  process.exit(1);
});
