import { loadFeatureStatus } from './featureStatus';

// The redesign ships behind the same hidden-feature list as the install card
// and push notifications (GET /players/me/features). The key matches
// models.FeatureDesignV2. App.vue paints a `design-v2` class on the app
// root from it (the router's v2-only routes ask too), and every v2 rule
// hangs off that class so a player without the feature — and every screen
// not restyled yet — keeps today's CSS.

export const DESIGN_V2_FEATURE = 'design-v2';

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

export async function loadDesignV2() {
  const { features } = await loadFeatureStatus();
  return hasDesignV2(features);
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
