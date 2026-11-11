import { shallowMount, flushPromises } from '@vue/test-utils';
import Predictions from '@/components/Predictions.vue';
import PredictionsPanel from '@/components/PredictionsPanel.vue';
import PredictionsStandingsTable from '@/components/PredictionsStandingsTable.vue';
import { getSeasons, getPredictionStandings } from '@/services/api';
import { resolveActiveGroup } from '@/services/activeGroup';
import { seasonOf } from '@/services/datetime';

jest.mock('@/services/api', () => ({
  getSeasons: jest.fn(),
  getPredictionStandings: jest.fn()
}));

jest.mock('@/services/activeGroup', () => ({
  resolveActiveGroup: jest.fn(),
  setActiveGroupId: jest.fn()
}));

const CURRENT = seasonOf(new Date());
const ROWS = [{ PlayerID: 'p1', Name: 'alice', Predictions: 2, Correct: 1, Points: 0.5, IsMember: true }];

const mountPage = async () => {
  const wrapper = shallowMount(Predictions);
  await flushPromises();
  return wrapper;
};

const tabs = (wrapper) => wrapper.findAll('.sub-tab-button');

beforeEach(() => {
  jest.clearAllMocks();
  resolveActiveGroup.mockResolvedValue({
    groups: [{ id: 'group-uuid', name: 'Test Group', role: 'member' }],
    activeGroupId: 'group-uuid'
  });
  getSeasons.mockResolvedValue(['2025-2026', CURRENT]);
  getPredictionStandings.mockResolvedValue(ROWS);
});

describe('Predictions page', () => {
  it('has a Matches and a Leaderboard tab, opening on Matches', async () => {
    const wrapper = await mountPage();

    expect(tabs(wrapper).map(t => t.text())).toEqual(['Matches', 'Leaderboard']);
    expect(tabs(wrapper)[0].classes('active')).toBe(true);
    expect(wrapper.findComponent(PredictionsPanel).props()).toEqual({ activeGroupId: 'group-uuid', season: CURRENT });
    expect(wrapper.findComponent(PredictionsStandingsTable).exists()).toBe(false);
  });

  it('shows the season leaderboard on its own tab', async () => {
    const wrapper = await mountPage();

    await tabs(wrapper)[1].trigger('click');

    expect(wrapper.findComponent(PredictionsPanel).exists()).toBe(false);
    expect(wrapper.findComponent(PredictionsStandingsTable).props('rows')).toEqual(ROWS);
    expect(getPredictionStandings).toHaveBeenCalledWith(CURRENT, 'group-uuid');
  });

  it('reloads the leaderboard when the season changes', async () => {
    const wrapper = await mountPage();

    await wrapper.find('#season-select').setValue('2025-2026');
    await flushPromises();

    expect(getPredictionStandings).toHaveBeenLastCalledWith('2025-2026', 'group-uuid');
  });
});
