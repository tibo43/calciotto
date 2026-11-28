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

  it('falls back to English for a key a language does not have', () => {
    i18n.global.mergeLocaleMessage('en', { test: { onlyEnglish: 'Only in English' } });
    i18n.global.locale.value = 'fr';
    try {
      expect(i18n.global.t('language.label')).toBe('Langue');
      expect(i18n.global.t('test.onlyEnglish')).toBe('Only in English');
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

  it('has a bundle for each of the five languages', () => {
    expect(Object.keys(messages).sort()).toEqual(['de', 'en', 'es', 'fr', 'it']);
  });
});
