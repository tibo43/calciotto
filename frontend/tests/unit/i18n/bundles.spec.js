// The four translated bundles against English: the same keys, the same
// placeholders and plural forms, and the glossary terms left untranslated.
import { describe, expect, it } from 'vitest';
import { messages } from '@/locales';
import { i18n } from '@/i18n';

const LANGUAGES = ['fr', 'it', 'de', 'es'];

function leaves(node, prefix = '') {
  return Object.entries(node).flatMap(([key, value]) => (
    typeof value === 'string' ? [[`${prefix}${key}`, value]] : leaves(value, `${prefix}${key}.`)
  ));
}

const placeholders = (text) => [...new Set([...text.matchAll(/\{(\w+)\}/g)].map((m) => m[1]))].sort();
const forms = (text) => text.split('|').length;
const params = new Proxy({}, { get: (_, name) => (typeof name === 'string' ? `<${name}>` : undefined), has: () => true });
const render = (key, locale, n = 2) => i18n.global.t(key, params, { locale, plural: n });

// Numbers are tied to their unit with a no-break space in the translations.
const plain = (text) => text.replace(/\u00a0/g, ' ');
const tn = (key, n, locale) => plain(i18n.global.t(key, n, { locale }));

const allEn = Object.fromEntries(leaves(messages.en));

// English alone writes its dates from message patterns; the other languages
// show Intl's own text (services/intl.js dateLabel), so they carry no copy.
const ENGLISH_ONLY = ['date.hero', 'date.row', 'date.dayMonth', 'date.deadline', 'date.weekdayTime', 'date.dayMonthYear'];
const en = Object.fromEntries(Object.entries(allEn).filter(([key]) => !ENGLISH_ONLY.includes(key)));

describe.each(LANGUAGES)('the %s bundle', (code) => {
  const bundle = Object.fromEntries(leaves(messages[code]));

  it('has exactly the English keys, but the English-only date patterns', () => {
    expect(Object.keys(bundle).sort()).toEqual(Object.keys(en).sort());
  });

  it('has the same placeholders and plural forms as English, and no empty text', () => {
    const mismatches = Object.keys(en).filter((key) => (
      !bundle[key]
      || JSON.stringify(placeholders(bundle[key])) !== JSON.stringify(placeholders(en[key]))
      || forms(bundle[key]) !== forms(en[key])
    ));
    expect(mismatches).toEqual([]);
  });

  it('keeps MVP and Agoranking untranslated, wherever English names them', () => {
    expect(messages[code].glossary).toEqual({ mvp: 'MVP', agoranking: 'Agoranking' });
    const lost = Object.keys(en).flatMap((key) => ['MVP', 'Agoranking'].filter((term) => (
      [1, 2].some((n) => render(key, 'en', n).includes(term) && !render(key, code, n).includes(term))
    )).map((term) => `${key}: ${term}`));
    expect(lost).toEqual([]);
  });

  it('renders every message, with its links and placeholders resolved', () => {
    const broken = Object.keys(en).filter((key) => [1, 2].some((n) => {
      const text = render(key, code, n);
      return text === key || text.includes('glossary.') || text.includes('@:') || /\{\w+\}/.test(text);
    }));
    expect(broken).toEqual([]);
  });

  it('counts MVP titles as « 3 MVP », with no plural -s', () => {
    expect(tn('rankings.awards', 3, code)).toBe('3 MVP');
  });
});

describe('date labels', () => {
  it('uses one Intl formatter for every language but English', async () => {
    const { heroDateLabel, rowDateLabel, dayMonthLabel } = await import('@/services/v2Home');
    const { setLocale } = await import('@/services/locale');
    const labels = {};
    for (const code of ['de', 'fr', 'it', 'es', 'en']) {
      setLocale(code);
      labels[code] = [heroDateLabel('2026-09-06'), heroDateLabel('2026-08-30'), rowDateLabel('2026-09-06'), dayMonthLabel('2026-09-06')].map(plain);
    }
    setLocale('en');
    expect(labels).toEqual({
      de: ['So., 6. Sept.', 'So., 30. Aug.', '06. Sept.', '6. Sept.'],
      fr: ['dim. 6 sept.', 'dim. 30 août', '06 sept.', '6 sept.'],
      it: ['dom 6 set', 'dom 30 ago', '06 set', '6 set'],
      es: ['dom, 6 sept', 'dom, 30 ago', '06 sept', '6 sept'],
      en: ['Sun 6 Sep', 'Sun 30 Aug', '06 Sep', '6 Sep'],
    });
  });
});

describe('plural rules per language', () => {
  it('counts 0 as singular in French, plural elsewhere', () => {
    expect(tn('rankings.matches', 0, 'fr')).toBe('0 match');
    expect(tn('rankings.matches', 0, 'de')).toBe('0 Spiele');
    expect(tn('rankings.matches', 0, 'it')).toBe('0 partite');
    expect(tn('rankings.matches', 0, 'es')).toBe('0 partidos');
    expect(tn('rankings.matches', 1, 'de')).toBe('1 Spiel');
    expect(tn('match.goals', 2, 'es')).toBe('2 goles');
  });

  it('names the award in full per language', () => {
    expect(['fr', 'de', 'it', 'es'].map((locale) => i18n.global.t('award.mvpOfTheMatch', {}, { locale })))
      .toEqual(['MVP du match', 'MVP des Spiels', 'MVP della partita', 'MVP del partido']);
    expect(i18n.global.t('rankings.empty.motm', {}, { locale: 'de' })).toBe('Noch kein MVP');
  });
});
