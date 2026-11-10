import { mount, flushPromises } from '@vue/test-utils';
import PredictionsPanel from '@/components/PredictionsPanel.vue';
import {
  getMatchesDetails,
  getMatchPredictions,
  predictForMatch,
  getToken
} from '@/services/api';

jest.mock('@/services/api', () => ({
  getMatchesDetails: jest.fn(),
  getMatchPredictions: jest.fn(),
  predictForMatch: jest.fn(),
  getToken: jest.fn()
}));

const ME = '11111111-1111-1111-1111-111111111111';
const KICKOFF = '2026-09-06T21:00:00+02:00';
const KICKOFF_MS = Date.parse(KICKOFF);

const tokenFor = (playerId) => {
  const payload = Buffer.from(JSON.stringify({ player_id: playerId })).toString('base64');
  return `header.${payload}.signature`;
};

const team = (id, name, players) => ({
  ID: id,
  Name: name,
  Colour: 'black',
  Players: players.map(p => ({ ID: p, Name: p }))
});

const scheduledComposed = (overrides = {}) => ({
  ID: 'match-1',
  Date: '2026-09-06',
  ScheduledAt: KICKOFF,
  Teams: [team('team-a', 'Black', ['alice']), team('team-b', 'White', ['bob'])],
  ...overrides
});

const summary = (overrides = {}) => ({
  Entries: [],
  Odds: { TeamAOdds: 1.8, DrawOdds: 3.5, TeamBOdds: 2.4 },
  MyPrediction: null,
  ...overrides
});

const pickButtons = (wrapper) => wrapper.findAll('.prediction-pick-btn');

const mountPanel = async (propsOverrides = {}) => {
  const wrapper = mount(PredictionsPanel, {
    props: { activeGroupId: 'group-uuid', season: '', ...propsOverrides }
  });
  await flushPromises();
  return wrapper;
};

let nowSpy;

beforeEach(() => {
  jest.clearAllMocks();
  getToken.mockReturnValue(tokenFor(ME));
  getMatchPredictions.mockResolvedValue(summary());
  // Two days before kick-off, so the window is open unless a test says so.
  nowSpy = jest.spyOn(Date, 'now').mockReturnValue(KICKOFF_MS - 2 * 24 * 3600 * 1000);
  jest.spyOn(console, 'error').mockImplementation(() => {});
});

afterEach(() => {
  nowSpy.mockRestore();
  console.error.mockRestore();
});

describe('PredictionsPanel — which matches are offered', () => {
  it('lists only scheduled matches whose roster is fully composed', async () => {
    getMatchesDetails.mockResolvedValue([
      scheduledComposed(),
      // Unscheduled — deliberately never predictable, even when composed.
      scheduledComposed({ ID: 'match-unscheduled', ScheduledAt: undefined }),
      // Scheduled, but one side still empty.
      scheduledComposed({ ID: 'match-half', Teams: [team('team-a', 'Black', ['alice']), team('team-b', 'White', [])] })
    ]);

    const wrapper = await mountPanel();

    expect(wrapper.vm.eligibleMatches.map(m => m.ID)).toEqual(['match-1']);
  });

  it('auto-selects the first eligible match and loads its predictions', async () => {
    getMatchesDetails.mockResolvedValue([scheduledComposed()]);

    const wrapper = await mountPanel();

    expect(wrapper.vm.selectedMatchId).toBe('match-1');
    expect(getMatchPredictions).toHaveBeenCalledWith('match-1');
    expect(wrapper.find('.match-details-container').exists()).toBe(true);
  });

  it('fetches nothing when no group is resolved', async () => {
    await mountPanel({ activeGroupId: '' });

    expect(getMatchesDetails).not.toHaveBeenCalled();
  });

  it('reloads matches when the season changes', async () => {
    getMatchesDetails.mockResolvedValue([scheduledComposed()]);
    const wrapper = await mountPanel();

    await wrapper.setProps({ season: '2025-2026' });
    await flushPromises();

    expect(getMatchesDetails).toHaveBeenLastCalledWith('group-uuid', '2025-2026');
  });
});

