import { afterEach, describe, expect, it, vi } from 'vitest';

// Fresh modules per test, with the given languages available, then the active
// one set. Spaces are normalised: Intl uses no-break spaces in some locales.
const load = async (code, available = ['fr', 'it', 'de', 'es']) => {
  vi.resetModules();
  const { i18n } = await import('@/i18n');
  available.forEach((c) => i18n.global.setLocaleMessage(c, { language: { label: c } }));
  const locale = await import('@/services/locale');
  const intl = await import('@/services/intl');
  locale.initLocale({ saved: code });
  return { i18n, ...locale, ...intl };
};
const plain = (text) => text.replace(/[\u00a0\u202f]/g, ' ');

const KICKOFF = new Date(2026, 8, 6, 20, 30);

describe('Intl helper', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('English is the en-US output the app already shows', () => {
    it('formats dates like the existing en-US calls', async () => {
      const { formatDate, activeTag } = await load('en');
      expect(activeTag()).toBe('en-US');
      const options = { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' };
      expect(formatDate(KICKOFF, options)).toBe(KICKOFF.toLocaleString('en-US', options));
      expect(plain(formatDate(KICKOFF, options))).toBe('Sun, Sep 6, 2026, 8:30 PM');
      const day = { year: 'numeric', month: 'short', day: 'numeric' };
      expect(formatDate(KICKOFF, day)).toBe(KICKOFF.toLocaleDateString('en-US', day));
    });

    it('formats numbers like en-US', async () => {
      const { formatNumber } = await load('en');
      expect(formatNumber(1234.5)).toBe('1,234.5');
      expect(formatNumber(0.5, { style: 'percent' })).toBe('50%');
    });
  });

  describe('follows the active language', () => {
    it('formats dates in each language, 24 h outside English', async () => {
      const options = { weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' };
      const { formatDate, setLocale } = await load('en');
      setLocale('fr');
      expect(plain(formatDate(KICKOFF, options))).toBe('dim. 6 sept., 20:30');
      setLocale('de');
      expect(plain(formatDate(KICKOFF, options))).toMatch(/^So\.?,? 6\. Sept\.?,? 20:30$/);
      setLocale('it');
      expect(plain(formatDate(KICKOFF, options))).toMatch(/^dom 6 set,? 20:30$/);
      setLocale('es');
      expect(plain(formatDate(KICKOFF, options))).toMatch(/^dom,? 6 sept,? 20:30$/);
    });

    it('formats numbers in each language', async () => {
      const { formatNumber, setLocale } = await load('en');
      setLocale('fr');
      expect(plain(formatNumber(1234.5))).toBe('1 234,5');
      setLocale('de');
      expect(formatNumber(1234.5)).toBe('1.234,5');
      setLocale('it');
      // Italian and Spanish group from five digits up (ICU's minimum grouping).
      expect(formatNumber(1234.5)).toBe('1234,5');
      expect(formatNumber(12345.5)).toBe('12.345,5');
      setLocale('es');
      expect(formatNumber(12345.5)).toBe('12.345,5');
    });

    it('uses the tag of the active language', async () => {
      const { activeTag, setLocale } = await load('en');
      const tags = {};
      ['en', 'fr', 'it', 'de', 'es'].forEach((code) => {
        setLocale(code);
        tags[code] = activeTag();
      });
      expect(tags).toEqual({ en: 'en-US', fr: 'fr-FR', it: 'it-IT', de: 'de-DE', es: 'es-ES' });
    });

    it('formats with the language active at the time of the call', async () => {
      const { formatNumber, setLocale } = await load('de');
      expect(formatNumber(1.5)).toBe('1,5');
      setLocale('en');
      expect(formatNumber(1.5)).toBe('1.5');
    });
  });

  describe('dates with no value', () => {
    it('gives an empty string, not "Invalid Date"', async () => {
      const { formatDate } = await load('en');
      expect(formatDate('')).toBe('');
      expect(formatDate(null)).toBe('');
      expect(formatDate(undefined)).toBe('');
      expect(formatDate('not a date')).toBe('');
      expect(formatDate(new Date('x'))).toBe('');
    });

    it('accepts an RFC3339 string and a timestamp', async () => {
      const { formatDate } = await load('en');
      const options = { year: 'numeric', month: 'short', day: 'numeric' };
      expect(formatDate('2026-09-06T20:30:00+02:00', options)).toBe(formatDate(new Date('2026-09-06T20:30:00+02:00'), options));
      expect(formatDate(KICKOFF.getTime(), options)).toBe(formatDate(KICKOFF, options));
    });
  });

  describe('plurals', () => {
    it('English: 1 is one, 0 and 2 are other', async () => {
      const { pluralCategory, plural } = await load('en');
      expect(pluralCategory(1)).toBe('one');
      expect(pluralCategory(0)).toBe('other');
      expect(pluralCategory(2)).toBe('other');
      expect(plural(1, { one: 'pt', other: 'pts' })).toBe('pt');
      expect(plural(0, { one: 'pt', other: 'pts' })).toBe('pts');
      expect(plural(13, { one: 'pt', other: 'pts' })).toBe('pts');
    });

    it('French: 0 and 1 are one, 2 is other', async () => {
      const { pluralCategory, setLocale } = await load('en');
      setLocale('fr');
      expect(pluralCategory(0)).toBe('one');
      expect(pluralCategory(1)).toBe('one');
      expect(pluralCategory(2)).toBe('other');
    });

    it('falls back to other when a category has no form', async () => {
      const { plural } = await load('en');
      expect(plural(1, { other: 'pts' })).toBe('pts');
    });

    it('drives vue-i18n message plurals, with the same rule', async () => {
      const { i18n, setLocale } = await load('en');
      i18n.global.setLocaleMessage('fr', { unit: { points: '{n} pt | {n} pts' } });
      const t = i18n.global.t;
      expect(t('unit.points', { n: 1 }, { plural: 1 })).toBe('1 pt');
      expect(t('unit.points', { n: 0 }, { plural: 0 })).toBe('0 pts');
      expect(t('unit.points', { n: 7 }, { plural: 7 })).toBe('7 pts');
      setLocale('fr');
      expect(t('unit.points', { n: 0 }, { plural: 0 })).toBe('0 pt');
      expect(t('unit.points', { n: 1 }, { plural: 1 })).toBe('1 pt');
      expect(t('unit.points', { n: 2 }, { plural: 2 })).toBe('2 pts');
    });
  });
});
