// The vue-i18n instance (composition API, no legacy mode). Components use
// `useI18n()` or `$t`; which language is active is owned by
// src/services/locale.js, not set from here.
import { createI18n } from 'vue-i18n';
import { LOCALES, DEFAULT_LOCALE, messages } from '@/locales';
import { pluralCategory } from '@/services/pluralRules';

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

// For the services (outside components): the global translate function, in
// the active language at the time of the call.
export const t = (...args) => i18n.global.t(...args);

export default i18n;
