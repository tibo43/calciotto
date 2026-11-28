import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const setLanguages = (languages) => {
  Object.defineProperty(window.navigator, 'languages', { value: languages, configurable: true });
};

const ALL = ['fr', 'it', 'de', 'es'];

// A fresh i18n instance and locale service per test (module state). Only the
// given languages stay "available": the others get an empty bundle, which is
// how the app tells a language it cannot show yet.
const load = async (available = []) => {
  vi.resetModules();
  const { i18n } = await import('@/i18n');
  ALL.filter((code) => !available.includes(code)).forEach((code) => i18n.global.setLocaleMessage(code, {}));
  const service = await import('@/services/locale');
  return { i18n, ...service };
};

// Each fresh module registers its own 'languagechange' listener on the shared
// window; they are removed after each test so an old instance cannot answer
// the next test's events.
const registered = [];

describe('locale service', () => {
  beforeEach(() => {
    document.documentElement.setAttribute('lang', 'en');
    const add = window.addEventListener.bind(window);
    vi.spyOn(window, 'addEventListener').mockImplementation((type, handler, options) => {
      if (type === 'languagechange') registered.push(handler);
      add(type, handler, options);
    });
  });

  afterEach(() => {
    delete window.navigator.languages;
    vi.restoreAllMocks();
    registered.splice(0).forEach((handler) => window.removeEventListener('languagechange', handler));
  });

  describe('detectLocale', () => {
    it('takes the first supported language of the list', async () => {
      const { detectLocale } = await load(ALL);
      expect(detectLocale(['de-DE', 'fr'])).toBe('de');
      expect(detectLocale(['it'])).toBe('it');
      expect(detectLocale(['es-MX', 'en-US'])).toBe('es');
    });

    it('reads the language of a regional tag, in any case or separator', async () => {
      const { detectLocale } = await load(ALL);
      expect(detectLocale(['fr-CA'])).toBe('fr');
      expect(detectLocale(['FR'])).toBe('fr');
      expect(detectLocale(['de_AT'])).toBe('de');
    });

    it('skips unsupported languages for the next supported one, not straight to English', async () => {
      const { detectLocale } = await load(ALL);
      expect(detectLocale(['pt-BR', 'ja', 'it-IT', 'fr'])).toBe('it');
    });

    it('falls back to English with no supported language, an empty list or junk', async () => {
      const { detectLocale } = await load(ALL);
      expect(detectLocale(['pt-BR', 'ja'])).toBe('en');
      expect(detectLocale([])).toBe('en');
      expect(detectLocale(['', null, undefined, '*', '123'])).toBe('en');
    });

    it('treats English as supported', async () => {
      const { detectLocale } = await load(ALL);
      expect(detectLocale(['en-GB', 'fr'])).toBe('en');
    });

    it('only counts languages that have messages', async () => {
      const { detectLocale, availableLocales } = await load();
      expect(availableLocales()).toEqual(['en']);
      expect(detectLocale(['fr-FR', 'de'])).toBe('en');
    });

    it('lists available languages in the selector order', async () => {
      const { availableLocales } = await load(['es', 'de']);
      expect(availableLocales()).toEqual(['en', 'de', 'es']);
    });

    it('reads navigator.languages by default, then navigator.language', async () => {
      const { detectLocale } = await load(ALL);
      setLanguages(['it-IT', 'en']);
      expect(detectLocale()).toBe('it');
      setLanguages([]);
      vi.spyOn(window.navigator, 'language', 'get').mockReturnValue('de-CH');
      expect(detectLocale()).toBe('de');
    });
  });

  describe('initLocale', () => {
    it('follows the phone by default', async () => {
      setLanguages(['fr-FR']);
      const { initLocale, getLocale, getLocaleSource, i18n } = await load(ALL);
      initLocale();
      expect(getLocale()).toBe('fr');
      expect(getLocaleSource()).toBe('phone');
      expect(i18n.global.locale.value).toBe('fr');
    });

    it('is English with no supported phone language', async () => {
      setLanguages(['pt-BR']);
      const { initLocale, getLocale, getLocaleSource } = await load(ALL);
      initLocale();
      expect(getLocale()).toBe('en');
      expect(getLocaleSource()).toBe('phone');
    });

    it('restores a saved choice over the phone, as an explicit one', async () => {
      setLanguages(['fr-FR']);
      const { initLocale, getLocale, getLocaleSource } = await load(ALL);
      initLocale({ saved: 'de' });
      expect(getLocale()).toBe('de');
      expect(getLocaleSource()).toBe('explicit');
    });

    it('ignores a saved choice that is not supported (follows the phone)', async () => {
      setLanguages(['it']);
      const { initLocale, getLocale, getLocaleSource } = await load(ALL);
      initLocale({ saved: 'pt' });
      expect(getLocale()).toBe('it');
      expect(getLocaleSource()).toBe('phone');
    });

    it('does not announce the restore', async () => {
      const { initLocale, onLocaleChange } = await load(ALL);
      const listener = vi.fn();
      onLocaleChange(listener);
      initLocale({ saved: 'es' });
      expect(listener).not.toHaveBeenCalled();
    });
  });

  describe('setLocale / followPhone', () => {
    it('sets an explicit language everywhere and announces it', async () => {
      setLanguages(['fr']);
      const { initLocale, setLocale, getLocale, getLocaleSource, onLocaleChange, i18n } = await load(ALL);
      initLocale();
      const listener = vi.fn();
      onLocaleChange(listener);
      expect(setLocale('de')).toBe(true);
      expect(getLocale()).toBe('de');
      expect(getLocaleSource()).toBe('explicit');
      expect(i18n.global.locale.value).toBe('de');
      expect(listener).toHaveBeenCalledWith({ locale: 'de', source: 'explicit' });
    });

    it('chosen language equal to the phone one is still explicit', async () => {
      setLanguages(['fr']);
      const { initLocale, setLocale, getLocaleSource } = await load(ALL);
      initLocale();
      setLocale('fr');
      expect(getLocaleSource()).toBe('explicit');
    });

    it('refuses an unsupported language and changes nothing', async () => {
      const { initLocale, setLocale, getLocale, onLocaleChange } = await load(['fr']);
      initLocale();
      const listener = vi.fn();
      onLocaleChange(listener);
      expect(setLocale('de')).toBe(false);
      expect(setLocale('')).toBe(false);
      expect(setLocale(null)).toBe(false);
      expect(getLocale()).toBe('en');
      expect(listener).not.toHaveBeenCalled();
    });

    it('goes back to the phone language', async () => {
      setLanguages(['it']);
      const { initLocale, setLocale, followPhone, getLocale, getLocaleSource, onLocaleChange } = await load(ALL);
      initLocale();
      setLocale('de');
      const listener = vi.fn();
      onLocaleChange(listener);
      followPhone();
      expect(getLocale()).toBe('it');
      expect(getLocaleSource()).toBe('phone');
      expect(listener).toHaveBeenCalledWith({ locale: 'it', source: 'phone' });
    });

    it('stops telling a listener that unsubscribed', async () => {
      const { initLocale, setLocale, onLocaleChange } = await load(ALL);
      initLocale();
      const listener = vi.fn();
      const off = onLocaleChange(listener);
      off();
      setLocale('fr');
      expect(listener).not.toHaveBeenCalled();
    });

    it('exposes a read-only state for components', async () => {
      const { initLocale, setLocale, localeState } = await load(ALL);
      initLocale();
      setLocale('es');
      expect(localeState).toEqual({ locale: 'es', source: 'explicit' });
      vi.spyOn(console, 'warn').mockImplementation(() => {});
      localeState.locale = 'fr';
      expect(localeState.locale).toBe('es');
    });
  });

  describe('<html lang>', () => {
    it('is set at start-up from the phone', async () => {
      setLanguages(['de-DE']);
      const { initLocale } = await load(ALL);
      initLocale();
      expect(document.documentElement.getAttribute('lang')).toBe('de');
    });

    it('follows every change', async () => {
      setLanguages(['en']);
      const { initLocale, setLocale, followPhone } = await load(ALL);
      initLocale();
      expect(document.documentElement.getAttribute('lang')).toBe('en');
      setLocale('it');
      expect(document.documentElement.getAttribute('lang')).toBe('it');
      followPhone();
      expect(document.documentElement.getAttribute('lang')).toBe('en');
    });

    it('is not changed by a refused language', async () => {
      const { initLocale, setLocale } = await load();
      initLocale();
      setLocale('fr');
      expect(document.documentElement.getAttribute('lang')).toBe('en');
    });
  });

  describe('phone language change', () => {
    it('is followed while nothing was chosen', async () => {
      setLanguages(['fr']);
      const { initLocale, getLocale, onLocaleChange } = await load(ALL);
      initLocale();
      const listener = vi.fn();
      onLocaleChange(listener);
      setLanguages(['es-ES']);
      window.dispatchEvent(new Event('languagechange'));
      expect(getLocale()).toBe('es');
      expect(document.documentElement.getAttribute('lang')).toBe('es');
      expect(listener).toHaveBeenCalledWith({ locale: 'es', source: 'phone' });
    });

    it('is ignored once a language was chosen', async () => {
      setLanguages(['fr']);
      const { initLocale, setLocale, getLocale } = await load(ALL);
      initLocale();
      setLocale('de');
      setLanguages(['es']);
      window.dispatchEvent(new Event('languagechange'));
      expect(getLocale()).toBe('de');
    });

    it('does not announce when the detected language is the same', async () => {
      setLanguages(['fr-FR']);
      const { initLocale, onLocaleChange } = await load(ALL);
      initLocale();
      const listener = vi.fn();
      onLocaleChange(listener);
      setLanguages(['fr-CA']);
      window.dispatchEvent(new Event('languagechange'));
      expect(listener).not.toHaveBeenCalled();
    });
  });

  describe('storage', () => {
    it('never touches localStorage or sessionStorage', async () => {
      const get = vi.spyOn(Storage.prototype, 'getItem');
      const set = vi.spyOn(Storage.prototype, 'setItem');
      const remove = vi.spyOn(Storage.prototype, 'removeItem');
      setLanguages(['fr']);
      const { initLocale, setLocale, followPhone } = await load(ALL);
      initLocale({ saved: 'it' });
      setLocale('de');
      followPhone();
      expect(get).not.toHaveBeenCalled();
      expect(set).not.toHaveBeenCalled();
      expect(remove).not.toHaveBeenCalled();
    });
  });
});
