import { shallowMount, flushPromises } from '@vue/test-utils';
import { createRouter, createMemoryHistory } from 'vue-router';
import App from '@/App.vue';
import { getMyFeatures, clearToken } from '@/services/api';
import { clearActiveGroupId, clearMyGroupsCache } from '@/services/activeGroup';
import { disablePushOnLogout } from '@/services/pushNotifications';

jest.mock('@/services/api', () => ({
  getMyFeatures: jest.fn(),
  clearToken: jest.fn()
}));

jest.mock('@/services/activeGroup', () => ({
  clearActiveGroupId: jest.fn(),
  clearMyGroupsCache: jest.fn()
}));

jest.mock('@/services/theme', () => ({
  syncThemeColorMeta: jest.fn()
}));

jest.mock('@/services/pushNotifications', () => ({
  disablePushOnLogout: jest.fn(() => Promise.resolve())
}));

const STORAGE_KEY = 'calciotto-design-v2';

const mountAt = async (path) => {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'MatchesAndStandings', component: { template: '<div />' } },
      { path: '/profile', name: 'Profile', component: { template: '<div />' } },
      { path: '/predictions', name: 'Predictions', component: { template: '<div />' } },
      { path: '/login', name: 'Login', component: { template: '<div />' } },
      { path: '/signup', name: 'Signup', component: { template: '<div />' } },
      { path: '/forgot-password', name: 'ForgotPassword', component: { template: '<div />' } },
      { path: '/reset-password', name: 'ResetPassword', component: { template: '<div />' } }
    ]
  });
  await router.push(path);
  await router.isReady();
  const wrapper = shallowMount(App, {
    global: { plugins: [router] }
  });
  return { wrapper, router };
};

