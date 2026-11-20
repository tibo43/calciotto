import { mount, flushPromises } from '@vue/test-utils';
import MatchView from '@/components/v2/MatchView.vue';
import {
  getMatchRegistrations, registerForMatch, unregisterFromMatch, getMatchVotes, voteForMotm, removeMotmVote, getSeasons,
  getMatchPredictions, predictForMatch, getMatchPositions
} from '@/services/api';
import { findMatchInMyGroups } from '@/services/v2Match';

vi.mock('@/services/api', () => ({
  getMatchRegistrations: vi.fn(),
  registerForMatch: vi.fn(),
  unregisterFromMatch: vi.fn(),
  getMatchVotes: vi.fn(),
  voteForMotm: vi.fn(),
  removeMotmVote: vi.fn(),
  getSeasons: vi.fn(),
  getMatchPredictions: vi.fn(),
  predictForMatch: vi.fn(),
  getMatchPositions: vi.fn(() => Promise.resolve([])),
  // A token whose player_id is 'me'.
  getToken: vi.fn(() => `x.${btoa(JSON.stringify({ player_id: 'me' }))}.y`)
}));

vi.mock('@/services/v2Match', async (importOriginal) => ({
  ...(await importOriginal()),
  findMatchInMyGroups: vi.fn()
}));

const DAY = 24 * 60 * 60 * 1000;
const iso = (offsetMs) => new Date(Date.now() + offsetMs).toISOString();
const day = (offsetMs) => iso(offsetMs).slice(0, 10);

const GROUP = { id: 'second-group', role: 'member' };

function upcomingMatch() {
  return {
    ID: 'match', GroupID: 'second-group', Date: day(2 * DAY),
    ScheduledAt: iso(2 * DAY), RegistrationOpensAt: iso(-DAY), MaxPlayers: 16,
    Teams: [{ ID: 'a', Name: 'Black', Players: [] }, { ID: 'b', Name: 'White', Players: [] }]
  };
}

function playedMatch() {
  return {
    ID: 'match', GroupID: 'second-group', Date: day(-DAY),
    Teams: [
      { ID: 'a', Name: 'Black', Score: 2, Players: [{ ID: 'me', Name: 'Me' }, { ID: 'p2', Name: 'Paolo' }] },
      { ID: 'b', Name: 'White', Score: 1, Players: [{ ID: 'p3', Name: 'Luca' }] }
    ]
  };
}

async function mountView(match, { registrations = [], votes = null, role = 'member', positions = [] } = {}) {
  const group = { ...GROUP, role };
  getMatchPositions.mockResolvedValue(positions);
  findMatchInMyGroups.mockResolvedValue({ group, details: match, groups: [group] });
  getMatchRegistrations.mockResolvedValue(registrations);
  getMatchVotes.mockResolvedValue(votes);
  getSeasons.mockResolvedValue([]);
  const v2Context = { groups: [], seasons: [], confirming: false };
  const wrapper = mount(MatchView, {
    props: { id: 'match' },
    global: {
      provide: { v2Context },
      stubs: { 'router-link': { template: '<a :data-to="JSON.stringify(to)" v-bind="$attrs"><slot /></a>', props: ['to'], inheritAttrs: false } }
    }
  });
  await flushPromises();
  return { wrapper, v2Context };
}

const toastText = (wrapper) => wrapper.find('.toast-message span').text();

beforeEach(() => vi.clearAllMocks());

