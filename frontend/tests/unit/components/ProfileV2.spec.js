import { mount, flushPromises } from '@vue/test-utils';
import ProfileV2 from '@/components/v2/ProfileV2.vue';
import {
  getPlayerProfile, getMyGroups, getMyFeatures, getTeamsByGroup, getInviteCode, getGroupMembers, removeMember, updateGroupDetails,
  getMyPositions, updateMyPositions
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
  getMyPositions: vi.fn(() => Promise.resolve([])),
  updateMyPositions: vi.fn(),
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
  routeName = 'Profile', features = [], installState = 'unavailable', members = [], groupRole = 'admin', groupDetails = {}, perGroup = { Played: 3, Points: 6 },
} = {}) {
  getPlayerProfile.mockResolvedValue({
    Overall: { Name: 'Me' },
    PerGroup: [{ GroupID: GROUP_ID, GroupName: 'Giovedi', ...perGroup }],
    Seasons: []
  });
  getMyGroups.mockResolvedValue([{ id: GROUP_ID, name: 'Giovedi', role: groupRole, is_favorite: true, ...groupDetails }]);
  getMyFeatures.mockResolvedValue(features);
  getTeamsByGroup.mockResolvedValue([]);
  getInviteCode.mockResolvedValue({ invite_code: 'ABC' });
  getGroupMembers.mockResolvedValue(members);
  getInstallTracker.mockReturnValue({ state: () => installState, subscribe: () => () => {}, prompt: vi.fn() });

  const router = { replace: vi.fn() };
  const context = { confirming: false, pageAction: null };
  const wrapper = mount(ProfileV2, {
    global: {
      provide: { v2Context: context },
      mocks: { $route: { name: routeName, params: { id: GROUP_ID }, fullPath: '/profile' }, $router: router },
      stubs: { 'router-link': { template: '<a :data-to="to"><slot /></a>', props: ['to'] } }
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
    expect(wrapper.text()).not.toContain('Install Agoran');
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

  it('lists MVP announced right under Final score', async () => {
    const { wrapper } = await mountProfile({ routeName: 'ProfileNotifications', features: ['push-notifications'] });
    const text = wrapper.text();
    expect(text).toContain('MVP announced');
    expect(text.indexOf('Final score')).toBeLessThan(text.indexOf('MVP announced'));
    expect(text.indexOf('MVP announced')).toBeLessThan(text.indexOf('Predictions open'));
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

  describe('gear entries', () => {
    it('shows the account gear on the profile home, with no text link', async () => {
      const { wrapper } = await mountProfile({ routeName: 'Profile' });
      const gear = wrapper.find('.ident a[data-to="/profile/account"]');
      expect(gear.exists()).toBe(true);
      expect(gear.attributes('aria-label')).toBe('Manage account');
      expect(gear.attributes('title')).toBe('Manage account');
      expect(gear.classes()).toContain('group-gear');
      expect(gear.text()).toBe('');
      expect(wrapper.text()).not.toContain('Manage account');
    });

    it('keeps the group-row gear for admins only', async () => {
      const admin = await mountProfile({ routeName: 'Profile' });
      const gear = admin.wrapper.find(`.group-row a[data-to="/profile/groups/${GROUP_ID}/settings"]`);
      expect(gear.attributes('aria-label')).toBe('Group settings');
      expect(gear.attributes('title')).toBe('Group settings');
      const member = await mountProfile({ routeName: 'Profile', groupRole: 'member' });
      expect(member.wrapper.find('.group-row .group-gear').exists()).toBe(false);
    });

    it('sets the page action on the group page for an admin, with no body button', async () => {
      const { wrapper } = await mountProfile({ routeName: 'ProfileGroup', groupRole: 'admin' });
      expect(wrapper.vm.v2Context.pageAction).toEqual({ to: `/profile/groups/${GROUP_ID}/settings`, label: 'Group settings' });
      expect(wrapper.find('a.btn').exists()).toBe(false);
      expect(wrapper.text()).not.toContain('Group settings');
    });

    it('sets no page action for a non-admin, nor on other pages', async () => {
      const member = await mountProfile({ routeName: 'ProfileGroup', groupRole: 'member' });
      expect(member.wrapper.vm.v2Context.pageAction).toBeNull();
      const home = await mountProfile({ routeName: 'Profile' });
      expect(home.wrapper.vm.v2Context.pageAction).toBeNull();
    });

    it('clears the page action on leave', async () => {
      const { wrapper } = await mountProfile({ routeName: 'ProfileGroup', groupRole: 'admin' });
      const context = wrapper.vm.v2Context;
      expect(context.pageAction).not.toBeNull();
      wrapper.unmount();
      expect(context.pageAction).toBeNull();
    });
  });
});

describe('ProfileV2.vue preferred positions (Account)', () => {
  const buttons = (wrapper) => wrapper.findAll('.account-positions .pos-pick');
  const state = (wrapper) => buttons(wrapper).map((b) => `${b.text()}:${b.attributes('aria-pressed')}`);
  const toast = (wrapper) => wrapper.find('.toast-message');

  it('shows four toggles in a labelled group, with the saved ones pressed and ticked', async () => {
    getMyPositions.mockResolvedValueOnce(['MID', 'DEF']);
    const { wrapper } = await mountProfile({ routeName: 'ProfileAccount' });
    const section = wrapper.get('.account-positions');
    expect(section.get('h2').text()).toBe('Positions');
    expect(section.get('[role="group"]').attributes('aria-label')).toBe('Where you like to play');
    expect(state(wrapper)).toEqual(['GK:false', '✓DEF:true', '✓MID:true', 'FWD:false']);
    expect(buttons(wrapper)[1].classes()).toContain('is-on');
    expect(buttons(wrapper)[1].find('.tick').attributes('aria-hidden')).toBe('true');
    expect(section.text()).toContain('Pick one or more. Your groups see them, and admins use them to balance teams.');
    // Under the name, before the email.
    const headings = wrapper.findAll('.sec-head h2').map((h) => h.text());
    expect(headings.indexOf('Positions')).toBe(headings.indexOf('Identity') + 1);
  });

  it('is not loaded outside the Account page', async () => {
    await mountProfile({ routeName: 'Profile' });
    expect(getMyPositions).not.toHaveBeenCalled();
  });

  it('saves the whole set at each tap, in canonical order, and says so', async () => {
    getMyPositions.mockResolvedValueOnce(['MID']);
    updateMyPositions.mockImplementation((list) => Promise.resolve(list));
    const { wrapper } = await mountProfile({ routeName: 'ProfileAccount' });
    await buttons(wrapper)[0].trigger('click');
    // Pressed at once, before the answer.
    expect(buttons(wrapper)[0].attributes('aria-pressed')).toBe('true');
    await flushPromises();
    expect(updateMyPositions).toHaveBeenLastCalledWith(['GK', 'MID']);
    expect(toast(wrapper).text()).toContain('Positions saved.');
    expect(toast(wrapper).classes()).toContain('success');

    await buttons(wrapper)[2].trigger('click');
    await flushPromises();
    expect(updateMyPositions).toHaveBeenLastCalledWith(['GK']);
    await buttons(wrapper)[0].trigger('click');
    await flushPromises();
    // No position at all is allowed.
    expect(updateMyPositions).toHaveBeenLastCalledWith([]);
    expect(state(wrapper)).toEqual(['GK:false', 'DEF:false', 'MID:false', 'FWD:false']);
  });

  it('puts the buttons back and shows the server\'s reason when a save is refused', async () => {
    getMyPositions.mockResolvedValueOnce(['DEF']);
    updateMyPositions.mockRejectedValueOnce({ response: { data: { error: 'invalid position "X": use GK, DEF, MID or FWD' } } });
    const { wrapper } = await mountProfile({ routeName: 'ProfileAccount' });
    await buttons(wrapper)[3].trigger('click');
    await flushPromises();
    expect(state(wrapper)).toEqual(['GK:false', '✓DEF:true', 'MID:false', 'FWD:false']);
    expect(toast(wrapper).classes()).toContain('error');
    expect(toast(wrapper).text()).toContain('invalid position "X": use GK, DEF, MID or FWD');
  });

  it('sends overlapping saves one after the other and keeps the latest tap', async () => {
    getMyPositions.mockResolvedValueOnce([]);
    let firstResolve;
    updateMyPositions
      .mockImplementationOnce(() => new Promise((resolve) => { firstResolve = resolve; }))
      .mockImplementationOnce((list) => Promise.resolve(list));
    const { wrapper } = await mountProfile({ routeName: 'ProfileAccount' });
    await buttons(wrapper)[0].trigger('click');
    await buttons(wrapper)[1].trigger('click');
    await flushPromises();
    // The second save waits for the first answer.
    expect(updateMyPositions).toHaveBeenCalledTimes(1);
    firstResolve(['GK']);
    await flushPromises();
    expect(updateMyPositions).toHaveBeenCalledTimes(2);
    expect(updateMyPositions).toHaveBeenLastCalledWith(['GK', 'DEF']);
    expect(state(wrapper)).toEqual(['✓GK:true', '✓DEF:true', 'MID:false', 'FWD:false']);
  });

  it('disables the toggles when the positions could not be loaded', async () => {
    getMyPositions.mockRejectedValueOnce(new Error('offline'));
    const { wrapper } = await mountProfile({ routeName: 'ProfileAccount' });
    buttons(wrapper).forEach((b) => expect(b.attributes('disabled')).toBeDefined());
    expect(wrapper.text()).toContain("Couldn't load your positions.");
    await buttons(wrapper)[0].trigger('click');
    expect(updateMyPositions).not.toHaveBeenCalled();
  });
});

describe('ProfileV2.vue MVP', () => {
  it('labels the awards stat MVP', async () => {
    const { wrapper } = await mountProfile();
    const labels = wrapper.findAll('.stat').map((stat) => stat.text());
    expect(labels.some((text) => /MVP$/.test(text))).toBe(true);
    expect(wrapper.text()).not.toContain('MOTM');
    expect(wrapper.find('.stats').text()).not.toContain('MVP of the Match');
  });

  it('names the titles MVP of the Match in the account deletion text', async () => {
    const { wrapper } = await mountProfile({ routeName: 'ProfileAccount' });
    expect(wrapper.find('.danger').text()).toContain('Match history (goals, MVP of the Match titles) stays');
  });
});

describe('ProfileV2.vue group counts', () => {
  it.each([
    [{ Played: 3, Points: 6 }, '3 matches · 6 pts'],
    [{ Played: 1, Points: 1 }, '1 match · 1 pt'],
    [{ Played: 0, Points: 0 }, '0 matches · 0 pts'],
    [{ Played: 2, Points: 1 }, '2 matches · 1 pt'],
  ])('writes %j as « %s »', async (perGroup, text) => {
    const { wrapper } = await mountProfile({ perGroup });
    expect(wrapper.find('.meta').text()).toBe(text);
  });
});
