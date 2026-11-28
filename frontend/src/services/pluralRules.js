// Plural category of a number in a language ('one', 'other', and for some
// languages 'few', 'many'...), from Intl.PluralRules. A module of its own
// because both src/i18n.js (message plurals) and src/services/intl.js (the
// Intl helper) need it, and intl.js depends on i18n.js through the locale
// service.
import { LOCALES, DEFAULT_LOCALE } from '@/locales';

export const intlTag = (code) => (LOCALES.find((locale) => locale.code === code) || LOCALES[0]).tag;

export const pluralCategory = (n, code = DEFAULT_LOCALE) => new Intl.PluralRules(intlTag(code)).select(Number(n));
