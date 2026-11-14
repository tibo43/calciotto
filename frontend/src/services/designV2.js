// The redesign ships behind the same hidden-feature list as the install card
// and push notifications (GET /players/me/features). The key matches
// models.FeatureDesignV2. App.vue is the only reader: it paints a
// `design-v2` class on the app root, and every v2 rule hangs off that class
// so a player without the feature — and every screen not restyled yet —
// keeps today's CSS.

export const DESIGN_V2_FEATURE = 'design-v2';

// Per-browser memory of the last answer, so the class can be on for the
// first paint instead of flashing the current design while the features
// request is in flight. Not authoritative: the response overwrites it, and
// logout / a 401 / account deletion clear it. A viewer convenience only.
const STORAGE_KEY = 'calciotto-design-v2';
const STORAGE_ON = '1';

// Display, labels, body. The files live under public/fonts/v2 and are
// injected only once the feature is actually on (see ensureDesignV2Fonts),
// so a player without it never downloads them. Big Shoulders Display is the
// condensed display face, Space Mono the tracked label, DM Sans the body.
const FONT_STYLE_ID = 'design-v2-fonts';

export const DESIGN_V2_FONT_CSS = `
@font-face{font-family:"Big Shoulders Display";font-style:normal;font-weight:100 900;font-display:swap;src:url("/fonts/v2/BigShouldersDisplay-normal-800-latin.woff2") format("woff2");}
@font-face{font-family:"DM Sans";font-style:normal;font-weight:100 900;font-display:swap;src:url("/fonts/v2/DMSans-normal-400-latin.woff2") format("woff2");}
@font-face{font-family:"Space Mono";font-style:normal;font-weight:400;font-display:swap;src:url("/fonts/v2/SpaceMono-normal-400-latin.woff2") format("woff2");}
@font-face{font-family:"Space Mono";font-style:normal;font-weight:700;font-display:swap;src:url("/fonts/v2/SpaceMono-normal-700-latin.woff2") format("woff2");}
`;

export function readDesignV2Cached() {
  try {
    return localStorage.getItem(STORAGE_KEY) === STORAGE_ON;
  } catch {
    // Storage can throw (blocked, or a privacy mode that disables it).
    // Missing cache means "don't guess" — the class waits for the API.
    return false;
  }
}

export function writeDesignV2Cached(enabled) {
  try {
    if (enabled) {
      localStorage.setItem(STORAGE_KEY, STORAGE_ON);
    } else {
      localStorage.removeItem(STORAGE_KEY);
    }
  } catch {
    // Same failure mode as the read. This view still follows the API;
    // the next load just can't skip the flash.
  }
}

export function clearDesignV2Cached() {
  writeDesignV2Cached(false);
}

export function hasDesignV2(features) {
  return Array.isArray(features) && features.includes(DESIGN_V2_FEATURE);
}

// Inject the @font-face rules the first time this viewer actually has the
// feature. Players without it never request the font files. A second call
// is a no-op, so a route change doesn't add another style tag.
export function ensureDesignV2Fonts() {
  if (typeof document === 'undefined' || !document.head) {
    return;
  }
  if (document.getElementById(FONT_STYLE_ID)) {
    return;
  }
  const style = document.createElement('style');
  style.id = FONT_STYLE_ID;
  style.textContent = DESIGN_V2_FONT_CSS;
  document.head.appendChild(style);
}

export function removeDesignV2Fonts() {
  if (typeof document === 'undefined') {
    return;
  }
  const style = document.getElementById(FONT_STYLE_ID);
  if (style) {
    style.remove();
  }
}
