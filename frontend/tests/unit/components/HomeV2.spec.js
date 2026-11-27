import { mount, flushPromises } from '@vue/test-utils';
import HomeV2 from '@/components/v2/HomeV2.vue';
import { getMatchesDetails, getMatchVotes, getMatchRegistrations, getToken } from '@/services/api';

vi.mock('@/services/api', () => ({
  getMatchesDetails: vi.fn(),
  getMatchRegistrations: vi.fn(() => Promise.resolve([])),
  getMatchVotes: vi.fn(),
  registerForMatch: vi.fn(),
  unregisterFromMatch: vi.fn(),
  getToken: vi.fn(() => '')
}));

const DAY = 24 * 60 * 60 * 1000;
const HOUR = 60 * 60 * 1000;
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
    expect(hero.find('.hero-status-btn').text()).toBe('View match');
    expect(JSON.parse(hero.find('.hero-status-btn').attributes('data-to'))).toEqual({ name: 'MatchView', params: { id: 'today' } });
    expect(hero.find('.hero-manage').attributes('aria-label')).toBe('Manage match');
    expect(hero.find('.hero-manage').attributes('title')).toBe('Manage match');
    expect(JSON.parse(hero.find('.hero-manage').attributes('data-to'))).toBe('/matches/today/edit');
    expect(wrapper.text()).not.toContain('Sign-ups');
    const listed = wrapper.findAll('.list .row').map((row) => JSON.parse(row.attributes('data-to')).params.id);
    expect(listed).toEqual(['before']);
    expect(wrapper.find('.fab').exists()).toBe(true);
  });

  it('shows the MOTM line only while the header match vote is open', async () => {
    const yesterday = { ...played('y', 1), ResultRecordedAt: new Date(Date.now() - DAY).toISOString() };
    const open = await mountHome([yesterday]);
    expect(open.find('.motm-row').text()).toBe('Star of the Match vote open · Vote in match details');
    expect(JSON.parse(open.find('.motm-row').attributes('data-to')).params.id).toBe('y');
    const closed = await mountHome([played('last', 3)]);
    expect(closed.find('.motm-row').exists()).toBe(false);
  });
});

