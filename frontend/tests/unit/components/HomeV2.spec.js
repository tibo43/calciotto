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
  it('keeps the last match in the header, with its score, and lists the others under Latest matches', async () => {
    const wrapper = await mountHome([played('last', 3), played('before', 10), played('oldest', 17)]);
    const hero = wrapper.find('.hero-last');
    expect(hero.find('.label').text()).toBe('Last match');
    expect(hero.find('.board-score').text()).toBe('1–0');
    expect(hero.find('.hero-status-btn').text()).toBe('View match');
    expect(hero.find('.hero-spots').exists()).toBe(false);
    expect(hero.text()).not.toMatch(/Sign up|Withdraw|Sign-ups|places/);
    const listed = wrapper.findAll('.list .row').map((row) => JSON.parse(row.attributes('data-to')).params.id);
    expect(listed).toEqual(['before', 'oldest']);
  });

  it('keeps a match whose score is just recorded in the header, Manage match for an admin', async () => {
    const recorded = {
      ...played('today', 0), ScheduledAt: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
      ResultRecordedAt: new Date(Date.now() - 60 * 1000).toISOString(), MaxPlayers: 16,
    };
    const wrapper = await mountHome([recorded, played('before', 7)], true);
    const hero = wrapper.find('.hero-last');
    expect(hero.find('.label').text()).toBe('Last match');
    expect(JSON.parse(hero.find('.hero-status-btn').attributes('data-to'))).toBe('/matches/today/edit');
    expect(wrapper.text()).not.toContain('Sign-ups');
    const listed = wrapper.findAll('.list .row').map((row) => JSON.parse(row.attributes('data-to')).params.id);
    expect(listed).toEqual(['before']);
    expect(wrapper.find('.fab').exists()).toBe(true);
  });

  it('shows the MOTM line only while the header match vote is open', async () => {
    const yesterday = { ...played('y', 1), ResultRecordedAt: new Date(Date.now() - DAY).toISOString() };
    const open = await mountHome([yesterday]);
    expect(open.find('.motm-row').text()).toBe('MOTM vote open · Vote in match details');
    expect(JSON.parse(open.find('.motm-row').attributes('data-to')).params.id).toBe('y');
    const closed = await mountHome([played('last', 3)]);
    expect(closed.find('.motm-row').exists()).toBe(false);
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
    expect((await mountHome([played('last', 3)], false)).text()).toContain('Last match');
  });
});

describe('HomeV2.vue with a match scheduled', () => {
  beforeEach(() => jest.clearAllMocks());

  const scheduled = (extra = {}) => ({
    ...played('next', -3), ScheduledAt: new Date(Date.now() + 3 * DAY).toISOString(),
    RegistrationOpensAt: new Date(Date.now() - DAY).toISOString(), MaxPlayers: 2, ...extra
  });

  it("moves the last match under Latest matches, and leaves out its MOTM vote even while it's open", async () => {
    const yesterday = { ...played('y', 1), ResultRecordedAt: new Date(Date.now() - DAY).toISOString() };
    const wrapper = await mountHome([scheduled(), yesterday]);
    expect(wrapper.find('.hero .label').text()).toBe('Next match');
    expect(wrapper.find('.hero-last').exists()).toBe(false);
    expect(JSON.parse(wrapper.find('.list .row').attributes('data-to')).params.id).toBe('y');
    expect(wrapper.find('.motm-row').exists()).toBe(false);
    expect(getMatchVotes).not.toHaveBeenCalled();
  });

  it('marks that match row « MOTM vote open » in grey, in place of its kick-off time, while its vote is open', async () => {
    const yesterday = {
      ...played('y', 1), ScheduledAt: new Date(Date.now() - DAY).toISOString(),
      ResultRecordedAt: new Date(Date.now() - DAY).toISOString(),
    };
    const before = { ...played('before', 8), ScheduledAt: new Date(Date.now() - 8 * DAY).toISOString() };
    const wrapper = await mountHome([scheduled(), yesterday, before]);
    const [row, older] = wrapper.findAll('.list .row');
    expect(JSON.parse(row.attributes('data-to')).params.id).toBe('y');
    expect(row.findAll('.row-sub')).toHaveLength(1);
    expect(row.find('.row-sub').text()).toBe('MOTM vote open');
    expect(row.find('.row-sub').classes()).toContain('row-sub-vote');
    expect(older.find('.row-sub').text()).toMatch(/^\w+ \d{2}:\d{2}$/);
    expect(older.find('.row-sub-vote').exists()).toBe(false);
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

describe('HomeV2.vue match button', () => {
  const upcoming = (extra = {}) => ({
    ...played('next', -3), Teams: [], ScheduledAt: new Date(Date.now() + 3 * DAY).toISOString(),
    RegistrationOpensAt: new Date(Date.now() - DAY).toISOString(), MaxPlayers: 16, ...extra,
  });

  it('offers a member the match page', async () => {
    const wrapper = await mountHome([upcoming()], false);
    const button = wrapper.find('.hero-status .hero-status-btn');
    expect(button.text()).toBe('View match');
    expect(JSON.parse(button.attributes('data-to'))).toEqual({ name: 'MatchView', params: { id: 'next' } });
    // No sign-up status line under the date any more: only the button.
    expect(wrapper.find('.hero-status .hero-meta').exists()).toBe(false);
  });

  it('offers an admin the editing page', async () => {
    const wrapper = await mountHome([upcoming()], true);
    const button = wrapper.find('.hero-status .hero-status-btn');
    expect(button.text()).toBe('Manage match');
    expect(JSON.parse(button.attributes('data-to'))).toBe('/matches/next/edit');
  });
});
