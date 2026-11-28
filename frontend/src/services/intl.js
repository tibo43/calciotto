// Dates, numbers and plurals in the active language (src/services/locale.js).
//
// The formatters are built per call from the language active at that moment,
// so a page that re-renders after a language change picks the new one up with
// nothing to invalidate. Options are Intl's own; nothing is added by default,
// each language's own conventions apply (24 h clocks in French, Italian,
// German and Spanish, 12 h in English).
//
// English formats with en-US, the tag of the 'en-US' calls elsewhere in the
// app, so moving one of them here leaves its output as it is.
import { getLocale } from '@/services/locale';
import { intlTag, pluralCategory as categoryOf } from '@/services/pluralRules';

// The BCP 47 tag of the active language (en -> en-US, fr -> fr-FR...).
export const activeTag = () => intlTag(getLocale());

// A Date, a timestamp or an RFC3339 string; '' for empty or unparseable input
// rather than "Invalid Date".
//
// Not a bare calendar day such as Match.Date ('2026-09-06'): new Date() reads a
// date-only string as UTC midnight, so it would show the day before anywhere
// west of Greenwich. Parse it with parseCalendarDay (services/datetime.js)
// first and pass the resulting Date.
export const formatDate = (value, options) => {
  if (value === null || value === undefined || value === '') return '';
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return new Intl.DateTimeFormat(activeTag(), options).format(date);
};

export const formatNumber = (value, options) => new Intl.NumberFormat(activeTag(), options).format(value);

// 'one', 'other'... for the active language (0 is 'one' in French, 'other' in
// English).
export const pluralCategory = (n) => categoryOf(n, getLocale());

// Picks the form for n out of { one, other, ... }, falling back to `other`:
//   plural(n, { one: 'point', other: 'points' })
// For texts written as messages, prefer vue-i18n's own `$t(key, n)`.
export const plural = (n, forms) => forms[pluralCategory(n)] ?? forms.other;
