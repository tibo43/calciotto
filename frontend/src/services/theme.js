// Whether dark mode is currently active. Meant for components that are
// Teleport'd to <body> (escaping #app, so they can't inherit the .dark-mode
// class App.vue toggles there via CSS custom property inheritance) and have
// no direct prop path back to App.vue to be handed the flag instead — e.g. a
// dialog opened from a routed view rather than straight from App.vue's own
// template.
//
// Queries by class, not by #app's id: public/index.html's own mount
// container also carries id="app", and Vue 3 doesn't replace that
// container — it nests its rendered root (the one actually carrying
// .dark-mode) inside it, leaving two elements sharing the id. That means
// getElementById('app') can return the wrong (always class-less) one; only
// one element in the document ever carries .dark-mode.
export function isDarkModeActive() {
  return document.querySelector('.dark-mode') !== null;
}

// The page background in each theme (global-styles.css's --bg-primary), which
// is also what the phone's status bar should show above it.
const THEME_COLORS = { light: '#ffffff', dark: '#0f172a' };

// Points index.html's <meta name="theme-color"> at the current theme, so the
// status bar of an installed app (and Chrome's toolbar on Android) follows the
// in-app light/dark toggle rather than staying white over a dark page.
export function syncThemeColorMeta(isDark) {
  const meta = document.querySelector('meta[name="theme-color"]');
  if (meta) {
    meta.setAttribute('content', isDark ? THEME_COLORS.dark : THEME_COLORS.light);
  }
}
