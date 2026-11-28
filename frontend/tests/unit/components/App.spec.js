import { shallowMount, flushPromises } from '@vue/test-utils';
import { createRouter, createMemoryHistory } from 'vue-router';
import App from '@/App.vue';
import PlayerGuide from '@/components/PlayerGuide.vue';
import GearLink from '@/components/v2/GearLink.vue';
import LocaleSaveErrorToast from '@/components/LocaleSaveErrorToast.vue';
import { getMyFeatureStatus, getSessionKey, clearToken, completeGuide } from '@/services/api';
import { resetFeatureStatus } from '@/services/featureStatus';
import { clearActiveGroupId, clearMyGroupsCache } from '@/services/activeGroup';
import { disablePushOnLogout } from '@/services/pushNotifications';
import { resetSavedLocale, syncSavedLocale } from '@/services/playerLocale';

vi.mock('@/services/api', () => ({
  getMyFeatureStatus: vi.fn(),
  getSessionKey: vi.fn(),
  completeGuide: vi.fn(),
  logoutSession: vi.fn(),
  clearToken: vi.fn()
}));

vi.mock('@/services/playerLocale', () => ({
  syncSavedLocale: vi.fn(() => Promise.resolve()),
  resetSavedLocale: vi.fn(),
  onLocaleSaveError: vi.fn(() => () => {})
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

describe('App.vue shell', () => {
  beforeEach(() => {
    resetFeatureStatus();
    localStorage.clear();
    localStorage.setItem('calciotto-theme', 'dark');
    getMyFeatureStatus.mockReset();
    getMyFeatureStatus.mockResolvedValue({ features: [] });
    getSessionKey.mockReset();
    getSessionKey.mockReturnValue('token-a');
    syncSavedLocale.mockClear();
    resetSavedLocale.mockClear();
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('restores the saved language on a signed-in page, not on a public one', async () => {
    await mountAt('/profile');
    expect(syncSavedLocale).toHaveBeenCalled();
    expect(resetSavedLocale).not.toHaveBeenCalled();
  });

  it('shows the browser\'s language on a public page, whatever was chosen', async () => {
    const { router } = await mountAt('/login');
    expect(resetSavedLocale).toHaveBeenCalled();
    expect(syncSavedLocale).not.toHaveBeenCalled();

    syncSavedLocale.mockClear();
    resetSavedLocale.mockClear();
    await router.push('/profile');
    await flushPromises();
    expect(syncSavedLocale).toHaveBeenCalled();
    await router.push('/signup');
    await flushPromises();
    expect(resetSavedLocale).toHaveBeenCalled();
  });

  it('mounts the failed-language-save toast once, on every signed-in page', async () => {
    for (const path of ['/', '/predictions', '/profile']) {
      const { wrapper } = await mountAt(path);
      expect(wrapper.findAllComponents(LocaleSaveErrorToast)).toHaveLength(1);
      wrapper.unmount();
    }
  });

  it('paints the signed-in app straight away, without waiting for the features request', async () => {
    const features = deferred();
    getMyFeatureStatus.mockReturnValue(features.promise);

    const { wrapper } = await mountAt('/');

    expect(wrapper.find('.app-container').exists()).toBe(true);
    expect(wrapper.find('.tabbar').exists()).toBe(true);
    expect(wrapper.find('.bottom-tab-bar').exists()).toBe(false);
    expect(wrapper.classes()).toContain('design-v2');
    expect(getMyFeatureStatus).toHaveBeenCalledTimes(1);

    features.resolve({ features: [] });
    await flushPromises();
    expect(wrapper.classes()).toContain('design-v2');
  });

  it('stays on the redesign when the features request fails', async () => {
    getMyFeatureStatus.mockRejectedValue(new Error('offline'));

    const { wrapper } = await mountAt('/');
    await flushPromises();

    expect(wrapper.find('.app-container').exists()).toBe(true);
    expect(wrapper.classes()).toContain('design-v2');
    expect(console.error).toHaveBeenCalled();
  });

  it.each(['/login', '/signup', '/forgot-password', '/reset-password'])(
    'paints %s in the v2 shell, with no menu, no theme toggle and no request',
    async (path) => {
      const { wrapper } = await mountAt(path);
      await flushPromises();
      expect(wrapper.classes()).toContain('design-v2');
      expect(wrapper.find('.topbar .brand').exists()).toBe(true);
      expect(wrapper.find('.topbar .ctx-pair').exists()).toBe(false);
      expect(wrapper.find('.top-navbar').exists()).toBe(false);
      expect(wrapper.find('.theme-toggle').exists()).toBe(false);
      expect(wrapper.find('.tabbar').exists()).toBe(false);
      expect(getMyFeatureStatus).not.toHaveBeenCalled();
    }
  );

  it('keeps the class from page to page, without refetching on the way back', async () => {
    const { wrapper, router } = await mountAt('/');
    await flushPromises();
    expect(wrapper.classes()).toContain('design-v2');

    await router.push('/signup');
    await flushPromises();
    expect(wrapper.classes()).toContain('design-v2');
    expect(wrapper.find('.tabbar').exists()).toBe(false);

    await router.push('/');
    await flushPromises();
    expect(wrapper.classes()).toContain('design-v2');
    expect(wrapper.find('.tabbar').exists()).toBe(true);
    expect(getMyFeatureStatus).toHaveBeenCalledTimes(1);
  });

  it('fetches features once per session across authenticated routes', async () => {
    const { router } = await mountAt('/');
    await flushPromises();

    await router.push('/profile');
    await flushPromises();
    await router.push('/predictions');
    await flushPromises();

    expect(getMyFeatureStatus).toHaveBeenCalledTimes(1);
  });

  it('ignores a features response that resolves after logout', async () => {
    const features = deferred();
    getMyFeatureStatus.mockReturnValue(features.promise);
    const { wrapper } = await mountAt('/');

    await wrapper.vm.logout();

    expect(clearToken).toHaveBeenCalled();
    expect(clearActiveGroupId).toHaveBeenCalled();
    expect(clearMyGroupsCache).toHaveBeenCalled();
    expect(disablePushOnLogout).toHaveBeenCalled();

    features.resolve({ features: [], guideCompleted: false });
    await flushPromises();

    // The late answer must not open the previous session's guide.
    expect(wrapper.vm.$route.name).toBe('Login');
    expect(wrapper.findComponent(PlayerGuide).exists()).toBe(false);
    expect(wrapper.find('.tabbar').exists()).toBe(false);
    expect(wrapper.find('.topbar .ctx-pair').exists()).toBe(false);
  });

  it('asks again for the next account after logout', async () => {
    const { wrapper, router } = await mountAt('/');
    await flushPromises();

    await wrapper.vm.logout();
    await flushPromises();
    expect(wrapper.vm.$route.name).toBe('Login');

    getSessionKey.mockReturnValue('token-b');
    getMyFeatureStatus.mockResolvedValueOnce({ features: [], guideCompleted: false });
    await router.push('/');
    await flushPromises();

    expect(getMyFeatureStatus).toHaveBeenCalledTimes(2);
    expect(wrapper.findComponent(PlayerGuide).props('required')).toBe(true);
  });

  it('goes back to the browser\'s language on logout', async () => {
    const { wrapper } = await mountAt('/profile');
    resetSavedLocale.mockClear();
    await wrapper.vm.logout();
    expect(clearToken).toHaveBeenCalled();
    expect(resetSavedLocale).toHaveBeenCalled();
  });

  it('stores nothing in localStorage', async () => {
    await mountAt('/');
    await flushPromises();
    expect(Object.keys(localStorage)).toEqual(['calciotto-theme']);
  });
});

describe('App.vue startup and player guide', () => {
  const guide = (wrapper) => wrapper.findComponent(PlayerGuide);

  beforeEach(() => {
    resetFeatureStatus();
    getMyFeatureStatus.mockReset();
    getSessionKey.mockReset();
    getSessionKey.mockReturnValue('token-a');
    completeGuide.mockReset();
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('opens the required guide once a slow features request answers', async () => {
    const features = deferred();
    getMyFeatureStatus.mockReturnValue(features.promise);

    const { wrapper } = await mountAt('/');
    expect(wrapper.find('.app-container').exists()).toBe(true);
    expect(guide(wrapper).exists()).toBe(false);

    features.resolve({ features: [], guideCompleted: false });
    await flushPromises();
    expect(guide(wrapper).props('required')).toBe(true);
  });

  it('keeps the required guide open when saving fails, and closes it once a retry succeeds', async () => {
    getMyFeatureStatus.mockResolvedValue({ features: [], guideCompleted: false });
    const { wrapper } = await mountAt('/');
    await flushPromises();

    completeGuide.mockRejectedValueOnce(new Error('offline'));
    guide(wrapper).vm.$emit('finish');
    await flushPromises();
    expect(guide(wrapper).props('error')).toContain('Could not save');

    completeGuide.mockResolvedValueOnce();
    guide(wrapper).vm.$emit('finish');
    await flushPromises();
    expect(guide(wrapper).exists()).toBe(false);
    expect(wrapper.find('.app-container').exists()).toBe(true);
    expect(completeGuide).toHaveBeenCalledTimes(2);
  });

  it('lets the player continue for this session after a failed save', async () => {
    getMyFeatureStatus.mockResolvedValue({ features: [], guideCompleted: false });
    const { wrapper, router } = await mountAt('/');
    await flushPromises();
    completeGuide.mockRejectedValueOnce(new Error('offline'));
    guide(wrapper).vm.$emit('finish');
    await flushPromises();

    guide(wrapper).vm.$emit('skip');
    await flushPromises();
    expect(guide(wrapper).exists()).toBe(false);

    // Not asked again while this session lasts.
    await router.push('/profile');
    await flushPromises();
    expect(guide(wrapper).exists()).toBe(false);
    expect(getMyFeatureStatus).toHaveBeenCalledTimes(1);
  });
});

describe('App.vue v2 season selector', () => {
  beforeEach(() => {
    resetFeatureStatus();
    localStorage.clear();
    localStorage.setItem('calciotto-theme', 'dark');
    document.getElementById('design-v2-fonts')?.remove();
    getMyFeatureStatus.mockReset();
    getMyFeatureStatus.mockResolvedValue({ features: ['design-v2'] });
    getSessionKey.mockReset();
    getSessionKey.mockReturnValue('token-a');
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    document.getElementById('design-v2-fonts')?.remove();
    vi.restoreAllMocks();
  });

  it('shows the page action as a gear on the title row, and drops it when cleared', async () => {
    const { wrapper } = await mountAt('/');
    await flushPromises();
    expect(wrapper.findComponent(GearLink).exists()).toBe(false);
    wrapper.vm.v2Context.pageTitle = 'Giovedi';
    wrapper.vm.v2Context.pageAction = { to: '/profile/groups/g/settings', label: 'Group settings' };
    await wrapper.vm.$nextTick();
    const gear = wrapper.findComponent(GearLink);
    expect(gear.exists()).toBe(true);
    expect(gear.props()).toEqual({ to: '/profile/groups/g/settings', label: 'Group settings' });
    expect(gear.classes()).toEqual(expect.arrayContaining(['hero-manage', 'page-action']));
    wrapper.vm.v2Context.pageAction = null;
    await wrapper.vm.$nextTick();
    expect(wrapper.findComponent(GearLink).exists()).toBe(false);
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

