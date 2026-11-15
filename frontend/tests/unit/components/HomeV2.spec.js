import { mount, flushPromises } from '@vue/test-utils';
import HomeV2 from '@/components/v2/HomeV2.vue';
import { getMatchesDetails, getMatchVotes, getMatchRegistrations, getToken } from '@/services/api';

jest.mock('@/services/api', () => ({
  getMatchesDetails: jest.fn(),
  getMatchRegistrations: jest.fn(() => Promise.resolve([])),
  getMatchVotes: jest.fn(),
  registerForMatch: jest.fn(),
  unregisterFromMatch: jest.fn(),
  getToken: jest.fn(() => '')
}));

const DAY = 24 * 60 * 60 * 1000;
const dayAgo = (n) => new Date(Date.now() - n * DAY).toISOString().slice(0, 10);
const played = (id, daysAgo) => ({
  ID: id, Date: dayAgo(daysAgo),
  Teams: [{ ID: 'a', Name: 'Black', Score: 1, Players: [{ ID: 'p' }] }, { ID: 'b', Name: 'White', Score: 0, Players: [{ ID: 'q' }] }]
});

async function mountHome(matches, isAdmin = false) {
  getMatchesDetails.mockResolvedValue(matches);
  getMatchVotes.mockResolvedValue(null);
  const wrapper = mount(HomeV2, {
    props: { activeGroupId: 'g', groupName: 'Group', season: '2026-2027', isAdmin },
    global: {
      provide: { v2Context: { confirming: false } },
      stubs: { 'router-link': { template: '<a :data-to="JSON.stringify(to)"><slot /></a>', props: ['to'] } }
    }
  });
  await flushPromises();
  return wrapper;
}

describe('HomeV2.vue with nothing scheduled', () => {
  it('shows the last match once: in the hero, not again at the top of Latest matches', async () => {
    const wrapper = await mountHome([played('last', 3), played('before', 10), played('oldest', 17)]);
    expect(wrapper.find('.hero-last').exists()).toBe(true);
    const listed = wrapper.findAll('.list .row').map((row) => JSON.parse(row.attributes('data-to')).params.id);
    expect(listed).toEqual(['before', 'oldest']);
  });

  it('hides Latest matches when the hero is the only match', async () => {
    const wrapper = await mountHome([played('only', 3)]);
    expect(wrapper.find('.hero-last').exists()).toBe(true);
    expect(wrapper.text()).not.toContain('Latest matches');
  });
});

describe('HomeV2.vue New match button', () => {
  it('floats for an admin, whatever the page shows', async () => {
    for (const matches of [[], [played('last', 3)], [{ ...played('next', -3), ScheduledAt: new Date(Date.now() + 3 * DAY).toISOString(), MaxPlayers: 16 }]]) {
      const admin = await mountHome(matches, true);
      expect(admin.find('.fab').attributes('aria-label')).toBe('New match');
    }
  });

  it('is absent for a member, who keeps the usual texts', async () => {
    const member = await mountHome([], false);
    expect(member.find('.fab').exists()).toBe(false);
    expect(member.text()).toContain('No matches yet');
    expect((await mountHome([played('last', 3)], false)).text()).toContain('No upcoming match yet');
  });
});

describe('HomeV2.vue with a match scheduled', () => {
  beforeEach(() => jest.clearAllMocks());

  const scheduled = (extra = {}) => ({
    ...played('next', -3), ScheduledAt: new Date(Date.now() + 3 * DAY).toISOString(),
    RegistrationOpensAt: new Date(Date.now() - DAY).toISOString(), MaxPlayers: 2, ...extra
  });

  it('drops the Man of the Match line and does not load the votes', async () => {
    const wrapper = await mountHome([scheduled(), played('last', 3)]);
    expect(wrapper.find('.motm-row').exists()).toBe(false);
    expect(getMatchVotes).not.toHaveBeenCalled();
  });

  it('turns Sign up into Join the waiting list once the places are taken', async () => {
    getMatchRegistrations.mockResolvedValueOnce([
      { PlayerID: 'x', IsWaiting: false }, { PlayerID: 'y', IsWaiting: false }, { PlayerID: 'z', IsWaiting: true }
    ]);
    const wrapper = await mountHome([scheduled()]);
    expect(wrapper.find('.hero .actions .btn').text()).toBe('Join the waiting list');
    expect(wrapper.find('.reg-you').text()).toContain('#2');
  });

  it("names the player's team once the teams are set", async () => {
    getToken.mockReturnValue(`x.${btoa(JSON.stringify({ player_id: 'me' }))}.y`);
    const teams = [{ ID: 'a', Name: 'Black', Players: [{ ID: 'me' }] }, { ID: 'b', Name: 'White', Players: [{ ID: 'q' }] }];
    const wrapper = await mountHome([scheduled({ RegistrationsClosedAt: new Date().toISOString(), Teams: teams })]);
    expect(wrapper.find('.hero-team-name').text()).toBe('Black');
    expect(wrapper.find('.hero-spots').exists()).toBe(false);
    getToken.mockReturnValue('');
  });
});
