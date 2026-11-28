// The vue-i18n instance (composition API, no legacy mode). Components use
// `useI18n()` or `$t`; which language is active is owned by
// src/services/locale.js, not set from here.
import { createI18n } from 'vue-i18n';
import { LOCALES, DEFAULT_LOCALE, messages } from '@/locales';
import { pluralCategory, intlTag } from '@/services/pluralRules';

// vue-i18n picks a plural form from the number of forms written in a message
// ("a | b" or "zero | one | other"); its built-in choice is English-shaped, so
// 0 counts as plural. French says "0 point", for instance. Every language
// goes through Intl.PluralRules instead, the same rule the Intl helper uses.
const pluralRules = Object.fromEntries(LOCALES.map(({ code }) => [
  code,
  (choice, choicesLength) => {
    const category = pluralCategory(choice, code);
    if (choicesLength === 2) return category === 'one' ? 0 : 1;
    if (choicesLength === 3) return choice === 0 ? 0 : (category === 'one' ? 1 : 2);
    return Math.min(choice, choicesLength - 1);
  },
]));

export const i18n = createI18n({
  legacy: false,
  locale: DEFAULT_LOCALE,
  fallbackLocale: DEFAULT_LOCALE,
  messages,
  pluralRules,
});

// Counts are shown the way the language writes numbers: « 1,5 » in French
// and German, « 1.5 » in English. vue-i18n puts the plural count into the
// message as the plain JavaScript number, so every message that reads {n} (or
// {count}) would print a dot whatever the language. The translate function
// hands it the number already formatted by Intl.NumberFormat, in the language
// the message is translated into: t(key, 1.5), t(key, named, { plural: 1.5 })
// and t(key, { n: 1.5 }) all do. The plural form is still chosen from the
// number itself, not from its text.
const composer = i18n.global;
const translate = composer.t;
const isNamed = (value) => value !== null && typeof value === 'object' && !Array.isArray(value);
const formatCount = (value, locale) => new Intl.NumberFormat(intlTag(locale)).format(value);

composer.t = (key, arg2, arg3) => {
  const options = isNamed(arg3) ? arg3 : {};
  const explicitPlural = [arg2, arg3, options.plural].find((value) => typeof value === 'number');
  const named = isNamed(arg2) ? arg2 : {};
  const isCount = (name) => typeof named[name] === 'number';
  if (explicitPlural === undefined && !isCount('n') && !isCount('count')) return translate(key, arg2, arg3);
  const locale = options.locale || composer.locale.value;
  const counted = { ...named };
  ['n', 'count'].forEach((name) => {
    if (isCount(name)) counted[name] = formatCount(named[name], locale);
    else if (explicitPlural !== undefined && named[name] === undefined) counted[name] = formatCount(explicitPlural, locale);
  });
  // Once {n} and {count} are text, vue-i18n can no longer pick the plural
  // form from them, so it is passed explicitly, from the original number.
  const plural = explicitPlural ?? (isCount('n') ? named.n : named.count);
  return translate(key, counted, { ...options, plural });
};

// For the services (outside components): the global translate function, in
// the active language at the time of the call.
export const t = (...args) => i18n.global.t(...args);

// A note or error a page keeps on screen until the next action. A translated
// string stored in data would stay in the language it was written in, so the
// page stores the key instead, as messageKey(key, named), and translates it
// when it renders, with showText. Text from the server is stored as it is
// and shown unchanged.
export const messageKey = (key, named) => ({ key, named });
export const showText = (value) => {
  if (value && typeof value === 'object') return value.named === undefined ? t(value.key) : t(value.key, value.named);
  return value || '';
};

export default i18n;
