import { mount, flushPromises } from '@vue/test-utils';
import MatchView from '@/components/v2/MatchView.vue';
import {
  getMatchRegistrations, registerForMatch, unregisterFromMatch, getMatchVotes, voteForMotm, removeMotmVote, getSeasons
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
