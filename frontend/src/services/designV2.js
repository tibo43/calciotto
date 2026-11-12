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

// Display, labels, body. Loaded only once the feature is actually on (see
// ensureDesignV2Fonts); the families are referenced from design-v2.css and
// nowhere else. Big Shoulders Display is the condensed sports gothic the
// mock-up's titles and scores sit closest to; Space Mono is the tracked
// label; DM Sans is the quiet geometric face for names and buttons.
export const DESIGN_V2_FONTS_HREF = 'https://fonts.googleapis.com/css2?family=Big+Shoulders+Display:wght@700;800&family=DM+Sans:wght@400;500;600&family=Space+Mono&display=swap';

const FONT_LINK_ID = 'design-v2-fonts';

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

// Inject the stylesheet the first time this viewer actually has the feature.
// Players without it never request the font files. A second call is a no-op,
// so a route change doesn't add another link.
export function ensureDesignV2Fonts() {
  if (typeof document === 'undefined' || !document.head) {
    return;
  }
  if (document.getElementById(FONT_LINK_ID)) {
    return;
  }
  const link = document.createElement('link');
  link.id = FONT_LINK_ID;
  link.rel = 'stylesheet';
  link.href = DESIGN_V2_FONTS_HREF;
  document.head.appendChild(link);
}

export function removeDesignV2Fonts() {
  if (typeof document === 'undefined') {
    return;
  }
  const link = document.getElementById(FONT_LINK_ID);
  if (link) {
    link.remove();
  }
}