describe('MatchView.vue (design v2)', () => {
  it('shows the result once the score is recorded, even on the match day, and nothing about sign-ups', async () => {
    const recorded = {
      ...upcomingMatch(), Date: day(0), ScheduledAt: iso(-2 * 60 * 60 * 1000), RegistrationsClosedAt: iso(-3 * 60 * 60 * 1000),
      ResultRecordedAt: iso(-60 * 1000),
      Teams: playedMatch().Teams,
    };
    const { wrapper } = await mountView(recorded);
    expect(wrapper.find('.board .board-score').exists()).toBe(true);
    expect(wrapper.find('.hero').exists()).toBe(false);
    expect(wrapper.find('.motm-line').exists()).toBe(true);
    expect(wrapper.text()).not.toMatch(/Sign up|Withdraw|Sign-ups|places left/);
  });

  it('keeps every MOTM count muted while the vote is open, the one ahead included', async () => {
    const votes = { Tally: [{ PlayerID: 'p2', Name: 'Paolo', Votes: 2 }], MyVoteFor: null };
    const { wrapper } = await mountView(playedMatch(), { votes });
    expect(wrapper.text()).toContain('MOTM vote open');
    expect(wrapper.findAll('.player.is-winner')).toHaveLength(0);
  });

  it('marks the winner once the vote is closed', async () => {
    const votes = { Tally: [{ PlayerID: 'p2', Name: 'Paolo', Votes: 2 }], MyVoteFor: null };
    const { wrapper } = await mountView({ ...playedMatch(), Date: day(-5 * DAY) }, { votes });
    const winners = wrapper.findAll('.player.is-winner');
    expect(winners).toHaveLength(1);
    expect(winners[0].text()).toContain('Paolo');
  });

  it('lists the waiting list by name only', async () => {
    const registrations = [{ PlayerID: 'w1', Name: 'Gigi', Position: 17, IsWaiting: true }];
    const { wrapper } = await mountView(upcomingMatch(), { registrations });
    const row = wrapper.findAll('.signup-ro li').find((li) => li.text().includes('Gigi'));
    expect(row.text()).toBe('Gigi');
  });

  it('loads the match through the group it belongs to', async () => {
    const { wrapper } = await mountView(upcomingMatch());
    expect(findMatchInMyGroups).toHaveBeenCalledWith('match');
    expect(wrapper.find('.hero').exists()).toBe(true);
  });

  it('withdraws after the confirmation and says so', async () => {
    const mine = [{ PlayerID: 'me', Name: 'Me', Position: 1, IsWaiting: false }];
    const { wrapper, v2Context } = await mountView(upcomingMatch(), { registrations: mine });
    unregisterFromMatch.mockResolvedValue({});

    await wrapper.find('.actions .btn-ghost').trigger('click');
    expect(v2Context.confirming).toBe(true);
    await wrapper.find('.confirm-sheet .btn-danger').trigger('click');
    await flushPromises();

    expect(unregisterFromMatch).toHaveBeenCalledWith('match');
    expect(v2Context.confirming).toBe(false);
    expect(toastText(wrapper)).toBe('You have withdrawn from this match.');
  });

  it("shows the backend's reason when signing up is refused", async () => {
    const { wrapper } = await mountView(upcomingMatch());
    registerForMatch.mockRejectedValue({ response: { data: { error: 'registrations are closed' } } });

    await wrapper.find('.actions .btn').trigger('click');
    await flushPromises();

    expect(wrapper.find('.toast-message').classes()).toContain('error');
    expect(toastText(wrapper)).toBe('registrations are closed');
  });

  it('takes the vote back when tapping the player already voted for', async () => {
    const votes = { Tally: [{ PlayerID: 'p2', Votes: 1 }], MyVoteFor: 'p2' };
    const { wrapper } = await mountView(playedMatch(), { votes });
    removeMotmVote.mockResolvedValue({});

    const paolo = wrapper.findAll('button.player').find((button) => button.text().includes('Paolo'));
    await paolo.trigger('click');
    await flushPromises();

    expect(removeMotmVote).toHaveBeenCalledWith('match');
    expect(voteForMotm).not.toHaveBeenCalled();
    expect(toastText(wrapper)).toBe('Your Man of the Match vote has been removed.');
  });

  it('casts a vote on any other player', async () => {
    const { wrapper } = await mountView(playedMatch(), { votes: { Tally: [], MyVoteFor: null } });
    voteForMotm.mockResolvedValue({});

    const luca = wrapper.findAll('button.player').find((button) => button.text().includes('Luca'));
    await luca.trigger('click');
    await flushPromises();

    expect(voteForMotm).toHaveBeenCalledWith('match', 'p3');
  });
});

