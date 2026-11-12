import { mount, flushPromises } from '@vue/test-utils';
import NotificationsCard from '@/components/NotificationsCard.vue';
import { PUSH_STATE, getPushState, enablePush, disablePush, resyncPushSubscription } from '@/services/pushNotifications';
import { getNotificationPreferences, updateNotificationPreferences } from '@/services/api';

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

jest.mock('@/services/api', () => ({
  getNotificationPreferences: jest.fn(),
  updateNotificationPreferences: jest.fn()
}));

async function mountIn(state) {
  getPushState.mockResolvedValue(state);
  const wrapper = mount(NotificationsCard);
  await flushPromises();
  return wrapper;
}

const ALL_ON = {
  registration_open: true,
  team_assigned: true,
  result_recorded: true,
  predictions_open: true,
  predictions_settled: true,
  kickoff_reminder: true
};

function preferenceSwitches(wrapper) {
  return wrapper.findAll('.notifications-preferences [role="switch"]');
}

beforeEach(() => {
  jest.clearAllMocks();
  getNotificationPreferences.mockResolvedValue({ ...ALL_ON });
  updateNotificationPreferences.mockImplementation(async (updates) => ({ ...ALL_ON, ...updates }));
});

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
    expect(wrapper.find('.notifications-preferences').exists()).toBe(false);
    expect(getNotificationPreferences).not.toHaveBeenCalled();
  });

  it('keeps the six notifications hidden while the device is off', async () => {
    const wrapper = await mountIn(PUSH_STATE.OFF);
    expect(wrapper.find('.notifications-preferences').exists()).toBe(false);
    expect(getNotificationPreferences).not.toHaveBeenCalled();
  });

  it('lists each notification under its heading, all on, once the device is on', async () => {
    const wrapper = await mountIn(PUSH_STATE.ON);
    expect(wrapper.text()).toContain('Match');
    expect(wrapper.text()).toContain('Predictions');
    expect(wrapper.text()).toContain('Reminder');
    const labels = ['Sign-ups open', 'Your team', 'Final score', 'Predictions open', 'Predictions settled', 'Kick-off in 30 minutes'];
    labels.forEach((label) => expect(wrapper.text()).toContain(label));
    const switches = preferenceSwitches(wrapper);
    expect(switches).toHaveLength(6);
    switches.forEach((toggle) => expect(toggle.attributes('aria-checked')).toBe('true'));
    expect(getNotificationPreferences).toHaveBeenCalledTimes(1);
  });

  it('shows a type the player turned off, and saves a switch on its own', async () => {
    getNotificationPreferences.mockResolvedValue({ ...ALL_ON, kickoff_reminder: false });
    const wrapper = await mountIn(PUSH_STATE.ON);
    const switches = () => wrapper.findAll('.notifications-preferences [role="switch"]');
    expect(switches()[5].attributes('aria-checked')).toBe('false');
    expect(switches()[0].attributes('aria-checked')).toBe('true');

    updateNotificationPreferences.mockResolvedValue({ ...ALL_ON, kickoff_reminder: false, registration_open: false });
    await switches()[0].trigger('click');
    await flushPromises();
    expect(updateNotificationPreferences).toHaveBeenCalledWith({ registration_open: false });
    expect(switches()[0].attributes('aria-checked')).toBe('false');
    expect(switches()[5].attributes('aria-checked')).toBe('false');
  });

  it('puts the switch back when the save fails', async () => {
    updateNotificationPreferences.mockRejectedValue({ response: { data: { error: 'unknown notification type' } } });
    const wrapper = await mountIn(PUSH_STATE.ON);
    await wrapper.findAll('.notifications-preferences [role="switch"]')[0].trigger('click');
    await flushPromises();
    expect(wrapper.find('.error-message').text()).toContain('unknown notification type');
    expect(wrapper.findAll('.notifications-preferences [role="switch"]')[0].attributes('aria-checked')).toBe('true');
  });

  it('hides the list when the device is turned off and shows the saved choices again when it is turned back on', async () => {
    getNotificationPreferences.mockResolvedValue({ ...ALL_ON, team_assigned: false });
    enablePush.mockResolvedValue(PUSH_STATE.ON);
    disablePush.mockResolvedValue(PUSH_STATE.OFF);
    const wrapper = await mountIn(PUSH_STATE.ON);
    expect(wrapper.findAll('.notifications-preferences [role="switch"]')[1].attributes('aria-checked')).toBe('false');

    await wrapper.find('[aria-label="Notifications on this device"]').trigger('click');
    await flushPromises();
    expect(wrapper.find('.notifications-preferences').exists()).toBe(false);

    await wrapper.find('[aria-label="Notifications on this device"]').trigger('click');
    await flushPromises();
    expect(wrapper.find('.notifications-preferences').exists()).toBe(true);
    expect(wrapper.findAll('.notifications-preferences [role="switch"]')[1].attributes('aria-checked')).toBe('false');
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
