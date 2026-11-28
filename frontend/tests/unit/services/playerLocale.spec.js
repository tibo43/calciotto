import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// The player's saved language, wired to the locale service. api.js is mocked:
// what matters here is when the server is asked and what the app does with
// the answer, never a network call.
const api = vi.hoisted(() => ({
  getMyLocale: vi.fn(),
  updateMyLocale: vi.fn(),
  getSessionKey: vi.fn(),
}));
vi.mock('@/services/api', () => api);

const setLanguages = (languages) => {
  Object.defineProperty(window.navigator, 'languages', { value: languages, configurable: true });
};

const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

const registered = [];

// Fresh i18n, locale service and sync service per test (module state).
const load = async () => {
  vi.resetModules();
  await import('@/i18n');
  const locale = await import('@/services/locale');
  const sync = await import('@/services/playerLocale');
  return { locale, sync };
};

describe('player locale sync', () => {
  beforeEach(() => {
    document.documentElement.setAttribute('lang', 'en');
    setLanguages(['de-DE']);
    api.getMyLocale.mockReset();
    api.updateMyLocale.mockReset().mockResolvedValue(null);
    api.getSessionKey.mockReset().mockReturnValue('session-1');
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

  it('restores the saved language over the phone\'s, without saving it back', async () => {
    const { locale, sync } = await load();
    locale.initLocale();
    sync.startLocaleSync();
    api.getMyLocale.mockResolvedValue('fr');

    await sync.syncSavedLocale();

    expect(locale.getLocale()).toBe('fr');
    expect(locale.getLocaleSource()).toBe(locale.SOURCE_EXPLICIT);
    expect(document.documentElement.getAttribute('lang')).toBe('fr');
    await flush();
    expect(api.updateMyLocale).not.toHaveBeenCalled();
  });

  it('keeps the phone\'s language when nothing is saved (null)', async () => {
    const { locale, sync } = await load();
    locale.initLocale();
    api.getMyLocale.mockResolvedValue(null);
    await sync.syncSavedLocale();
    expect(locale.getLocale()).toBe('de');
    expect(locale.getLocaleSource()).toBe(locale.SOURCE_PHONE);
  });

  it('asks the server once per session, and again for the next one', async () => {
    const { locale, sync } = await load();
    locale.initLocale();
    api.getMyLocale.mockResolvedValue('it');
    await Promise.all([sync.syncSavedLocale(), sync.syncSavedLocale()]);
    await sync.syncSavedLocale();
    expect(api.getMyLocale).toHaveBeenCalledTimes(1);

    api.getSessionKey.mockReturnValue('session-2');
    api.getMyLocale.mockResolvedValue(null);
    await sync.syncSavedLocale();
    expect(api.getMyLocale).toHaveBeenCalledTimes(2);
    expect(locale.getLocale()).toBe('de');
  });

  it('does not ask at all without a session', async () => {
    const { locale, sync } = await load();
    locale.initLocale();
    api.getSessionKey.mockReturnValue(null);
    await sync.syncSavedLocale();
    expect(api.getMyLocale).not.toHaveBeenCalled();
    expect(locale.getLocale()).toBe('de');
  });

  it('keeps the phone\'s language, and asks again later, when the read fails', async () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    const { locale, sync } = await load();
    locale.initLocale();
    api.getMyLocale.mockRejectedValueOnce(new Error('offline'));
    await sync.syncSavedLocale();
    expect(locale.getLocale()).toBe('de');

    api.getMyLocale.mockResolvedValueOnce('es');
    await sync.syncSavedLocale();
    expect(locale.getLocale()).toBe('es');
  });

  it('lets a pick made while the read is in flight win over its answer', async () => {
    const { locale, sync } = await load();
    locale.initLocale();
    sync.startLocaleSync();
    let answer;
    api.getMyLocale.mockReturnValue(new Promise((resolve) => { answer = resolve; }));
    const reading = sync.syncSavedLocale();

    locale.setLocale('it');
    answer('fr');
    await reading;

    expect(locale.getLocale()).toBe('it');
  });

  it('ignores an answer that arrives after the session ended', async () => {
    const { locale, sync } = await load();
    locale.initLocale();
    let answer;
    api.getMyLocale.mockReturnValue(new Promise((resolve) => { answer = resolve; }));
    const reading = sync.syncSavedLocale();

    api.getSessionKey.mockReturnValue(null);
    answer('fr');
    await reading;

    expect(locale.getLocale()).toBe('de');
  });

  it('ignores an answer that lands after a public page reset the language', async () => {
    const { locale, sync } = await load();
    locale.initLocale();
    sync.startLocaleSync();
    let answer;
    api.getMyLocale.mockReturnValue(new Promise((resolve) => { answer = resolve; }));
    const reading = sync.syncSavedLocale();

    // /login or /reset-password opened with a still-valid token: same session key.
    sync.resetSavedLocale();
    answer('fr');
    await reading;

    expect(locale.getLocale()).toBe('de');
    expect(locale.getLocaleSource()).toBe(locale.SOURCE_PHONE);
  });

  it('drops a queued pick when the session changed before its turn', async () => {
    const { locale, sync } = await load();
    locale.initLocale();
    sync.startLocaleSync();
    const listener = vi.fn();
    sync.onLocaleSaveError(listener);
    let release;
    api.updateMyLocale
      .mockImplementationOnce(() => new Promise((resolve) => { release = resolve; }))
      .mockRejectedValue(new Error('should not be sent'));

    locale.setLocale('fr');
    locale.setLocale('it'); // queued behind 'fr'
    await flush();
    // A logs out, B logs in, in the same tab.
    api.getSessionKey.mockReturnValue('session-2');
    release(null);
    await flush();

    expect(api.updateMyLocale.mock.calls.map(([value]) => value)).toEqual(['fr']);
    expect(listener).not.toHaveBeenCalled();
    expect(locale.getLocale()).toBe('it');
  });

  it('saves a pick, and null when the player goes back to the phone', async () => {
    const { locale, sync } = await load();
    locale.initLocale();
    sync.startLocaleSync();

    locale.setLocale('es');
    await flush();
    expect(api.updateMyLocale).toHaveBeenLastCalledWith('es');

    locale.followPhone();
    await flush();
    expect(api.updateMyLocale).toHaveBeenLastCalledWith(null);
    expect(api.updateMyLocale).toHaveBeenCalledTimes(2);
  });

  it('saves picks one after the other, in order', async () => {
    const { locale, sync } = await load();
    locale.initLocale();
    sync.startLocaleSync();
    const order = [];
    let release;
    api.updateMyLocale.mockImplementation((value) => {
      order.push(`start ${value}`);
      if (value === 'fr') return new Promise((resolve) => { release = () => { order.push('end fr'); resolve(value); }; });
      return Promise.resolve(value);
    });

    locale.setLocale('fr');
    locale.setLocale('it');
    await flush();
    expect(order).toEqual(['start fr']);
    release();
    await flush();
    expect(order).toEqual(['start fr', 'end fr', 'start it']);
  });

  it('does not save without a session', async () => {
    const { locale, sync } = await load();
    locale.initLocale();
    sync.startLocaleSync();
    api.getSessionKey.mockReturnValue(null);
    locale.setLocale('fr');
    await flush();
    expect(api.updateMyLocale).not.toHaveBeenCalled();
    expect(locale.getLocale()).toBe('fr');
  });

  describe('when a save fails', () => {
    beforeEach(() => {
      vi.spyOn(console, 'warn').mockImplementation(() => {});
    });

    it('puts the previous language back, then says so, and saves nothing more', async () => {
      const { locale, sync } = await load();
      locale.initLocale();
      sync.startLocaleSync();
      const seen = [];
      sync.onLocaleSaveError(() => seen.push({ locale: locale.getLocale(), lang: document.documentElement.getAttribute('lang') }));
      api.updateMyLocale.mockRejectedValueOnce(new Error('503'));

      locale.setLocale('es');
      expect(locale.getLocale()).toBe('es');
      await flush();

      // Back to the phone's German, interface and <html lang> both, before the listener ran.
      expect(locale.getLocale()).toBe('de');
      expect(locale.getLocaleSource()).toBe(locale.SOURCE_PHONE);
      expect(seen).toEqual([{ locale: 'de', lang: 'de' }]);
      expect(api.updateMyLocale).toHaveBeenCalledTimes(1);
    });

    it('goes back to the language the server holds, a restored one included', async () => {
      const { locale, sync } = await load();
      locale.initLocale();
      sync.startLocaleSync();
      api.getMyLocale.mockResolvedValue('fr');
      await sync.syncSavedLocale();
      const listener = vi.fn();
      sync.onLocaleSaveError(listener);
      api.updateMyLocale.mockRejectedValueOnce(new Error('offline'));

      locale.setLocale('it');
      await flush();

      expect(locale.getLocale()).toBe('fr');
      expect(locale.getLocaleSource()).toBe(locale.SOURCE_EXPLICIT);
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it('goes back to the last pick that was saved', async () => {
      const { locale, sync } = await load();
      locale.initLocale();
      sync.startLocaleSync();
      const listener = vi.fn();
      sync.onLocaleSaveError(listener);
      api.updateMyLocale.mockResolvedValueOnce('fr').mockRejectedValueOnce(new Error('500'));

      locale.setLocale('fr');
      await flush();
      expect(listener).not.toHaveBeenCalled();
      locale.setLocale('it');
      await flush();

      expect(locale.getLocale()).toBe('fr');
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it('lets a failure of an old save leave a newer pick alone', async () => {
      const { locale, sync } = await load();
      locale.initLocale();
      sync.startLocaleSync();
      const listener = vi.fn();
      sync.onLocaleSaveError(listener);
      let failFirst;
      api.updateMyLocale
        .mockImplementationOnce(() => new Promise((resolve, reject) => { failFirst = reject; }))
        .mockResolvedValueOnce('it');

      locale.setLocale('fr');
      locale.setLocale('it');
      await flush();
      failFirst(new Error('timeout'));
      await flush();

      expect(locale.getLocale()).toBe('it');
      expect(listener).not.toHaveBeenCalled();
      expect(api.updateMyLocale.mock.calls.map(([value]) => value)).toEqual(['fr', 'it']);

      // The newer pick is the one the server now holds: a later failure goes back to it.
      api.updateMyLocale.mockRejectedValueOnce(new Error('500'));
      locale.setLocale('es');
      await flush();
      expect(locale.getLocale()).toBe('it');
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it('answers a newer pick that fails too once, with the language the server holds', async () => {
      const { locale, sync } = await load();
      locale.initLocale();
      sync.startLocaleSync();
      const listener = vi.fn();
      sync.onLocaleSaveError(listener);
      api.updateMyLocale.mockRejectedValue(new Error('500'));

      locale.setLocale('fr');
      locale.setLocale('it');
      await flush();

      expect(locale.getLocale()).toBe('de');
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it('goes back to the server\'s language when the startup read landed after the pick', async () => {
      vi.spyOn(console, 'warn').mockImplementation(() => {});
      const { locale, sync } = await load();
      locale.initLocale();
      sync.startLocaleSync();
      const listener = vi.fn();
      sync.onLocaleSaveError(listener);
      let answer;
      api.getMyLocale.mockReturnValue(new Promise((resolve) => { answer = resolve; }));
      let failSave;
      api.updateMyLocale.mockImplementationOnce(() => new Promise((resolve, reject) => { failSave = reject; }));
      const reading = sync.syncSavedLocale();

      locale.setLocale('it');
      answer('fr');
      await reading;
      // The pick still wins on screen while its save is pending.
      expect(locale.getLocale()).toBe('it');

      failSave(new Error('500'));
      await flush();

      expect(locale.getLocale()).toBe('fr');
      expect(locale.getLocaleSource()).toBe(locale.SOURCE_EXPLICIT);
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it('goes back to the phone when the late startup read said nothing is saved', async () => {
      const { locale, sync } = await load();
      locale.initLocale();
      api.getMyLocale.mockResolvedValue('fr');
      sync.startLocaleSync();
      // A first session restores French, then the next account (nothing saved) starts.
      await sync.syncSavedLocale();
      api.getSessionKey.mockReturnValue('session-2');
      let answer;
      api.getMyLocale.mockReturnValue(new Promise((resolve) => { answer = resolve; }));
      api.updateMyLocale.mockRejectedValueOnce(new Error('500'));
      const reading = sync.syncSavedLocale();

      locale.setLocale('it');
      answer(null);
      await reading;
      await flush();

      expect(locale.getLocale()).toBe('de');
      expect(locale.getLocaleSource()).toBe(locale.SOURCE_PHONE);
    });

    it('keeps a save that succeeded over an older startup answer', async () => {
      const { locale, sync } = await load();
      locale.initLocale();
      sync.startLocaleSync();
      let answer;
      api.getMyLocale.mockReturnValue(new Promise((resolve) => { answer = resolve; }));
      const reading = sync.syncSavedLocale();

      locale.setLocale('it');
      await flush(); // 'it' saved
      answer('fr');
      await reading;
      api.updateMyLocale.mockRejectedValueOnce(new Error('500'));
      locale.setLocale('es');
      await flush();

      expect(locale.getLocale()).toBe('it');
    });

    it('stays silent when a save succeeds', async () => {
      const { locale, sync } = await load();
      locale.initLocale();
      sync.startLocaleSync();
      const listener = vi.fn();
      sync.onLocaleSaveError(listener);
      locale.setLocale('fr');
      locale.followPhone();
      await flush();
      expect(listener).not.toHaveBeenCalled();
      expect(api.updateMyLocale.mock.calls.map(([value]) => value)).toEqual(['fr', null]);
    });
  });

  it('goes back to the browser\'s language when the session ends, saving nothing', async () => {
    const { locale, sync } = await load();
    locale.initLocale();
    sync.startLocaleSync();
    api.getMyLocale.mockResolvedValue('fr');
    await sync.syncSavedLocale();
    expect(locale.getLocale()).toBe('fr');

    sync.resetSavedLocale();
    await flush();

    expect(locale.getLocale()).toBe('de');
    expect(locale.getLocaleSource()).toBe(locale.SOURCE_PHONE);
    expect(document.documentElement.getAttribute('lang')).toBe('de');
    expect(api.updateMyLocale).not.toHaveBeenCalled();
    // The next session reads again.
    api.getSessionKey.mockReturnValue('session-2');
    api.getMyLocale.mockResolvedValue(null);
    await sync.syncSavedLocale();
    expect(api.getMyLocale).toHaveBeenCalledTimes(2);
  });
});
