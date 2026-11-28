// The language the app is shown in: which one is active, and why.
//
// Two sources:
//  - 'phone': nothing chosen, so the first language of navigator.languages
//    the app speaks, else English. Follows the phone if it changes.
//  - 'explicit': the player picked one (setLocale).
//
// Not persisted anywhere yet, and on purpose not in localStorage either: an
// explicit choice lasts until the page is reloaded. The saved choice will be
// the player's own server-side setting (players.locale, NULL = follow the
// phone); the hooks for it are initLocale({ saved }), which restores a saved
// choice without announcing it, and onLocaleChange(listener), which is told
// of every later change and is where the save will be done.
//
// Pages that are not signed in never choose: they stay on the browser's
// language.
import { reactive, readonly } from 'vue';
import { i18n } from '@/i18n';
import { LOCALES, DEFAULT_LOCALE } from '@/locales';

export const SOURCE_PHONE = 'phone';
export const SOURCE_EXPLICIT = 'explicit';

const state = reactive({ locale: DEFAULT_LOCALE, source: SOURCE_PHONE });
const listeners = new Set();

// Read-only view for components (the language selector will bind to it).
export const localeState = readonly(state);

// "fr", "fr-CA", "FR_ca" -> "fr". Anything else -> ''.
const baseLanguage = (tag) => {
  const match = /^([a-z]{2,3})(?:[-_]|$)/i.exec(String(tag || '').trim());
  return match ? match[1].toLowerCase() : '';
};

// The languages the app speaks: those of LOCALES that have messages. Read from
// the i18n instance rather than from the bundles on disk, so a bundle added at
// runtime counts too (the tests do this).
export const availableLocales = () => LOCALES
  .map((locale) => locale.code)
  .filter((code) => {
    const bundle = i18n.global.getLocaleMessage(code);
    return bundle && Object.keys(bundle).length > 0;
  });

// The supported language a tag stands for ("fr-CA" -> "fr"), else null.
export const matchLocale = (tag, available = availableLocales()) => {
  const code = baseLanguage(tag);
  return available.includes(code) ? code : null;
};

const browserLanguages = () => {
  if (typeof navigator === 'undefined') return [];
  if (Array.isArray(navigator.languages) && navigator.languages.length) return navigator.languages;
  return navigator.language ? [navigator.language] : [];
};

// The first language of the list that the app speaks, else English. The list
// is the browser's order of preference, so an unsupported first choice (pt-BR)
// gives way to the next one (fr) rather than straight to English.
export const detectLocale = (languages = browserLanguages(), available = availableLocales()) => {
  for (const tag of languages) {
    const code = matchLocale(tag, available);
    if (code) return code;
  }
  return DEFAULT_LOCALE;
};

export const getLocale = () => state.locale;
export const getLocaleSource = () => state.source;

// Every place that must follow the language: vue-i18n, <html lang> (screen
// readers, hyphenation and the browser's own translation offer read it).
const apply = (locale, source) => {
  state.locale = locale;
  state.source = source;
  i18n.global.locale.value = locale;
  if (typeof document !== 'undefined') document.documentElement.setAttribute('lang', locale);
};

const announce = () => {
  const change = { locale: state.locale, source: state.source };
  listeners.forEach((listener) => listener(change));
};

// Called whenever the language changes after start-up. Returns the function
// that removes the listener.
export const onLocaleChange = (listener) => {
  listeners.add(listener);
  return () => listeners.delete(listener);
};

// The player picks a language. An unsupported code is refused (returns false)
// and changes nothing.
export const setLocale = (code) => {
  const locale = matchLocale(code);
  if (!locale) return false;
  apply(locale, SOURCE_EXPLICIT);
  announce();
  return true;
};

// The player goes back to the phone's language (the first line of the
// selector).
export const followPhone = () => {
  apply(detectLocale(), SOURCE_PHONE);
  announce();
};

let watchingPhone = false;

// A phone whose language changes while the app is open (rare, but it is one
// event): followed only while nothing was chosen.
const onLanguageChange = () => {
  if (state.source !== SOURCE_PHONE) return;
  const locale = detectLocale();
  if (locale === state.locale) return;
  apply(locale, SOURCE_PHONE);
  announce();
};

// At start-up, before the app mounts. `saved` is a choice kept elsewhere
// (later the player's setting); null or unsupported means follow the phone.
// Nothing is announced: this is a restore, there is nothing to save.
export const initLocale = ({ saved = null } = {}) => {
  const locale = saved ? matchLocale(saved) : null;
  if (locale) apply(locale, SOURCE_EXPLICIT);
  else apply(detectLocale(), SOURCE_PHONE);
  if (!watchingPhone && typeof window !== 'undefined') {
    window.addEventListener('languagechange', onLanguageChange);
    watchingPhone = true;
  }
};
