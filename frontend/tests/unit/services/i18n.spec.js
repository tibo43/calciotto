import { describe, expect, it } from 'vitest';
import en from '@/locales/en.json';
import { LOCALES, DEFAULT_LOCALE, messages } from '@/locales';
import { i18n } from '@/i18n';
import { MVP } from '@/services/v2Home';

describe('vue-i18n instance', () => {
  it('is the composition API, in English, falling back to English', () => {
    expect(i18n.mode).toBe('composition');
    expect(i18n.global.locale.value).toBe('en');
    expect(i18n.global.fallbackLocale.value).toBe('en');
  });

  it('has the English messages', () => {
    expect(messages.en).toEqual(en);
    expect(i18n.global.t('language.label')).toBe('Language');
  });

  it('falls back to English for a language with no message', () => {
    i18n.global.locale.value = 'fr';
    try {
      expect(i18n.global.t('language.label')).toBe('Language');
    } finally {
      i18n.global.locale.value = 'en';
    }
  });

  it('keeps MVP and Agoranking as single, untranslated glossary keys', () => {
    expect(en.glossary.mvp).toBe('MVP');
    expect(MVP).toBe('MVP');
    expect(en.glossary.agoranking).toBe('Agoranking');
    expect(i18n.global.t('award.mvpOfTheMatch')).toBe('MVP of the Match');
  });

  it('lists the five languages in the selector order, with their own names', () => {
    expect(LOCALES.map((l) => [l.code, l.name])).toEqual([
      ['en', 'English'], ['fr', 'Français'], ['it', 'Italiano'], ['de', 'Deutsch'], ['es', 'Español'],
    ]);
    expect(DEFAULT_LOCALE).toBe('en');
  });

  it('has a bundle for English only so far', () => {
    expect(Object.keys(messages)).toEqual(['en']);
  });
});

describe('t, the plural form from a named count', () => {
  it('picks the plural form from a numeric {n}', () => {
    expect(i18n.global.t('match.goals', { n: 3 })).toBe('3 goals');
    expect(i18n.global.t('match.goals', { n: 1 })).toBe('1 goal');
  });

  it('picks the plural form from a numeric {count}', () => {
    i18n.global.mergeLocaleMessage('en', { pluralTest: { items: '{count} item | {count} items' } });
    expect(i18n.global.t('pluralTest.items', { count: 1 })).toBe('1 item');
    expect(i18n.global.t('pluralTest.items', { count: 4 })).toBe('4 items');
  });

  it('treats a decimal {n} as plural and formats it per language', () => {
    expect(i18n.global.t('match.goals', { n: 1.5 })).toBe('1.5 goals');
  });
});

describe('messageKey and showText', () => {
  it('translates a stored key when shown, in the language active then', async () => {
    const { messageKey, showText } = await import('@/i18n');
    const stored = messageKey('language.label');
    i18n.global.mergeLocaleMessage('fr', { language: { label: 'Langue' } });
    expect(showText(stored)).toBe('Language');
    i18n.global.locale.value = 'fr';
    try {
      expect(showText(stored)).toBe('Langue');
    } finally {
      i18n.global.locale.value = 'en';
    }
  });

  it('shows server text unchanged, and nothing for an empty value', async () => {
    const { showText } = await import('@/i18n');
    expect(showText('name already used')).toBe('name already used');
    expect(showText('')).toBe('');
    expect(showText(null)).toBe('');
  });
});
