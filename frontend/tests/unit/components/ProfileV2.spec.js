import { mount, flushPromises } from '@vue/test-utils';
import ProfileV2 from '@/components/v2/ProfileV2.vue';
import {
  getPlayerProfile, getMyGroups, getMyFeatures, getTeamsByGroup, getInviteCode, getGroupMembers, removeMember, updateGroupDetails
} from '@/services/api';
import { getPushState } from '@/services/pushNotifications';
import { getInstallTracker } from '@/services/pwaInstall';

vi.mock('@/services/api', () => ({
  getPlayerProfile: vi.fn(),
  getMyGroups: vi.fn(),
  getMyFeatures: vi.fn(),
  getSeasons: vi.fn(() => Promise.resolve([])),
  getTeamsByGroup: vi.fn(),
  getInviteCode: vi.fn(),
  getGroupMembers: vi.fn(),
  updateGroupDetails: vi.fn(),
  removeMember: vi.fn(),
  updateMemberRole: vi.fn(),
  setFavoriteGroup: vi.fn(),
  updateMyName: vi.fn(),
  deleteMyAccount: vi.fn(),
  clearToken: vi.fn(),
  updateTeam: vi.fn(),
  getNotificationPreferences: vi.fn(() => Promise.resolve({})),
  updateNotificationPreferences: vi.fn(),
  forgotPassword: vi.fn(),
  getMyPasskeys: vi.fn(() => Promise.resolve([])),
  removeMyPasskey: vi.fn(),
  rotateGroupInvitation: vi.fn(),
  revokeGroupInvitation: vi.fn(),
  getToken: vi.fn(() => `x.${btoa(JSON.stringify({ player_id: 'me' }))}.y`)
}));

vi.mock('@/services/pushNotifications', async (importOriginal) => ({
  ...(await importOriginal()),
  getPushState: vi.fn(() => Promise.resolve('off'))
}));

vi.mock('@/services/pwaInstall', async (importOriginal) => ({
  ...(await importOriginal()),
  getInstallTracker: vi.fn()
}));

const GROUP_ID = 'group-uuid';

async function mountProfile({
  routeName = 'Profile', features = [], installState = 'unavailable', members = [], groupRole = 'admin', groupDetails = {},
} = {}) {
  getPlayerProfile.mockResolvedValue({
    Overall: { Name: 'Me' },
    PerGroup: [{ GroupID: GROUP_ID, GroupName: 'Giovedi', Played: 3, Points: 6 }],
    Seasons: []
  });
  getMyGroups.mockResolvedValue([{ id: GROUP_ID, name: 'Giovedi', role: groupRole, is_favorite: true, ...groupDetails }]);
  getMyFeatures.mockResolvedValue(features);
  getTeamsByGroup.mockResolvedValue([]);
  getInviteCode.mockResolvedValue({ invite_code: 'ABC' });
  getGroupMembers.mockResolvedValue(members);
  getInstallTracker.mockReturnValue({ state: () => installState, subscribe: () => () => {}, prompt: vi.fn() });

  const router = { replace: vi.fn() };
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

beforeEach(() => vi.clearAllMocks());

describe('ProfileV2.vue group details', () => {
  const details = {
    description: 'Thursday evening football',
    website_url: 'https://example.test/group',
    website_label: 'Request to join our group chat',
  };

  it('shows the description and custom link to a group member', async () => {
    const { wrapper } = await mountProfile({ routeName: 'ProfileGroup', groupRole: 'member', groupDetails: details });
    expect(wrapper.text()).toContain(details.description);
    const link = wrapper.get('.group-website');
    expect(link.text()).toContain(details.website_label);
    expect(link.attributes('href')).toBe(details.website_url);
    expect(wrapper.find('.group-details-form').exists()).toBe(false);
  });

  it('saves the optional group details for an administrator', async () => {
    const { wrapper } = await mountProfile({ routeName: 'ProfileGroupSettings' });
    updateGroupDetails.mockResolvedValue({ id: GROUP_ID, ...details });
    await wrapper.get('#group-description').setValue(details.description);
    await wrapper.get('#group-website-url').setValue(details.website_url);
    await wrapper.get('#group-website-label').setValue(details.website_label);
    await wrapper.get('.group-details-form').trigger('submit');
    await flushPromises();
    expect(updateGroupDetails).toHaveBeenCalledWith(GROUP_ID, details);
    expect(wrapper.get('[role="status"]').text()).toBe('Group details saved.');
  });

  it('redirects a member who visits the group settings', async () => {
    const { wrapper, router } = await mountProfile({ routeName: 'ProfileGroupSettings', groupRole: 'member' });
    expect(router.replace).toHaveBeenCalledWith(`/profile/groups/${GROUP_ID}`);
    expect(wrapper.find('.group-details-form').exists()).toBe(false);
  });
});

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
    getPushState.mockResolvedValue('needs-install');
    const { wrapper } = await mountProfile({ routeName: 'ProfileNotifications', features: ['push-notifications'] });
    const toggle = wrapper.find('[aria-label="Notifications on this device"]');
    expect(toggle.attributes('disabled')).toBeDefined();
    expect(wrapper.text()).toContain('Install the app first');
  });
});

describe('ProfileV2.vue invitation and passkeys', () => {
  it('uses the v2 invitation card in group settings and applies its updates', async () => {
    const { wrapper } = await mountProfile({ routeName: 'ProfileGroupSettings' });
    const invitation = wrapper.findComponent({ name: 'InvitationV2' });
    expect(invitation.exists()).toBe(true);
    expect(wrapper.findComponent({ name: 'InvitationActions' }).exists()).toBe(false);
    expect(invitation.props('code')).toBe('ABC');
    expect(invitation.props('shareText')).toContain('ABC');
    invitation.vm.$emit('updated', { invite_code: '' });
    await flushPromises();
    expect(invitation.props('code')).toBe('');
  });

  it("puts the shell in confirming mode while a child's confirmation is open", async () => {
    const { wrapper } = await mountProfile({ routeName: 'ProfileGroupSettings' });
    const context = wrapper.vm.v2Context;
    wrapper.findComponent({ name: 'InvitationV2' }).vm.$emit('confirming', true);
    expect(context.confirming).toBe(true);
    wrapper.findComponent({ name: 'InvitationV2' }).vm.$emit('confirming', false);
    expect(context.confirming).toBe(false);
  });

  it('shows the Passkeys section only with both design-v2 and passkeys', async () => {
    const without = await mountProfile({ routeName: 'ProfileAccount', features: ['design-v2'] });
    expect(without.wrapper.findComponent({ name: 'PasskeysV2' }).exists()).toBe(false);
    const { wrapper } = await mountProfile({ routeName: 'ProfileAccount', features: ['design-v2', 'passkeys'] });
    expect(wrapper.findComponent({ name: 'PasskeysV2' }).exists()).toBe(true);
    const headings = wrapper.findAll('h2').map((h) => h.text());
    expect(headings.indexOf('Passkeys')).toBe(headings.indexOf('Password') + 1);
  });
});
