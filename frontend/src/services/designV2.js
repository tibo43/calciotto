import { getMyFeatures, getToken } from './api';

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

// The answer to GET /players/me/features for the current session, kept in
// memory only: nothing about the redesign is written to the browser. The
// entry is keyed by the token it was fetched with, so a different session
// (logout, then another account; a 401 that ended this one) never reads
// the previous one's answer, even if nothing cleared it explicitly.
let session = { token: null, promise: null, enabled: null };

// The flag for this session, from the server. Resolves false with no token
// and on a failed request (today's design, never a hang); a failure also
// forgets the entry so the next call asks again. Concurrent callers (the
// router guards and App.vue) share one request.
export function loadDesignV2() {
  const token = getToken();
  if (!token) {
    return Promise.resolve(false);
  }
  if (session.token === token && session.promise) {
    return session.promise;
  }
  const entry = { token, promise: null, enabled: null };
  entry.promise = getMyFeatures()
    .then((features) => {
      entry.enabled = hasDesignV2(features);
      return entry.enabled;
    })
    .catch((error) => {
      console.error('Error fetching features:', error);
      if (session === entry) {
        session = { token: null, promise: null, enabled: null };
      }
      return false;
    });
  session = entry;
  return entry.promise;
}

// The answer already known for this session, synchronously: true or false
// once the response has landed, null while it hasn't (or for another token).
export function knownDesignV2() {
  const token = getToken();
  return token && session.token === token ? session.enabled : null;
}

export function resetDesignV2() {
  session = { token: null, promise: null, enabled: null };
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
