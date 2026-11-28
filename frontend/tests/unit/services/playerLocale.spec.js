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

  it('does not save without a session, and survives a failed save', async () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    const { locale, sync } = await load();
    locale.initLocale();
    sync.startLocaleSync();

    api.getSessionKey.mockReturnValue(null);
    locale.setLocale('fr');
    await flush();
    expect(api.updateMyLocale).not.toHaveBeenCalled();

    api.getSessionKey.mockReturnValue('session-1');
    api.updateMyLocale.mockRejectedValueOnce(new Error('503'));
    locale.setLocale('it');
    await flush();
    locale.setLocale('es');
    await flush();
    expect(api.updateMyLocale).toHaveBeenLastCalledWith('es');
    expect(locale.getLocale()).toBe('es');
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