// Teams set before kick-off: the player's side, the line-ups and the
// prediction, all on the page the notification opens.
describe('MatchView.vue once the teams are set', () => {
  const composed = (withMe) => ({
    ...upcomingMatch(),
    RegistrationsClosedAt: iso(-DAY / 2),
    Teams: [
      { ID: 'a', Name: 'Black', Players: [{ ID: withMe ? 'me' : 'p1', Name: 'Me' }, { ID: 'p2', Name: 'Paolo' }] },
      { ID: 'b', Name: 'White', Players: [{ ID: 'p3', Name: 'Luca' }, { ID: 'p4', Name: 'Gino' }] }
    ]
  });
  const ODDS = { TeamAOdds: 1.85, DrawOdds: 3.4, TeamBOdds: 2.1 };

  it("says which team the player is on and lists both, instead of the places", async () => {
    getMatchPredictions.mockResolvedValue({ Odds: ODDS });
    const { wrapper } = await mountView(composed(true));
    expect(wrapper.find('.hero-team-line').text()).toBe('You play for Black');
    expect(wrapper.find('.hero-team-mine').text()).toBe('Black');
    expect(wrapper.find('.hero-spots').exists()).toBe(false);
    expect(wrapper.findAll('.lineups-upcoming .lineup')).toHaveLength(2);
    expect(wrapper.find('.lineups-upcoming li.is-me').text()).toContain('You');
  });

  it('only says the teams are set to a player in neither', async () => {
    getMatchPredictions.mockResolvedValue({ Odds: ODDS });
    const { wrapper } = await mountView(composed(false));
    expect(wrapper.find('.hero-team').text()).toContain('Teams are set');
    expect(wrapper.find('.hero-team-line').exists()).toBe(false);
    expect(wrapper.find('.hero-team-name').exists()).toBe(false);
  });

  it('offers the prediction next to the teams, and saves a pick', async () => {
    getMatchPredictions.mockResolvedValue({ Odds: ODDS });
    predictForMatch.mockResolvedValue({ Odds: ODDS, MyPrediction: { PredictedTeamID: null } });
    const { wrapper } = await mountView(composed(true));
    const picks = wrapper.findAll('.bet-bar .pick');
    expect(picks.map((pick) => pick.text())).toEqual(['Black1.85', 'Draw3.40', 'White2.10']);

    await picks[1].trigger('click');
    await flushPromises();
    expect(predictForMatch).toHaveBeenCalledWith('match', null);
    expect(wrapper.find('.bet-bar .pick.is-on').text()).toContain('Draw');
    expect(wrapper.find('.bet-bar .cta-msg').text()).toBe('Your prediction · Draw · 3.40');
  });

  it('has no prediction bar while the teams are not set', async () => {
    const { wrapper } = await mountView(upcomingMatch());
    expect(getMatchPredictions).not.toHaveBeenCalled();
    expect(wrapper.find('.bet-bar').exists()).toBe(false);
  });

  it('offers the waiting list once the places are taken', async () => {
    const full = Array.from({ length: 17 }, (_, index) => ({ PlayerID: `p${index}`, Name: `P${index}`, Position: index + 1, IsWaiting: index >= 16 }));
    const { wrapper } = await mountView(upcomingMatch(), { registrations: full });
    expect(wrapper.find('.actions .btn').text()).toBe('Join the waiting list');
    expect(wrapper.find('.reg-you').text()).toContain('#2');
  });
});