describe('HomeV2.vue Latest matches rows', () => {
  it('shows the teams with vs in the middle and the score only on the right, read with the day', async () => {
    const before = { ...played('before', 8), Date: '2026-09-04' };
    const wrapper = await mountHome([played('last', 3), before]);
    const row = wrapper.find('.list .row');
    expect(row.find('.row-title').text().replace(/\s+/g, ' ')).toBe('Black vs White');
    expect(row.find('.row-vs').text()).toBe('vs');
    expect(row.find('.row-title').text()).not.toMatch(/\d/);
    expect(row.find('.score').text()).toBe('1–0');
    expect(row.attributes('aria-label')).toBe('Black 1–0 White, 4 Sep');
  });

  it('keeps « A vs » and « B » as two parts, the whole of a long name, never cut', async () => {
    const long = { ...played('long', 3), Teams: [
      { ID: 'a', Name: 'Association Sportive de Saint Etienne', Score: 4, Players: [{ ID: 'p' }] },
      { ID: 'b', Name: 'Paris Saint-Germain', Score: 6, Players: [{ ID: 'q' }] },
    ] };
    const wrapper = await mountHome([played('last', 1), long]);
    const row = wrapper.find('.list .row');
    const sides = row.findAll('.row-side');
    expect(sides).toHaveLength(2);
    expect(sides[0].text().replace(/\s+/g, ' ')).toBe('Association Sportive de Saint Etienne vs');
    expect(sides[0].find('.row-vs').text()).toBe('vs');
    expect(sides[1].text()).toBe('Paris Saint-Germain');
    expect(row.text()).not.toContain('…');
    expect(row.find('.score').text()).toBe('4–6');
    expect(row.attributes('aria-label')).toMatch(/^Association Sportive de Saint Etienne 4–6 Paris Saint-Germain, /);
  });

  it('describes a row by its sub-line when it has one', async () => {
    const yesterday = { ...played('y', 1), ResultRecordedAt: new Date(Date.now() - DAY).toISOString() };
    const next = { ...played('next', -3), ScheduledAt: new Date(Date.now() + 3 * DAY).toISOString(), RegistrationOpensAt: new Date(Date.now() - DAY).toISOString(), MaxPlayers: 2 };
    const wrapper = await mountHome([next, yesterday, played('old', 9)]);
    const [row, old] = wrapper.findAll('.list .row');
    expect(row.attributes('aria-describedby')).toBe('row-sub-y');
    expect(row.find('#row-sub-y').text()).toBe('Star of the Match vote open');
    expect(old.attributes('aria-describedby')).toBeUndefined();
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

describe('HomeV2.vue empty group', () => {
  it('shows the start-the-season empty state, a Create match button, and the FAB for an admin', async () => {
    const wrapper = await mountHome([], true);
    expect(wrapper.find('.label').text()).toBe('No matches yet');
    expect(wrapper.find('h1').text()).toBe('Start the season');
    expect(wrapper.text()).toContain('Create the first match to open sign-ups.');
    expect(wrapper.text()).not.toContain('An admin will create the first match.');
    const create = wrapper.findAll('a').find((link) => link.text() === 'Create match');
    expect(create).toBeTruthy();
    expect(JSON.parse(create.attributes('data-to'))).toEqual({ name: 'NewMatch' });
    expect(wrapper.find('.fab').exists()).toBe(true);
    expect(wrapper.text()).not.toContain('Latest matches');
  });

  it('tells a member an admin will create the first match, with no button and no FAB', async () => {
    const wrapper = await mountHome([], false);
    expect(wrapper.find('.label').text()).toBe('No matches yet');
    expect(wrapper.find('h1').text()).toBe('Start the season');
    expect(wrapper.text()).toContain('An admin will create the first match.');
    expect(wrapper.text()).not.toContain('Create the first match to open sign-ups.');
    expect(wrapper.findAll('a').every((link) => link.text() !== 'Create match')).toBe(true);
    expect(wrapper.find('.fab').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('Latest matches');
  });
});

describe('HomeV2.vue with a match scheduled', () => {
  beforeEach(() => vi.clearAllMocks());

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

  it('marks that match row « Star of the Match vote open » in grey, in place of its kick-off time, while its vote is open', async () => {
    // Kicked off 20 hours ago: inside the 24-hour vote window.
    const yesterday = {
      ...played('y', 1), ScheduledAt: new Date(Date.now() - 20 * HOUR).toISOString(),
      ResultRecordedAt: new Date(Date.now() - DAY).toISOString(),
    };
    const before = { ...played('before', 8), ScheduledAt: new Date(Date.now() - 8 * DAY).toISOString() };
    const wrapper = await mountHome([scheduled(), yesterday, before]);
    const [row, older] = wrapper.findAll('.list .row');
    expect(JSON.parse(row.attributes('data-to')).params.id).toBe('y');
    expect(row.findAll('.row-sub')).toHaveLength(1);
    expect(row.find('.row-sub').text()).toBe('Star of the Match vote open');
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

  it('offers a member only View match', async () => {
    const wrapper = await mountHome([upcoming()], false);
    expect(wrapper.find('.hero-status .hero-manage').exists()).toBe(false);
    expect(wrapper.findAll('.hero-status .hero-status-btn')).toHaveLength(1);
  });

  it('offers an admin View match and a Manage gear', async () => {
    const wrapper = await mountHome([upcoming()], true);
    const view = wrapper.find('.hero-status .hero-status-btn');
    expect(view.text()).toBe('View match');
    expect(JSON.parse(view.attributes('data-to'))).toEqual({ name: 'MatchView', params: { id: 'next' } });
    const manage = wrapper.find('.hero-status .hero-manage');
    expect(manage.attributes('aria-label')).toBe('Manage match');
    expect(manage.attributes('title')).toBe('Manage match');
    expect(JSON.parse(manage.attributes('data-to'))).toBe('/matches/next/edit');
    const kids = wrapper.find('.hero-status').findAll('a');
    expect(kids).toHaveLength(2);
    expect(kids.at(0).text()).toBe('View match');
    expect(kids.at(1).classes()).toContain('hero-manage');
  });
});
