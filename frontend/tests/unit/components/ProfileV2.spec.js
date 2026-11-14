import { mount, flushPromises } from '@vue/test-utils';
import ProfileV2 from '@/components/v2/ProfileV2.vue';
import {
  getPlayerProfile, getMyGroups, getMyFeatures, getTeamsByGroup, getInviteCode, getGroupMembers, removeMember
} from '@/services/api';
import { getInstallTracker } from '@/services/pwaInstall';

jest.mock('@/services/api', () => ({
  getPlayerProfile: jest.fn(),
  getMyGroups: jest.fn(),
  getMyFeatures: jest.fn(),
  getSeasons: jest.fn(() => Promise.resolve([])),
  getTeamsByGroup: jest.fn(),
  getInviteCode: jest.fn(),
  getGroupMembers: jest.fn(),
  removeMember: jest.fn(),
  updateMemberRole: jest.fn(),
  setFavoriteGroup: jest.fn(),
  updateMyName: jest.fn(),
  deleteMyAccount: jest.fn(),
  clearToken: jest.fn(),
  updateTeam: jest.fn(),
  getNotificationPreferences: jest.fn(() => Promise.resolve({})),
  updateNotificationPreferences: jest.fn(),
  forgotPassword: jest.fn(),
  getToken: jest.fn(() => `x.${btoa(JSON.stringify({ player_id: 'me' }))}.y`)
}));

jest.mock('@/services/pushNotifications', () => ({
  ...jest.requireActual('@/services/pushNotifications'),
  getPushState: jest.fn(() => Promise.resolve('off'))
}));

jest.mock('@/services/pwaInstall', () => ({
  ...jest.requireActual('@/services/pwaInstall'),
  getInstallTracker: jest.fn()
}));

const GROUP_ID = 'group-uuid';

async function mountProfile({ routeName = 'Profile', features = [], installState = 'unavailable', members = [] } = {}) {
  getPlayerProfile.mockResolvedValue({
    Overall: { Name: 'Me' },
    PerGroup: [{ GroupID: GROUP_ID, GroupName: 'Giovedi', Played: 3, Points: 6 }],
    Seasons: []
  });
  getMyGroups.mockResolvedValue([{ id: GROUP_ID, name: 'Giovedi', role: 'admin', is_favorite: true }]);
  getMyFeatures.mockResolvedValue(features);
  getTeamsByGroup.mockResolvedValue([]);
  getInviteCode.mockResolvedValue({ invite_code: 'ABC' });
  getGroupMembers.mockResolvedValue(members);
  getInstallTracker.mockReturnValue({ state: () => installState, subscribe: () => () => {}, prompt: jest.fn() });

  const router = { replace: jest.fn() };
  const wrapper = mount(ProfileV2, {
    global: {
      provide: { v2Context: { confirming: false } },
      mocks: { $route: { name: routeName, params: { id: GROUP_ID }, fullPath: '/profile' }, $router: router },
      stubs: { 'router-link': { template: '<a :data-to="to"><slot /></a>', props: ['to'] }, TeamColourPicker: true }
    }
  });
  await flushPromises();
  return { wrapper, router };
}

beforeEach(() => jest.clearAllMocks());

describe('ProfileV2.vue hidden features', () => {
  it('hides Notifications and Install without their features', async () => {
    const { wrapper } = await mountProfile({ installState: 'prompt' });
    expect(wrapper.find('[data-to="/profile/notifications"]').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('Install Calciotto');
  });

  it('shows both with their features, Install only where the browser offers it', async () => {
    const { wrapper } = await mountProfile({ features: ['push-notifications', 'pwa-install'], installState: 'ios' });
    expect(wrapper.find('[data-to="/profile/notifications"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('Add to Home Screen');
  });

  it('sends a player without the feature from the notifications page back to the profile', async () => {
    const { router } = await mountProfile({ routeName: 'ProfileNotifications' });
    expect(router.replace).toHaveBeenCalledWith('/profile');
  });
});

describe('ProfileV2.vue removing a member', () => {
  const members = [
    { id: 'me', name: 'Me', role: 'admin', has_account: true },
    { id: 'luca', name: 'Luca', role: 'member', has_account: true }
  ];

  it('offers it on every row but your own, and removes after confirming', async () => {
    const { wrapper } = await mountProfile({ routeName: 'ProfileGroupSettings', members });
    const removeButtons = wrapper.findAll('.btn-remove');
    expect(removeButtons).toHaveLength(1);

    removeMember.mockResolvedValue({});
    await removeButtons[0].trigger('click');
    await wrapper.find('.confirm-sheet .btn-danger').trigger('click');
    await flushPromises();

    expect(removeMember).toHaveBeenCalledWith(GROUP_ID, 'luca');
    expect(getGroupMembers).toHaveBeenCalledTimes(2);
  });

  it("shows the backend's refusal", async () => {
    const { wrapper } = await mountProfile({ routeName: 'ProfileGroupSettings', members });
    removeMember.mockRejectedValue({ response: { data: { error: 'cannot remove this member' } } });
    await wrapper.find('.btn-remove').trigger('click');
    await wrapper.find('.confirm-sheet .btn-danger').trigger('click');
    await flushPromises();
    expect(wrapper.text()).toContain('cannot remove this member');
  });
});

describe('ProfileV2.vue notifications switch', () => {
  it('cannot be turned on outside the installed app', async () => {
    const { getPushState } = require('@/services/pushNotifications');
    getPushState.mockResolvedValue('needs-install');
    const { wrapper } = await mountProfile({ routeName: 'ProfileNotifications', features: ['push-notifications'] });
    const toggle = wrapper.find('[aria-label="Notifications on this device"]');
    expect(toggle.attributes('disabled')).toBeDefined();
    expect(wrapper.text()).toContain('Install the app first');
  });
});