describe('MatchView.vue admin manage gear', () => {
  it('puts the Manage gear on the date line of an upcoming match, and hides Sign-ups closed', async () => {
    const closed = { ...upcomingMatch(), RegistrationsClosedAt: iso(-60 * 1000) };
    const { wrapper } = await mountView(closed, { role: 'admin' });
    const manage = wrapper.find('.hero-date-row .hero-manage');
    expect(manage.exists()).toBe(true);
    expect(manage.attributes('aria-label')).toBe('Manage match');
    expect(manage.attributes('title')).toBe('Manage match');
    expect(JSON.parse(manage.attributes('data-to'))).toBe('/matches/match/edit');
    expect(wrapper.find('.hero-meta').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('Sign-ups closed');
    expect(wrapper.text()).not.toContain('View match');
    expect(wrapper.findAll('.actions .hero-manage')).toHaveLength(0);
    expect(wrapper.text()).not.toMatch(/Manage match/);
  });

  it('puts the Manage gear on the date line of a finished match, not as a full-width button', async () => {
    const { wrapper } = await mountView(playedMatch(), { role: 'admin' });
    const manage = wrapper.find('.board-date-row .hero-manage');
    expect(manage.exists()).toBe(true);
    expect(manage.attributes('aria-label')).toBe('Manage match');
    expect(JSON.parse(manage.attributes('data-to'))).toBe('/matches/match/edit');
    expect(wrapper.find('.actions .hero-manage').exists()).toBe(false);
    expect(wrapper.findAll('.hero-manage')).toHaveLength(1);
  });

  it('hides the gear from a member', async () => {
    const { wrapper } = await mountView(upcomingMatch(), { role: 'member' });
    expect(wrapper.find('.hero-manage').exists()).toBe(false);
  });
});

describe('MatchView.vue preferred positions', () => {
  const positions = [
    { PlayerID: 'p2', PreferredPositions: ['MID', 'DEF'] },
    { PlayerID: 'p3', PreferredPositions: ['GK'] },
  ];

  it('shows an admin each player\'s positions after the name, in canonical order, nothing without any', async () => {
    const { wrapper } = await mountView(playedMatch(), { role: 'admin', positions });
    expect(getMatchPositions).toHaveBeenCalledWith('match');
    const rows = wrapper.findAll('.lineup .player');
    const paolo = rows.find((row) => row.text().includes('Paolo'));
    expect(paolo.find('.nm-line .nm').text()).toBe('Paolo');
    expect(paolo.find('.nm-line .pos-tag').text()).toBe('DEF · MID');
    expect(rows.find((row) => row.text().includes('Luca')).find('.pos-tag').text()).toBe('GK');
    const me = rows.find((row) => row.text().includes('Me'));
    expect(me.find('.pos-tag').exists()).toBe(false);
    expect(me.find('.nm-line').exists()).toBe(false);
  });

  it('shows the tags on the teams of an upcoming match too, for an admin', async () => {
    const match = { ...upcomingMatch(), RegistrationsClosedAt: iso(-60 * 1000), Teams: playedMatch().Teams.map((team) => ({ ...team, Score: 0 })) };
    const { wrapper } = await mountView(match, { role: 'admin', positions });
    const tags = wrapper.findAll('.lineups-upcoming .pos-tag').map((tag) => tag.text());
    expect(tags).toEqual(['DEF · MID', 'GK']);
  });

  it('never asks for positions, nor shows any, for a plain member', async () => {
    const { wrapper } = await mountView(playedMatch(), { role: 'member', positions });
    expect(getMatchPositions).not.toHaveBeenCalled();
    expect(wrapper.find('.pos-tag').exists()).toBe(false);
    expect(wrapper.find('.nm-line').exists()).toBe(false);
  });

  it('still renders the match for an admin when the positions request fails', async () => {
    getMatchPositions.mockRejectedValueOnce(new Error('boom'));
    const group = { ...GROUP, role: 'admin' };
    findMatchInMyGroups.mockResolvedValue({ group, details: playedMatch(), groups: [group] });
    getMatchRegistrations.mockResolvedValue([]);
    getMatchVotes.mockResolvedValue(null);
    getSeasons.mockResolvedValue([]);
    const wrapper = mount(MatchView, { props: { id: 'match' }, global: { provide: { v2Context: { groups: [], seasons: [] } }, stubs: { 'router-link': true } } });
    await flushPromises();
    expect(wrapper.find('.board').exists()).toBe(true);
    expect(wrapper.find('.pos-tag').exists()).toBe(false);
  });
});
