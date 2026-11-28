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
import {
  detectLocale, getLocale, getLocaleSource, initLocale, matchLocale, onLocaleChange, SOURCE_EXPLICIT, SOURCE_PHONE,
} from './locale';

let loaded = null; // { key, promise }: one read per session
let changes = 0; // how many times the player chose since start-up
let savesDone = 0; // how many saves succeeded since start-up

// The language the server holds (as the locale service reports it: language
// and source): what a failed save goes back to. Updated by every read, every
// reset and every save that succeeded.
let confirmed = null;
const remember = () => { confirmed = { locale: getLocale(), source: getLocaleSource() }; };

// Back to a language without announcing it (nothing to save: it is already
// what the server holds).
const restore = ({ locale, source }) => initLocale({ saved: source === SOURCE_EXPLICIT ? locale : null });

const failureListeners = new Set();

// Told whenever a save failed and the language was put back, so the app can
// say so (LocaleSaveErrorToast, mounted by the app shell, shows a toast on
// whatever page the player is). The listener runs after the language is
// restored: its text comes out in the previous language.
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
  const seenSaves = savesDone;
  const entry = { key, promise: null };
  entry.promise = getMyLocale()
    .then((saved) => {
      // The read was cancelled (a public page reset the language, or another
      // read replaced it) or the session ended: the answer means nothing now.
      if (loaded !== entry || key !== getSessionKey()) return;
      if (seen !== changes) {
        // The player chose meanwhile: that wins on screen. But until their
        // save succeeds, what the server holds is this answer, and a failed
        // save must go back to it (not to the phone's language). A save that
        // already succeeded is newer than this answer: keep it.
        if (seenSaves === savesDone) {
          const locale = saved ? matchLocale(saved) : null;
          confirmed = locale
            ? { locale, source: SOURCE_EXPLICIT }
            : { locale: detectLocale(), source: SOURCE_PHONE };
        }
        return;
      }
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
//
// A pick belongs to the session it was made in: if that session ended before
// its turn came (logout, then another account in the same tab), it is
// dropped, since the token sent would be the next account's.
export function startLocaleSync() {
  remember();
  return onLocaleChange(({ source, locale }) => {
    changes += 1;
    const key = getSessionKey();
    if (!key) return;
    const ticket = changes;
    const pick = { locale, source };
    saving = saving.then(async () => {
      if (getSessionKey() !== key) return;
      try {
        await updateMyLocale(source === SOURCE_PHONE ? null : locale);
        savesDone += 1;
        if (getSessionKey() === key) confirmed = pick;
      } catch (error) {
        console.warn('Could not save the language:', error);
        if (ticket !== changes || getSessionKey() !== key) return;
        restore(confirmed);
        failureListeners.forEach((listener) => listener(error));
      }
    });
  });
}
