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
  it('says there is no upcoming match and lists every finished one under Latest matches', async () => {
    const wrapper = await mountHome([played('last', 3), played('before', 10), played('oldest', 17)]);
    expect(wrapper.find('.hero-none .label').text()).toBe('Next match');
    expect(wrapper.find('.hero-none .hero-meta').text()).toBe('No upcoming match');
    const listed = wrapper.findAll('.list .row').map((row) => JSON.parse(row.attributes('data-to')).params.id);
    expect(listed).toEqual(['last', 'before', 'oldest']);
  });

  it('moves a match whose score is recorded out of Next match, to the top of Latest matches', async () => {
    const recorded = {
      ...played('today', 0), ScheduledAt: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
      ResultRecordedAt: new Date(Date.now() - 60 * 1000).toISOString(), MaxPlayers: 16,
    };
    const wrapper = await mountHome([recorded, played('before', 7)], true);
    expect(wrapper.find('.hero-none').exists()).toBe(true);
    expect(wrapper.text()).not.toContain('Manage match');
    expect(wrapper.text()).not.toContain('Sign-ups');
    const rows = wrapper.findAll('.list .row');
    expect(JSON.parse(rows[0].attributes('data-to')).params.id).toBe('today');
    expect(rows[0].find('.score').text()).toBe('1–0');
    expect(wrapper.find('.fab').exists()).toBe(true);
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
    expect((await mountHome([played('last', 3)], false)).text()).toContain('No upcoming match');
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
    expect(wrapper.find('.hero-team-line').text()).toBe('You play for Black');
    expect(wrapper.find('.hero-team-mine').text()).toBe('Black');
    expect(wrapper.find('.hero-spots').exists()).toBe(false);
    getToken.mockReturnValue('');
  });
});

describe('HomeV2.vue sign-up status line', () => {
  const upcoming = (extra = {}) => ({
    ...played('next', -3), Teams: [], ScheduledAt: new Date(Date.now() + 3 * DAY).toISOString(),
    RegistrationOpensAt: new Date(Date.now() - DAY).toISOString(), MaxPlayers: 16, ...extra,
  });

  it('offers a member the match page', async () => {
    const wrapper = await mountHome([upcoming()], false);
    const button = wrapper.find('.hero-status .hero-status-btn');
    expect(button.text()).toBe('View match');
    expect(JSON.parse(button.attributes('data-to'))).toEqual({ name: 'MatchView', params: { id: 'next' } });
    expect(wrapper.find('.hero-status .hero-meta').text()).toMatch(/^Sign-ups opened \w{3} \d{1,2} \w{3} · \d{2}:\d{2}$/);
  });

  it('offers an admin the editing page', async () => {
    const wrapper = await mountHome([upcoming()], true);
    const button = wrapper.find('.hero-status .hero-status-btn');
    expect(button.text()).toBe('Manage match');
    expect(JSON.parse(button.attributes('data-to'))).toBe('/matches/next/edit');
  });

  it('says when sign-ups will open', async () => {
    const wrapper = await mountHome([upcoming({ RegistrationOpensAt: new Date(Date.now() + DAY).toISOString() })]);
    expect(wrapper.find('.hero-status .hero-meta').text()).toMatch(/^Sign-ups open \w{3} \d{1,2} \w{3} · \d{2}:\d{2}$/);
  });
});
