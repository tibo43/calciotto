import { mount, flushPromises } from '@vue/test-utils';
import NotificationsCard from '@/components/NotificationsCard.vue';
import { PUSH_STATE, getPushState, enablePush, disablePush, resyncPushSubscription } from '@/services/pushNotifications';

jest.mock('@/services/pushNotifications', () => {
  const actual = jest.requireActual('@/services/pushNotifications');
  return {
    ...actual,
    getPushState: jest.fn(),
    enablePush: jest.fn(),
    disablePush: jest.fn(),
    resyncPushSubscription: jest.fn()
  };
});

async function mountIn(state) {
  getPushState.mockResolvedValue(state);
  const wrapper = mount(NotificationsCard);
  await flushPromises();
  return wrapper;
}

beforeEach(() => jest.clearAllMocks());

describe('NotificationsCard.vue', () => {
  it('turns notifications on from the switch', async () => {
    enablePush.mockResolvedValue(PUSH_STATE.ON);
    const wrapper = await mountIn(PUSH_STATE.OFF);
    expect(wrapper.find('[role="switch"]').attributes('aria-checked')).toBe('false');

    await wrapper.find('[role="switch"]').trigger('click');
    await flushPromises();
    expect(enablePush).toHaveBeenCalledTimes(1);
    expect(wrapper.find('[role="switch"]').attributes('aria-checked')).toBe('true');
  });

  it('turns them off, and re-sends the subscription on load when on', async () => {
    disablePush.mockResolvedValue(PUSH_STATE.OFF);
    const wrapper = await mountIn(PUSH_STATE.ON);
    expect(resyncPushSubscription).toHaveBeenCalledTimes(1);

    await wrapper.find('[role="switch"]').trigger('click');
    await flushPromises();
    expect(disablePush).toHaveBeenCalledTimes(1);
    expect(wrapper.find('[role="switch"]').attributes('aria-checked')).toBe('false');
  });

  it.each([
    [PUSH_STATE.NEEDS_INSTALL, 'Add to Home Screen'],
    [PUSH_STATE.DENIED, 'settings'],
    [PUSH_STATE.UNSUPPORTED, "can't receive"]
  ])('explains %s with no switch', async (state, text) => {
    const wrapper = await mountIn(state);
    expect(wrapper.text()).toContain(text);
    expect(wrapper.find('[role="switch"]').exists()).toBe(false);
  });

  it("shows the backend's refusal and keeps the state", async () => {
    enablePush.mockRejectedValue({ response: { data: { error: 'push notifications are not enabled for this player' } } });
    const wrapper = await mountIn(PUSH_STATE.OFF);
    await wrapper.find('[role="switch"]').trigger('click');
    await flushPromises();
    expect(wrapper.find('.error-message').text()).toContain('not enabled');
    expect(wrapper.find('[role="switch"]').attributes('aria-checked')).toBe('false');
  });
});