describe('PredictionsPanel — the pick buttons', () => {
  it('puts each match\'s own odds and pick on its own card', async () => {
    getMatchesDetails.mockResolvedValue([scheduledComposed(), scheduledComposed({ ID: 'match-2' })]);
    getMatchPredictions.mockImplementation(async (id) => (id === 'match-2'
      ? summary({ Odds: { TeamAOdds: 3, DrawOdds: 3.2, TeamBOdds: 2 }, MyPrediction: { PredictedTeamID: 'team-b' } })
      : summary()));
    const wrapper = await mountPanel();
    const [first, second] = wrapper.findAll('.match-card-horizontal');

    expect(first.findAll('.pick-odds').map(n => n.text())).toEqual(['1.80x', '3.50x', '2.40x']);
    expect(second.findAll('.pick-odds').map(n => n.text())).toEqual(['3.00x', '3.20x', '2.00x']);
    expect(first.findAll('.prediction-pick-btn').map(b => b.classes('selected'))).toEqual([false, false, false]);
    expect(second.findAll('.prediction-pick-btn').map(b => b.classes('selected'))).toEqual([false, false, true]);
  });

  it('places a bet from a card that isn\'t selected, and selects it', async () => {
    getMatchesDetails.mockResolvedValue([scheduledComposed(), scheduledComposed({ ID: 'match-2' })]);
    predictForMatch.mockResolvedValue(summary({ MyPrediction: { PredictedTeamID: null } }));
    const wrapper = await mountPanel();

    await wrapper.findAll('.match-card-horizontal')[1].findAll('.prediction-pick-btn')[1].trigger('click');
    await flushPromises();

    expect(predictForMatch).toHaveBeenCalledWith('match-2', null);
    expect(wrapper.vm.selectedMatchId).toBe('match-2');
    expect(wrapper.findAll('.match-card-horizontal')[1].findAll('.prediction-pick-btn')[1].classes('selected')).toBe(true);
  });

  beforeEach(() => {
    getMatchesDetails.mockResolvedValue([scheduledComposed()]);
  });

  it('renders team A / Draw / team B with their odds', async () => {
    const wrapper = await mountPanel();
    const buttons = pickButtons(wrapper);

    expect(buttons.map(b => b.find('.pick-label').text())).toEqual(['Black', 'Draw', 'White']);
    expect(buttons.map(b => b.find('.pick-odds').text())).toEqual(['1.80x', '3.50x', '2.40x']);
  });

  it('hides the odds while the backend has none to give', async () => {
    getMatchPredictions.mockResolvedValue(summary({ Odds: null }));
    const wrapper = await mountPanel();

    expect(wrapper.findAll('.pick-odds')).toHaveLength(0);
  });

  it('highlights nothing before the caller has predicted', async () => {
    const wrapper = await mountPanel();

    expect(pickButtons(wrapper).filter(b => b.classes('selected'))).toHaveLength(0);
  });

  // A null PredictedTeamID inside a present MyPrediction is a draw, not
  // "no prediction".
  it('highlights Draw when the caller predicted a draw', async () => {
    getMatchPredictions.mockResolvedValue(summary({ MyPrediction: { PredictedTeamID: null } }));
    const wrapper = await mountPanel();

    expect(pickButtons(wrapper).map(b => b.classes('selected'))).toEqual([false, true, false]);
  });

  it('highlights the backed team', async () => {
    getMatchPredictions.mockResolvedValue(summary({ MyPrediction: { PredictedTeamID: 'team-b' } }));
    const wrapper = await mountPanel();

    expect(pickButtons(wrapper).map(b => b.classes('selected'))).toEqual([false, false, true]);
  });

  it('sends the team id, or null for Draw, and adopts the returned summary', async () => {
    predictForMatch.mockResolvedValue(summary({ MyPrediction: { PredictedTeamID: null } }));
    const wrapper = await mountPanel();

    await pickButtons(wrapper)[1].trigger('click');
    await flushPromises();

    expect(predictForMatch).toHaveBeenCalledWith('match-1', null);
    expect(pickButtons(wrapper)[1].classes('selected')).toBe(true);

    predictForMatch.mockResolvedValue(summary({ MyPrediction: { PredictedTeamID: 'team-a' } }));
    await pickButtons(wrapper)[0].trigger('click');
    await flushPromises();

    expect(predictForMatch).toHaveBeenLastCalledWith('match-1', 'team-a');
    expect(pickButtons(wrapper).map(b => b.classes('selected'))).toEqual([true, false, false]);
  });

  it('surfaces the backend error and reloads the summary on a rejection', async () => {
    predictForMatch.mockRejectedValue({ response: { status: 409, data: { error: 'predictions are closed' } } });
    const wrapper = await mountPanel();
    getMatchPredictions.mockClear();

    await pickButtons(wrapper)[0].trigger('click');
    await flushPromises();

    expect(wrapper.find('.prediction-inline-message.error').text()).toBe('predictions are closed');
    expect(getMatchPredictions).toHaveBeenCalledWith('match-1');
  });
});

