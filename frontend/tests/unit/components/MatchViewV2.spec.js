import { mount, flushPromises } from '@vue/test-utils';
import MatchView from '@/components/v2/MatchView.vue';
import {
  getMatchRegistrations, registerForMatch, unregisterFromMatch, getMatchVotes, voteForMotm, removeMotmVote, getSeasons,
  getMatchPredictions, predictForMatch
} from '@/services/api';
import { findMatchInMyGroups } from '@/services/v2Match';

jest.mock('@/services/api', () => ({
  getMatchRegistrations: jest.fn(),
  registerForMatch: jest.fn(),
  unregisterFromMatch: jest.fn(),
  getMatchVotes: jest.fn(),
  voteForMotm: jest.fn(),
  removeMotmVote: jest.fn(),
  getSeasons: jest.fn(),
  getMatchPredictions: jest.fn(),
  predictForMatch: jest.fn(),
  // A token whose player_id is 'me'.
  getToken: jest.fn(() => `x.${btoa(JSON.stringify({ player_id: 'me' }))}.y`)
}));

jest.mock('@/services/v2Match', () => ({
  ...jest.requireActual('@/services/v2Match'),
  findMatchInMyGroups: jest.fn()
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

async function mountView(match, { registrations = [], votes = null } = {}) {
  findMatchInMyGroups.mockResolvedValue({ group: GROUP, details: match, groups: [GROUP] });
  getMatchRegistrations.mockResolvedValue(registrations);
  getMatchVotes.mockResolvedValue(votes);
  getSeasons.mockResolvedValue([]);
  const v2Context = { groups: [], seasons: [], confirming: false };
  const wrapper = mount(MatchView, {
    props: { id: 'match' },
    global: { provide: { v2Context }, stubs: { 'router-link': true } }
  });
  await flushPromises();
  return { wrapper, v2Context };
}

const toastText = (wrapper) => wrapper.find('.toast-message span').text();

beforeEach(() => jest.clearAllMocks());

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
    expect(wrapper.find('.hero-team').text()).toContain('You play for');
    expect(wrapper.find('.hero-team-name').text()).toBe('Black');
    expect(wrapper.find('.hero-spots').exists()).toBe(false);
    expect(wrapper.findAll('.lineups-upcoming .lineup')).toHaveLength(2);
    expect(wrapper.find('.lineups-upcoming li.is-me').text()).toContain('You');
  });

  it('only says the teams are set to a player in neither', async () => {
    getMatchPredictions.mockResolvedValue({ Odds: ODDS });
    const { wrapper } = await mountView(composed(false));
    expect(wrapper.find('.hero-team').text()).toContain('Teams are set');
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
