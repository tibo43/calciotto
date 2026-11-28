// The player's language, kept on the server (players.locale, NULL = follow the
// phone) and tied to the locale service (services/locale.js) by the two hooks
// it was built with: initLocale({ saved }) to restore a saved choice, and
// onLocaleChange(listener) to save a later one.
//
// There is no local copy of the choice, not in localStorage nor anywhere else
// (Thibaut refused one for the language). So:
//  - signed in: the app starts in the phone's language and switches to the
//    saved one when the answer arrives (main.js waits briefly for it before
//    the first paint, so most of the time nobody sees the switch);
//  - signed out (login, signup, password pages): always the browser's language,
//    a chosen language included. The choice follows the account, not the
//    device, and the selector does not exist on those pages.
import { getMyLocale, updateMyLocale, getSessionKey } from './api';
import { initLocale, onLocaleChange, getLocaleSource, SOURCE_PHONE } from './locale';

let loaded = null; // { key, promise }: one read per session
let changes = 0; // how many times the player chose since start-up

// Reads the saved language once per session and applies it, without
// announcing it (it is a restore: there is nothing to save back). Resolves
// when done; a failed read keeps the phone's language, silently.
export function syncSavedLocale() {
  const key = getSessionKey();
  if (!key) return Promise.resolve();
  if (loaded && loaded.key === key) return loaded.promise;
  const seen = changes;
  const entry = { key, promise: null };
  entry.promise = getMyLocale()
    .then((saved) => {
      // Someone chose a language meanwhile (or the session ended): that wins.
      if (seen !== changes || key !== getSessionKey()) return;
      initLocale({ saved });
    })
    .catch((error) => {
      console.warn('Could not load the saved language:', error);
      if (loaded === entry) loaded = null;
    });
  loaded = entry;
  return entry.promise;
}

// Back to the browser's language when nothing is signed in (logout, expired
// session, any public page). Silent, like the restore.
export function resetSavedLocale() {
  loaded = null;
  if (getLocaleSource() !== SOURCE_PHONE) initLocale();
}

let saving = Promise.resolve();

// Saves every language the player picks, in order: a quick second pick must
// not be overtaken by the first one's slower answer. No toast, no rollback:
// the language is already on screen, and a failed save only means the next
// start follows the old setting.
export function startLocaleSync() {
  return onLocaleChange(({ source, locale }) => {
    changes += 1;
    if (!getSessionKey()) return;
    const value = source === SOURCE_PHONE ? null : locale;
    saving = saving
      .then(() => updateMyLocale(value))
      .catch((error) => console.warn('Could not save the language:', error));
  });
}