describe('PredictionsPanel — the kick-off window', () => {
  beforeEach(() => {
    getMatchesDetails.mockResolvedValue([scheduledComposed()]);
  });

  it('enables every pick before kick-off, with no closed reason', async () => {
    const wrapper = await mountPanel();

    expect(pickButtons(wrapper).every(b => !b.classes('locked') && b.attributes('aria-disabled') === 'false')).toBe(true);
    expect(wrapper.find('.match-card-horizontal .prediction-state-badge').text()).toBe('Predictions open');
  });

  it('disables every pick at kick-off and says the result is awaited', async () => {
    nowSpy.mockReturnValue(KICKOFF_MS);
    const wrapper = await mountPanel();

    expect(pickButtons(wrapper).every(b => b.classes('locked') && b.attributes('aria-disabled') === 'true')).toBe(true);
    expect(wrapper.find('.match-card-horizontal .prediction-state-badge').text()).toBe('Awaiting result');
    // Nothing is settled yet, so no row carries points.
    expect(wrapper.findAll('.prediction-entry-points')).toHaveLength(0);
  });

  // Reading is never gated on the window, only casting.
  // A locked row is not a disabled <button>: tapping it must still select
  // the match, it just doesn't place a bet.
  it('selects a closed match from its rows without predicting', async () => {
    nowSpy.mockReturnValue(KICKOFF_MS);
    getMatchesDetails.mockResolvedValue([scheduledComposed(), scheduledComposed({ ID: 'match-2' })]);
    const wrapper = await mountPanel();

    await wrapper.findAll('.match-card-horizontal')[1].findAll('.prediction-pick-btn')[0].trigger('click');
    await flushPromises();

    expect(wrapper.vm.selectedMatchId).toBe('match-2');
    expect(predictForMatch).not.toHaveBeenCalled();
  });

  it('still loads and shows the summary after kick-off', async () => {
    nowSpy.mockReturnValue(KICKOFF_MS + 3600 * 1000);
    getMatchPredictions.mockResolvedValue(summary({ MyPrediction: { PredictedTeamID: 'team-a' } }));
    const wrapper = await mountPanel();

    expect(getMatchPredictions).toHaveBeenCalledWith('match-1');
    expect(pickButtons(wrapper)[0].classes('selected')).toBe(true);
  });
});

describe('PredictionsPanel — others\' picks before kick-off', () => {
  beforeEach(() => {
    getMatchesDetails.mockResolvedValue([scheduledComposed()]);
  });

  it('shows only the caller\'s own pick, the total count and a hint while picks are hidden', async () => {
    getMatchPredictions.mockResolvedValue(summary({
      Entries: [{ PlayerID: ME, Name: 'me', PredictedTeamID: 'team-a' }],
      Count: 5,
      Revealed: false,
      MyPrediction: { PlayerID: ME, Name: 'me', PredictedTeamID: 'team-a' }
    }));
    const wrapper = await mountPanel();

    expect(wrapper.find('.count-badge').text()).toBe('5');
    expect(wrapper.findAll('.signup-entry')).toHaveLength(1);
    expect(wrapper.find('.prediction-hidden-hint').text()).toBe('Everyone\'s picks are revealed at kick-off.');
  });

  it('keeps the section, with no list, when others predicted but the caller hasn\'t', async () => {
    getMatchPredictions.mockResolvedValue(summary({ Entries: [], Count: 3, Revealed: false }));
    const wrapper = await mountPanel();

    expect(wrapper.find('.count-badge').text()).toBe('3');
    expect(wrapper.find('.signup-entries').exists()).toBe(false);
    expect(wrapper.find('.prediction-hidden-hint').exists()).toBe(true);
  });

  it('drops the hint once picks are revealed', async () => {
    getMatchPredictions.mockResolvedValue(summary({
      Entries: [{ PlayerID: 'p2', Name: 'zoe', PredictedTeamID: null }],
      Count: 1,
      Revealed: true
    }));
    const wrapper = await mountPanel();

    expect(wrapper.find('.prediction-hidden-hint').exists()).toBe(false);
    expect(wrapper.findAll('.signup-entry')).toHaveLength(1);
  });
});

