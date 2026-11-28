// The languages the app can be shown in, and the message bundles that exist.
//
// LOCALES lists every language the product commits to, in the order the
// Account language selector shows them (English, Français, Italiano, Deutsch,
// Español), each with its name in its own language and the BCP 47 tag the Intl
// helpers format with. It is *not* the list of languages the app speaks yet:
// a language is usable once its bundle (src/locales/<code>.json) exists and is
// not empty, and the glob below picks bundles up by file name. Adding French
// is therefore just adding fr.json, with no code change here.
//
// "en" maps to en-US, not plain "en", so that English dates and numbers keep
// the exact output of the 'en-US' formatters the app already uses.
export const LOCALES = [
  { code: 'en', name: 'English', tag: 'en-US' },
  { code: 'fr', name: 'Français', tag: 'fr-FR' },
  { code: 'it', name: 'Italiano', tag: 'it-IT' },
  { code: 'de', name: 'Deutsch', tag: 'de-DE' },
  { code: 'es', name: 'Español', tag: 'es-ES' },
];

// The language of the texts that every other bundle falls back to, and the one
// used when nothing else matches.
export const DEFAULT_LOCALE = 'en';

// code -> messages, from every src/locales/<code>.json present.
export const messages = Object.fromEntries(
  Object.entries(import.meta.glob('./*.json', { eager: true, import: 'default' }))
    .map(([path, bundle]) => [path.replace(/^\.\/|\.json$/g, ''), bundle]),
);
