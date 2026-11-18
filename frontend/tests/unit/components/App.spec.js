import { shallowMount, flushPromises } from '@vue/test-utils';
import { createRouter, createMemoryHistory } from 'vue-router';
import App from '@/App.vue';
import { getMyFeatures, getToken, clearToken } from '@/services/api';
import { resetDesignV2 } from '@/services/designV2';
import { clearActiveGroupId, clearMyGroupsCache } from '@/services/activeGroup';
import { disablePushOnLogout } from '@/services/pushNotifications';

vi.mock('@/services/api', () => ({
  getMyFeatures: vi.fn(),
  getToken: vi.fn(),
  logoutSession: vi.fn(),
  clearToken: vi.fn()
}));

vi.mock('@/services/activeGroup', () => ({
  clearActiveGroupId: vi.fn(),
  clearMyGroupsCache: vi.fn()
}));

vi.mock('@/services/theme', () => ({
  syncThemeColorMeta: vi.fn(),
  prefersDarkTheme: vi.fn(() => true)
}));

vi.mock('@/services/pushNotifications', () => ({
  disablePushOnLogout: vi.fn(() => Promise.resolve()),
  stopPushIfAppUninstalled: vi.fn(() => Promise.resolve(false))
}));

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

const deferred = () => {
  let resolve;
  let reject;
  const promise = new Promise((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
};

describe('App.vue design-v2', () => {
  beforeEach(() => {
    resetDesignV2();
    localStorage.clear();
    localStorage.setItem('calciotto-theme', 'dark');
    document.getElementById('design-v2-fonts')?.remove();
    getMyFeatures.mockReset();
    getMyFeatures.mockResolvedValue([]);
    getToken.mockReset();
    getToken.mockReturnValue('token-a');
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    document.getElementById('design-v2-fonts')?.remove();
    vi.restoreAllMocks();
  });

  it('renders today\'s design while the features request is pending, then switches to v2', async () => {
    const features = deferred();
    getMyFeatures.mockReturnValue(features.promise);

    const { wrapper } = await mountAt('/');

    expect(wrapper.find('.app-container').exists()).toBe(true);
    expect(wrapper.find('.bottom-tab-bar').exists()).toBe(true);
    expect(wrapper.attributes('aria-busy')).toBeUndefined();
    expect(wrapper.classes()).not.toContain('design-v2');
    expect(document.getElementById('design-v2-fonts')).toBeNull();
    expect(getMyFeatures).toHaveBeenCalledTimes(1);

    features.resolve(['pwa-install', 'design-v2']);
    await flushPromises();

    expect(wrapper.find('.bottom-tab-bar').exists()).toBe(false);
    expect(wrapper.find('.tabbar').exists()).toBe(true);
    expect(wrapper.classes()).toContain('design-v2');
    expect(document.getElementById('design-v2-fonts').textContent).toContain('Big Shoulders Display');
  });

  it('renders today\'s design, without the fonts, once the feature comes back absent', async () => {
    getMyFeatures.mockResolvedValue(['pwa-install']);

    const { wrapper } = await mountAt('/');
    await flushPromises();

    expect(wrapper.find('.app-container').exists()).toBe(true);
    expect(wrapper.classes()).not.toContain('design-v2');
    expect(document.getElementById('design-v2-fonts')).toBeNull();
  });

  it('stays on today\'s design when the features request fails', async () => {
    getMyFeatures.mockRejectedValue(new Error('offline'));

    const { wrapper } = await mountAt('/');
    await flushPromises();

    expect(wrapper.find('.app-container').exists()).toBe(true);
    expect(wrapper.classes()).not.toContain('design-v2');
    expect(console.error).toHaveBeenCalled();
  });

  it('asks again in the background on the next navigation after a failure', async () => {
    getMyFeatures.mockRejectedValueOnce(new Error('offline'));
    const { wrapper, router } = await mountAt('/');
    await flushPromises();
    expect(wrapper.classes()).not.toContain('design-v2');

    const retry = deferred();
    getMyFeatures.mockReturnValueOnce(retry.promise);
    await router.push('/profile');
    await flushPromises();
    expect(wrapper.find('.app-container').exists()).toBe(true);
    expect(wrapper.classes()).not.toContain('design-v2');

    retry.resolve(['design-v2']);
    await flushPromises();
    expect(getMyFeatures).toHaveBeenCalledTimes(2);
    expect(wrapper.classes()).toContain('design-v2');
  });

  it.each(['/login', '/signup', '/forgot-password', '/reset-password'])(
    'keeps %s on the current design, with no request',
    async (path) => {
      const { wrapper } = await mountAt(path);
      await flushPromises();
      expect(wrapper.classes()).not.toContain('design-v2');
      expect(getMyFeatures).not.toHaveBeenCalled();
    }
  );

  it('drops the class on a public page even for a v2 player, without refetching on the way back', async () => {
    getMyFeatures.mockResolvedValue(['design-v2']);
    const { wrapper, router } = await mountAt('/');
    await flushPromises();
    expect(wrapper.classes()).toContain('design-v2');

    await router.push('/login');
    await flushPromises();
    expect(wrapper.classes()).not.toContain('design-v2');

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

  it('ignores a features response that resolves after logout', async () => {
    const features = deferred();
    getMyFeatures.mockReturnValue(features.promise);
    const { wrapper } = await mountAt('/');

    await wrapper.vm.logout();

    expect(clearToken).toHaveBeenCalled();
    expect(clearActiveGroupId).toHaveBeenCalled();
    expect(clearMyGroupsCache).toHaveBeenCalled();
    expect(disablePushOnLogout).toHaveBeenCalled();

    features.resolve(['design-v2']);
    await flushPromises();

    expect(wrapper.vm.$route.name).toBe('Login');
    expect(wrapper.classes()).not.toContain('design-v2');
    expect(document.getElementById('design-v2-fonts')).toBeNull();
  });

  it('asks again for the next account after logout', async () => {
    getMyFeatures.mockResolvedValueOnce([]);
    const { wrapper, router } = await mountAt('/');
    await flushPromises();
    expect(wrapper.classes()).not.toContain('design-v2');

    await wrapper.vm.logout();
    await flushPromises();
    expect(wrapper.vm.$route.name).toBe('Login');

    getToken.mockReturnValue('token-b');
    getMyFeatures.mockResolvedValueOnce(['design-v2']);
    await router.push('/');
    await flushPromises();

    expect(getMyFeatures).toHaveBeenCalledTimes(2);
    expect(wrapper.classes()).toContain('design-v2');
  });

  it('never stores the flag in localStorage', async () => {
    getMyFeatures.mockResolvedValue(['design-v2']);
    await mountAt('/');
    await flushPromises();
    expect(Object.keys(localStorage)).toEqual(['calciotto-theme']);
  });
});

describe('App.vue v2 season selector', () => {
  beforeEach(() => {
    resetDesignV2();
    localStorage.clear();
    localStorage.setItem('calciotto-theme', 'dark');
    document.getElementById('design-v2-fonts')?.remove();
    getMyFeatures.mockReset();
    getMyFeatures.mockResolvedValue(['design-v2']);
    getToken.mockReset();
    getToken.mockReturnValue('token-a');
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    document.getElementById('design-v2-fonts')?.remove();
    vi.restoreAllMocks();
  });

  it('shows a disabled No season yet control when the group has no seasons, in the same place', async () => {
    const { wrapper } = await mountAt('/');
    await flushPromises();
    wrapper.vm.v2Context.groups = [{ id: 'test-group', name: 'Test' }];
    wrapper.vm.v2Context.groupId = 'test-group';
    wrapper.vm.v2Context.seasons = [];
    wrapper.vm.v2Context.season = '';
    await wrapper.vm.$nextTick();

    expect(wrapper.vm.v2SeasonLabel).toBe('No season yet');
    expect(wrapper.vm.v2HasSeasons).toBe(false);
    const seasonLabel = wrapper.findAll('label.ctx').at(1);
    expect(seasonLabel.classes()).toContain('is-disabled');
    expect(seasonLabel.find('.ctx-name').text()).toBe('No season yet');
    expect(seasonLabel.find('select').element.disabled).toBe(true);
    expect(seasonLabel.find('.chev').exists()).toBe(false);
    const groupLabel = wrapper.findAll('label.ctx').at(0);
    expect(groupLabel.find('.ctx-name').text()).toBe('Test');
    expect(groupLabel.classes()).not.toContain('is-disabled');
    expect(groupLabel.find('.chev').exists()).toBe(true);
  });

  it('keeps the live season label and an enabled select when seasons exist', async () => {
    const { wrapper } = await mountAt('/');
    await flushPromises();
    wrapper.vm.v2Context.groups = [{ id: 'g', name: 'Giovedi' }];
    wrapper.vm.v2Context.groupId = 'g';
    wrapper.vm.v2Context.seasons = ['2025-2026', '2026-2027'];
    wrapper.vm.v2Context.season = '2026-2027';
    await wrapper.vm.$nextTick();

    expect(wrapper.vm.v2SeasonLabel).toBe('2026-2027');
    const seasonLabel = wrapper.findAll('label.ctx').at(1);
    expect(seasonLabel.classes()).not.toContain('is-disabled');
    expect(seasonLabel.find('.ctx-name').text()).toBe('2026-2027');
    expect(seasonLabel.find('select').element.disabled).toBe(false);
    expect(seasonLabel.find('.chev').exists()).toBe(true);
  });
});

