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

// Every message that puts a count next to a noun, at 0, 1, 2 and 1.5: the
// singular form where the language's own rule says so (French: 0 and 1, and
// anything under 2), the plural form everywhere else.
describe('count messages agree with their number in every language', () => {
  const SINGULAR = { fr: [0, 1, 1.5], it: [1], de: [1], es: [1] };
  const PLURAL = { fr: [2], it: [0, 1.5, 2], de: [0, 1.5, 2], es: [0, 1.5, 2] };
  // [key, { language: [singular, plural] }]; « # » stands for the count.
  const FORMS = {
    'home.placesLeft': { fr: ['place restante', 'places restantes'], it: ['posto libero', 'posti liberi'], de: ['Platz frei', 'Plätze frei'], es: ['plaza libre', 'plazas libres'] },
    'edit.confirmedCount': { fr: ['# confirmé', '# confirmés'], it: ['# confermato', '# confermati'], de: ['# bestätigt', '# bestätigt'], es: ['# confirmado', '# confirmados'] },
    'edit.placesCount': { fr: ['# place', '# places'], it: ['# posto', '# posti'], de: ['# Platz', '# Plätze'], es: ['# plaza', '# plazas'] },
    'profile.groupMatches': { fr: ['# J', '# J'], it: ['# G', '# G'], de: ['# Sp.', '# Sp.'], es: ['# PJ', '# PJ'] },
    'profile.groupPoints': { fr: ['# Pt', '# Pts'], it: ['# Pt', '# Pt'], de: ['# Pkt.', '# Pkt.'], es: ['# Pt', '# Pts'] },
    'profile.groupsCombined': { fr: ['# groupe cumulé', '# groupes cumulés'], it: ['# gruppo insieme', '# gruppi insieme'], de: ['# Gruppe zusammen', '# Gruppen zusammen'], es: ['# grupo en total', '# grupos en total'] },
  };
  const shown = (key, n, code) => {
    const text = tn(key, n, code);
    return key === 'home.placesLeft' ? text.replace(/^[\d.]+ /, '') : text;
  };

  describe.each(LANGUAGES)('in %s', (code) => {
    it.each(Object.entries(FORMS))('%s', (key, byLanguage) => {
      const [one, other] = byLanguage[code];
      const expected = (form, n) => (key === 'home.placesLeft' ? form : form.replace('#', String(n).replace('.', ',')));
      SINGULAR[code].forEach((n) => expect(shown(key, n, code)).toBe(expected(one, n)));
      PLURAL[code].forEach((n) => expect(shown(key, n, code)).toBe(expected(other, n)));
    });
  });

  it('agrees the reserve note and the two-count lines with their own counts', () => {
    const named = (key, params, n, code) => plain(i18n.global.t(key, params, { locale: code, plural: n }));
    for (const code of LANGUAGES) {
      [0, 1, 2].forEach((n) => {
        const note = named('edit.signups.notComposed', { confirmed: n, bench: '' }, n, code);
        expect(note).toContain(String(n));
        expect(note).not.toContain('|');
      });
      const line = named('edit.confirmedPlaces', { confirmed: tn('edit.confirmedCount', 1, code), cap: tn('edit.placesCount', 1, code) }, 1, code);
      expect(line).toBe(`${tn('edit.confirmedCount', 1, code)} / ${tn('edit.placesCount', 1, code)}`);
      const meta = named('profile.groupMeta', { matches: tn('profile.groupMatches', 1, code), points: tn('profile.groupPoints', 1, code) }, 1, code);
      expect(meta).toBe(`${tn('profile.groupMatches', 1, code)} · ${tn('profile.groupPoints', 1, code)}`);
    }
    expect(tn('edit.confirmedCount', 1, 'fr')).toBe('1 confirmé');
    expect(`${tn('profile.groupMatches', 1, 'de')} · ${tn('profile.groupPoints', 2, 'de')}`).toBe('1 Sp. · 2 Pkt.');
    expect(`${tn('profile.groupMatches', 0, 'fr')} · ${tn('profile.groupPoints', 0, 'fr')}`).toBe('0 J · 0 Pt');
  });
});

describe('numbers in count messages follow the language', () => {
  it('writes 1.5 with a comma in French, German, Italian and Spanish, a dot in English', () => {
    expect(tn('unit.points', 1.5, 'fr')).toBe('1,5 pt');
    expect(tn('unit.points', 1.5, 'de')).toBe('1,5 Pkt.');
    expect(tn('unit.points', 1.5, 'it')).toBe('1,5 pt');
    expect(tn('unit.points', 1.5, 'es')).toBe('1,5 pts');
    expect(tn('unit.points', 1.5, 'en')).toBe('1.5 pts');
    expect(tn('profile.groupPoints', 1.5, 'fr')).toBe('1,5 Pt');
    expect(tn('profile.groupPoints', 2.5, 'fr')).toBe('2,5 Pts');
    expect(tn('profile.groupPoints', 1.5, 'de')).toBe('1,5 Pkt.');
  });

  it('groups thousands the way each language does', () => {
    expect(tn('rankings.matches', 1234, 'de')).toBe('1.234 Spiele');
    expect(tn('rankings.matches', 1234, 'en')).toBe('1,234 matches');
    // French groups with a narrow no-break space (U+202F).
    expect(plain(i18n.global.t('rankings.matches', { n: 1234 }, { locale: 'fr', plural: 1234 })).replace(/\u202f/g, ' ')).toBe('1 234 matchs');
  });
});

describe('the places toast', () => {
  it('says the places were updated, without a number, in every language', () => {
    const toast = ['en', 'fr', 'de', 'it', 'es'].map((locale) => i18n.global.t('edit.toast.placesSet', {}, { locale }));
    expect(toast).toEqual(['Places updated.', 'Places mises à jour.', 'Plätze aktualisiert.', 'Posti aggiornati.', 'Plazas actualizadas.']);
  });
});
