import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import LocaleSaveErrorToast from '@/components/LocaleSaveErrorToast.vue';
import { startLocaleSync } from '@/services/playerLocale';
import { getLocale, initLocale, setLocale } from '@/services/locale';

const api = vi.hoisted(() => ({
  getMyLocale: vi.fn(),
  updateMyLocale: vi.fn(),
  getSessionKey: vi.fn(() => 'session-1'),
}));
vi.mock('@/services/api', () => api);

const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

const setLanguages = (languages) => {
  Object.defineProperty(window.navigator, 'languages', { value: languages, configurable: true });
};

// The app shell's toast for a failed language save: it does not need the
// selector on the page, so a pick that fails is reported wherever the player is.
describe('LocaleSaveErrorToast', () => {
  let stop;

  beforeEach(() => {
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    api.updateMyLocale.mockReset().mockResolvedValue(null);
    setLanguages(['en']);
    initLocale();
    stop = startLocaleSync();
  });

  afterEach(() => {
    stop();
    vi.restoreAllMocks();
    delete window.navigator.languages;
    initLocale();
  });

  it('shows the error toast, in the previous language, without the selector', async () => {
    const wrapper = mount(LocaleSaveErrorToast);
    api.updateMyLocale.mockRejectedValueOnce(new Error('503'));
    setLocale('fr');
    await flush();
    await wrapper.vm.$nextTick();
    expect(getLocale()).toBe('en');
    expect(wrapper.find('.toast-message').text()).toContain("Couldn't save your language. Try again.");
    expect(wrapper.find('.toast-message').classes()).toContain('error');
  });

  it('shows nothing when the save succeeds', async () => {
    const wrapper = mount(LocaleSaveErrorToast);
    setLocale('fr');
    await flush();
    await wrapper.vm.$nextTick();
    expect(wrapper.find('.toast-message').exists()).toBe(false);
  });

  it('stops listening once unmounted', async () => {
    const wrapper = mount(LocaleSaveErrorToast);
    const show = vi.spyOn(wrapper.vm.$refs.toast, 'show');
    wrapper.unmount();
    api.updateMyLocale.mockRejectedValueOnce(new Error('503'));
    setLocale('fr');
    await flush();
    expect(show).not.toHaveBeenCalled();
  });
});
