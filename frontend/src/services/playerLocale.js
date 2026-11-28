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
import { getLocale, getLocaleSource, initLocale, onLocaleChange, SOURCE_EXPLICIT, SOURCE_PHONE } from './locale';

let loaded = null; // { key, promise }: one read per session
let changes = 0; // how many times the player chose since start-up

// The language the server holds (as the locale service reports it: language
// and source): what a failed save goes back to. Updated by every read, every
// reset and every save that succeeded.
let confirmed = null;
const remember = () => { confirmed = { locale: getLocale(), source: getLocaleSource() }; };

// Back to a language without announcing it (nothing to save: it is already
// what the server holds).
const restore = ({ locale, source }) => initLocale({ saved: source === SOURCE_EXPLICIT ? locale : null });

const failureListeners = new Set();

// Told whenever a save failed and the language was put back, so the page can
// say so (the Account selector shows a toast). The listener runs after the
// language is restored: its text comes out in the previous language.
export const onLocaleSaveError = (listener) => {
  failureListeners.add(listener);
  return () => failureListeners.delete(listener);
};

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
      remember();
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
  remember();
}

let saving = Promise.resolve();

// Saves every language the player picks, in order: a quick second pick must
// not be overtaken by the first one's slower answer.
//
// A save that fails puts the language back to what the server holds (the
// interface text and the selector's checkmark both follow the locale service)
// and tells onLocaleSaveError listeners. Only the latest pick can do that: a
// failure of an older save is dropped, because the newer pick has its own save
// queued behind it and decides what the server ends up with. A success is
// silent.
export function startLocaleSync() {
  remember();
  return onLocaleChange(({ source, locale }) => {
    changes += 1;
    if (!getSessionKey()) return;
    const ticket = changes;
    const pick = { locale, source };
    saving = saving
      .then(() => updateMyLocale(source === SOURCE_PHONE ? null : locale))
      .then(() => { confirmed = pick; })
      .catch((error) => {
        console.warn('Could not save the language:', error);
        if (ticket !== changes) return;
        restore(confirmed);
        failureListeners.forEach((listener) => listener(error));
      });
  });
}