describe('App.vue design-v2', () => {
  beforeEach(() => {
    localStorage.clear();
    localStorage.setItem('calciotto-theme', 'dark');
    document.getElementById('design-v2-fonts')?.remove();
    getMyFeatures.mockReset();
    getMyFeatures.mockResolvedValue([]);
    jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    document.getElementById('design-v2-fonts')?.remove();
    jest.restoreAllMocks();
  });

  it('applies the cached class before the features request resolves, on an authenticated route', async () => {
    let resolveFeatures;
    getMyFeatures.mockReturnValue(new Promise((resolve) => {
      resolveFeatures = resolve;
    }));
    localStorage.setItem(STORAGE_KEY, '1');

    const { wrapper } = await mountAt('/');

    expect(wrapper.classes()).toContain('design-v2');
    expect(document.getElementById('design-v2-fonts')).not.toBeNull();
    expect(getMyFeatures).toHaveBeenCalledTimes(1);

    resolveFeatures([]);
    await flushPromises();

    expect(wrapper.classes()).not.toContain('design-v2');
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull();
    expect(document.getElementById('design-v2-fonts')).toBeNull();
  });

  it('turns the class on and remembers it when the response includes design-v2', async () => {
    getMyFeatures.mockResolvedValue(['pwa-install', 'design-v2']);

    const { wrapper } = await mountAt('/');
    await flushPromises();

    expect(wrapper.classes()).toContain('design-v2');
    expect(localStorage.getItem(STORAGE_KEY)).toBe('1');
    expect(document.getElementById('design-v2-fonts').href).toContain('fonts.googleapis.com');
  });

  it('leaves the class off, and does not request the fonts, when the feature is absent', async () => {
    getMyFeatures.mockResolvedValue(['pwa-install']);

    const { wrapper } = await mountAt('/');
    await flushPromises();

    expect(wrapper.classes()).not.toContain('design-v2');
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull();
    expect(document.getElementById('design-v2-fonts')).toBeNull();
  });

  it('keeps the cached class when the features request fails', async () => {
    localStorage.setItem(STORAGE_KEY, '1');
    getMyFeatures.mockRejectedValue(new Error('offline'));

    const { wrapper } = await mountAt('/');
    await flushPromises();

    expect(wrapper.classes()).toContain('design-v2');
    expect(localStorage.getItem(STORAGE_KEY)).toBe('1');
    expect(console.error).toHaveBeenCalled();
  });

  it('does not apply the class or fetch features on a public route, even with a cached flag', async () => {
    localStorage.setItem(STORAGE_KEY, '1');

    const { wrapper } = await mountAt('/login');
    await flushPromises();

    expect(wrapper.classes()).not.toContain('design-v2');
    expect(getMyFeatures).not.toHaveBeenCalled();
    expect(localStorage.getItem(STORAGE_KEY)).toBe('1');
    expect(document.getElementById('design-v2-fonts')).toBeNull();
  });

  it.each(['/signup', '/forgot-password', '/reset-password'])(
    'keeps %s on the current design',
    async (path) => {
      localStorage.setItem(STORAGE_KEY, '1');
      const { wrapper } = await mountAt(path);
      await flushPromises();
      expect(wrapper.classes()).not.toContain('design-v2');
      expect(getMyFeatures).not.toHaveBeenCalled();
    }
  );

  it('drops the class on a public route without forgetting the cache or refetching', async () => {
    getMyFeatures.mockResolvedValue(['design-v2']);
    const { wrapper, router } = await mountAt('/');
    await flushPromises();
    expect(getMyFeatures).toHaveBeenCalledTimes(1);

    await router.push('/login');
    await flushPromises();

    expect(wrapper.classes()).not.toContain('design-v2');
    expect(localStorage.getItem(STORAGE_KEY)).toBe('1');
    expect(getMyFeatures).toHaveBeenCalledTimes(1);

    await router.push('/');
    await flushPromises();

    expect(wrapper.classes()).toContain('design-v2');
    expect(getMyFeatures).toHaveBeenCalledTimes(1);
  });

  it('fetches features once per session across authenticated routes', async () => {
    getMyFeatures.mockResolvedValue(['design-v2']);
    const { wrapper, router } = await mountAt('/');
    await flushPromises();

    await router.push('/profile');
    await flushPromises();
    await router.push('/predictions');
    await flushPromises();

    expect(getMyFeatures).toHaveBeenCalledTimes(1);
    expect(wrapper.classes()).toContain('design-v2');
  });

  it('clears the cache on logout and ignores a features response that resolves afterwards', async () => {
    let resolveFeatures;
    getMyFeatures.mockReturnValue(new Promise((resolve) => {
      resolveFeatures = resolve;
    }));
    localStorage.setItem(STORAGE_KEY, '1');
    const { wrapper } = await mountAt('/');
    expect(wrapper.classes()).toContain('design-v2');

    await wrapper.vm.logout();

    expect(clearToken).toHaveBeenCalled();
    expect(clearActiveGroupId).toHaveBeenCalled();
    expect(clearMyGroupsCache).toHaveBeenCalled();
    expect(disablePushOnLogout).toHaveBeenCalled();
    expect(wrapper.classes()).not.toContain('design-v2');
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull();
    expect(document.getElementById('design-v2-fonts')).toBeNull();

    resolveFeatures(['design-v2']);
    await flushPromises();

    expect(wrapper.classes()).not.toContain('design-v2');
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull();
  });

  it('loads features again for the next account after logout', async () => {
    getMyFeatures.mockResolvedValueOnce([]);
    const { wrapper, router } = await mountAt('/');
    await flushPromises();
    expect(wrapper.classes()).not.toContain('design-v2');

    await wrapper.vm.logout();
    await flushPromises();
    expect(wrapper.vm.$route.name).toBe('Login');
    expect(wrapper.classes()).not.toContain('design-v2');

    getMyFeatures.mockResolvedValueOnce(['design-v2']);
    await router.push('/');
    await flushPromises();

    expect(getMyFeatures).toHaveBeenCalledTimes(2);
    expect(wrapper.classes()).toContain('design-v2');
    expect(localStorage.getItem(STORAGE_KEY)).toBe('1');
  });
});
