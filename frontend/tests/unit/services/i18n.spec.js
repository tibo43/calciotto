import { describe, expect, it } from 'vitest';
import en from '@/locales/en.json';
import { LOCALES, DEFAULT_LOCALE, messages } from '@/locales';
import { i18n } from '@/i18n';
import { MVP, MVP_OF_THE_MATCH } from '@/services/v2Home';

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

  it('keeps the award names of the screens (until they come from here)', () => {
    expect(en.award.mvp).toBe(MVP);
    expect(en.award.mvpOfTheMatch).toBe(MVP_OF_THE_MATCH);
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
