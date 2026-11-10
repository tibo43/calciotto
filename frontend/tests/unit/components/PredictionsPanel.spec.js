import { mount, flushPromises } from '@vue/test-utils';
import PredictionsPanel from '@/components/PredictionsPanel.vue';
import {
  getMatchesDetails,
  getMatchPredictions,
  predictForMatch,
  getPredictionStandings,
  getToken
} from '@/services/api';

jest.mock('@/services/api', () => ({
  getMatchesDetails: jest.fn(),
  getMatchPredictions: jest.fn(),
  predictForMatch: jest.fn(),
  getPredictionStandings: jest.fn(),
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
  getPredictionStandings.mockResolvedValue([]);
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
    expect(getPredictionStandings).not.toHaveBeenCalled();
  });

  it('reloads matches and standings when the season changes', async () => {
    getMatchesDetails.mockResolvedValue([scheduledComposed()]);
    const wrapper = await mountPanel();

    await wrapper.setProps({ season: '2025-2026' });
    await flushPromises();

    expect(getMatchesDetails).toHaveBeenLastCalledWith('group-uuid', '2025-2026');
    expect(getPredictionStandings).toHaveBeenLastCalledWith('2025-2026', 'group-uuid');
  });
});

describe('PredictionsPanel — the pick buttons', () => {
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

    expect(pickButtons(wrapper).every(b => b.attributes('disabled') === undefined)).toBe(true);
    expect(wrapper.find('.prediction-inline-detail').exists()).toBe(false);
  });

  it('disables every pick at kick-off and explains why', async () => {
    nowSpy.mockReturnValue(KICKOFF_MS);
    const wrapper = await mountPanel();

    expect(pickButtons(wrapper).every(b => b.attributes('disabled') !== undefined)).toBe(true);
    expect(wrapper.find('.prediction-inline-detail').text()).toBe('Predictions are closed — kick-off has passed.');
  });

  // Reading is never gated on the window, only casting.
  it('still loads and shows the summary after kick-off', async () => {
    nowSpy.mockReturnValue(KICKOFF_MS + 3600 * 1000);
    getMatchPredictions.mockResolvedValue(summary({ MyPrediction: { PredictedTeamID: 'team-a' } }));
    const wrapper = await mountPanel();

    expect(getMatchPredictions).toHaveBeenCalledWith('match-1');
    expect(pickButtons(wrapper)[0].classes('selected')).toBe(true);
  });
});
