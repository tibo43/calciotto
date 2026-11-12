import { mount } from '@vue/test-utils';
import InstallAppCard from '@/components/InstallAppCard.vue';
import { getInstallTracker, INSTALL_STATE } from '@/services/pwaInstall';

jest.mock('@/services/pwaInstall', () => {
  const actual = jest.requireActual('@/services/pwaInstall');
  return { ...actual, getInstallTracker: jest.fn() };
});

// A tracker whose state the test sets, and whose subscribers it can notify.
function stubTracker(initialState) {
  let listener = null;
  const tracker = {
    state: () => initialState,
    subscribe: jest.fn((fn) => {
      listener = fn;
      return jest.fn();
    }),
    prompt: jest.fn(() => Promise.resolve('accepted')),
    emit: (state) => listener(state)
  };
  getInstallTracker.mockReturnValue(tracker);
  return tracker;
}

describe('InstallAppCard.vue', () => {
  it('renders nothing where the app is installed or cannot be', () => {
    stubTracker(INSTALL_STATE.INSTALLED);
    expect(mount(InstallAppCard).find('.install-app-card').exists()).toBe(false);
    stubTracker(INSTALL_STATE.UNAVAILABLE);
    expect(mount(InstallAppCard).find('.install-app-card').exists()).toBe(false);
  });

  it("opens Chrome's install dialog from its button", async () => {
    const tracker = stubTracker(INSTALL_STATE.PROMPT);
    const wrapper = mount(InstallAppCard);
    await wrapper.find('button').trigger('click');
    expect(tracker.prompt).toHaveBeenCalledTimes(1);
  });

  it('explains the Share menu on iOS, with no button to press', () => {
    stubTracker(INSTALL_STATE.IOS);
    const wrapper = mount(InstallAppCard);
    expect(wrapper.text()).toContain('Add to Home Screen');
    expect(wrapper.find('button').exists()).toBe(false);
  });

  it('appears when Chrome makes its offer after mount, and goes once installed', async () => {
    const tracker = stubTracker(INSTALL_STATE.UNAVAILABLE);
    const wrapper = mount(InstallAppCard);
    expect(wrapper.find('.install-app-card').exists()).toBe(false);

    tracker.emit(INSTALL_STATE.PROMPT);
    await wrapper.vm.$nextTick();
    expect(wrapper.find('.install-app-card').exists()).toBe(true);

    tracker.emit(INSTALL_STATE.INSTALLED);
    await wrapper.vm.$nextTick();
    expect(wrapper.find('.install-app-card').exists()).toBe(false);
  });
});
