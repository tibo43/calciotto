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

// A number with exactly `digits` decimals and no grouping: odds, ratings
// (« 2.35 » in English, « 2,35 » in French). toFixed's own rounding, with
// the language's decimal separator, so English is toFixed's output exactly.
export const formatDecimal = (value, digits = 2) => {
  const decimal = new Intl.NumberFormat(activeTag()).formatToParts(1.5).find((part) => part.type === 'decimal');
  return Number(value).toFixed(digits).replace('.', decimal ? decimal.value : '.');
};

// The name of the weekday or month of a Date on its own, in the active
// language: 'short' (Sun, Sep in English), 'long' (Sunday, September)...
// For labels whose order comes from a message (date.* in the bundles) rather
// than from Intl's own patterns. undefined for an invalid Date.
const namePart = (date, field, width) => {
  if (!(date instanceof Date) || Number.isNaN(date.getTime())) return undefined;
  const part = new Intl.DateTimeFormat(activeTag(), { [field]: width })
    .formatToParts(date).find((item) => item.type === field);
  return part && part.value;
};

export const weekdayName = (date, width = 'short') => namePart(date, 'weekday', width);
export const monthName = (date, width = 'short') => namePart(date, 'month', width);

// 'one', 'other'... for the active language (0 is 'one' in French, 'other' in
// English).
export const pluralCategory = (n) => categoryOf(n, getLocale());

// Picks the form for n out of { one, other, ... }, falling back to `other`:
//   plural(n, { one: 'point', other: 'points' })
// For texts written as messages, prefer vue-i18n's own `$t(key, n)`.
export const plural = (n, forms) => forms[pluralCategory(n)] ?? forms.other;