describe('PredictionsPanel — a settled match', () => {
  const settledMatch = () => {
    const match = scheduledComposed({ ResultRecordedAt: '2026-09-06T23:10:00+02:00' });
    match.Teams[0].Score = 3;
    match.Teams[1].Score = 1;
    return match;
  };

  beforeEach(() => {
    nowSpy.mockReturnValue(KICKOFF_MS + 24 * 3600 * 1000);
    getMatchesDetails.mockResolvedValue([settledMatch()]);
    getMatchPredictions.mockResolvedValue(summary({
      Entries: [
        { PlayerID: ME, Name: 'me', PredictedTeamID: 'team-a' },
        { PlayerID: 'p2', Name: 'zoe', PredictedTeamID: null }
      ],
      Count: 2,
      Revealed: true,
      MyPrediction: { PlayerID: ME, Name: 'me', PredictedTeamID: 'team-a' }
    }));
  });

  it('shows the final score and the settled badge on the card', async () => {
    const wrapper = await mountPanel();

    expect(wrapper.find('.match-card-horizontal .prediction-state-badge').text()).toBe('Settled');
    expect(wrapper.find('.match-card-horizontal .card-score').text()).toBe('3 – 1');
  });

  it('marks the winning outcome and colours the caller\'s own pick by its result', async () => {
    const wrapper = await mountPanel();
    const buttons = pickButtons(wrapper);

    expect(buttons.map(b => b.classes('winner'))).toEqual([true, false, false]);
    expect(buttons[0].find('.pick-winner-icon').exists()).toBe(true);
    expect(buttons[0].classes('won')).toBe(true);
  });

  it('shows a losing pick in red and says so', async () => {
    getMatchPredictions.mockResolvedValue(summary({
      Entries: [{ PlayerID: ME, Name: 'me', PredictedTeamID: null }],
      Count: 1,
      Revealed: true,
      MyPrediction: { PlayerID: ME, Name: 'me', PredictedTeamID: null }
    }));
    const wrapper = await mountPanel();
    const buttons = pickButtons(wrapper);

    expect(buttons[1].classes('lost')).toBe(true);
    expect(buttons[0].classes('winner')).toBe(true);
  });

  it('colours nothing as won or lost when the caller made no prediction', async () => {
    getMatchPredictions.mockResolvedValue(summary({ Entries: [], Count: 0, Revealed: true }));
    const wrapper = await mountPanel();

    expect(pickButtons(wrapper).some(b => b.classes('won') || b.classes('lost'))).toBe(false);
    expect(pickButtons(wrapper)[0].classes('winner')).toBe(true);
  });

  it('shows each prediction\'s points: odds minus one on a hit, minus one on a miss', async () => {
    const wrapper = await mountPanel();
    const points = wrapper.findAll('.prediction-entry-points');

    expect(points.map(n => n.text())).toEqual(['+0.80', '-1.00']);
    expect(points.map(n => n.classes('won'))).toEqual([true, false]);
  });
});

describe('PredictionsPanel — collapsible sections', () => {
  beforeEach(() => {
    getMatchesDetails.mockResolvedValue([scheduledComposed(), scheduledComposed({ ID: 'match-2' })]);
    getMatchPredictions.mockResolvedValue(summary({ Entries: [{ PlayerID: 'p2', Name: 'zoe', PredictedTeamID: null }] }));
  });

  const toggle = (wrapper, label) => wrapper.findAll('.signup-toggle-btn').find(b => b.text().includes(label));

  it('starts with the teams collapsed and the predictions shown', async () => {
    const wrapper = await mountPanel();

    expect(wrapper.find('.teams-columns').exists()).toBe(false);
    expect(wrapper.find('.signup-entries').exists()).toBe(true);
  });

  it('expands and collapses each section independently', async () => {
    const wrapper = await mountPanel();

    await toggle(wrapper, 'Show teams').trigger('click');
    expect(wrapper.find('.teams-columns').exists()).toBe(true);

    await toggle(wrapper, 'Hide predictions').trigger('click');
    expect(wrapper.find('.signup-entries').exists()).toBe(false);
    expect(wrapper.find('.teams-columns').exists()).toBe(true);
  });

  it('resets to the default when another match is selected', async () => {
    const wrapper = await mountPanel();
    await toggle(wrapper, 'Show teams').trigger('click');

    await wrapper.findAll('.match-card-horizontal')[1].trigger('click');
    await flushPromises();

    expect(wrapper.find('.teams-columns').exists()).toBe(false);
  });
});
