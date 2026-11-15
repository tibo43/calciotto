import { mount, flushPromises } from '@vue/test-utils';
import PredictionsV2 from '@/components/v2/PredictionsV2.vue';
import {
  getMatchesDetails, getPredictionStandings, getSeasons, getToken
} from '@/services/api';
import { resolveActiveGroup } from '@/services/activeGroup';

jest.mock('@/services/api', () => ({
  getMatchesDetails: jest.fn(),
  getMatchPredictions: jest.fn(),
  predictForMatch: jest.fn(),
  getPredictionStandings: jest.fn(),
  getSeasons: jest.fn(),
  getToken: jest.fn(() => '')
}));

jest.mock('@/services/activeGroup', () => ({
  resolveActiveGroup: jest.fn(),
  setActiveGroupId: jest.fn()
}));

function mountPredictions({ seasons = [], matches = [], route = { query: {} } } = {}) {
  resolveActiveGroup.mockResolvedValue({
    groups: [{ id: 'test-group', name: 'Test', role: 'admin' }],
    activeGroupId: 'test-group'
  });
  getSeasons.mockResolvedValue(seasons);
  getMatchesDetails.mockResolvedValue(matches);
  getPredictionStandings.mockResolvedValue([]);
  return mount(PredictionsV2, {
    global: {
      provide: { v2Context: { groups: [], seasons: [], groupId: '', season: '', hide: false } },
      mocks: { $route: route },
      stubs: { 'router-link': { template: '<a><slot /></a>', props: ['to'] } }
    }
  });
}

describe('PredictionsV2.vue empty group', () => {
  beforeEach(() => {
    getToken.mockReturnValue('');
  });

  it('shows the home-style empty state and hides the Matches/Leaderboard selector when there is no season', async () => {
    const wrapper = mountPredictions({ seasons: [], matches: [] });
    await flushPromises();
    expect(wrapper.find('.subnav').exists()).toBe(false);
    expect(wrapper.find('.state .label').text()).toBe('No matches yet');
    expect(wrapper.find('.state h1').text()).toBe('Predictions');
    expect(wrapper.findAll('.state h1')).toHaveLength(1);
    expect(wrapper.find('.state p:not(.label)').text()).toBe('Predictions open when the first match is created.');
    expect(wrapper.text()).not.toContain('No predictions yet');
    wrapper.unmount();
  });

  it('keeps No predictions yet when the group has a season but nothing is eligible', async () => {
    const wrapper = mountPredictions({
      seasons: ['2026-2027'],
      matches: [{ ID: 'm1', Date: '2026-10-01', Teams: [] }]
    });
    await flushPromises();
    expect(wrapper.find('.subnav').exists()).toBe(true);
    expect(wrapper.find('h1').text()).toBe('No predictions yet');
    expect(wrapper.text()).not.toContain('No matches yet');
    wrapper.unmount();
  });

  it('shows No matches yet on the leaderboard tab of an empty group', async () => {
    const wrapper = mountPredictions({
      seasons: [],
      matches: [],
      route: { query: { tab: 'leaderboard' } }
    });
    await flushPromises();
    expect(wrapper.find('.subnav').exists()).toBe(false);
    expect(wrapper.find('.state .label').text()).toBe('No matches yet');
    expect(wrapper.find('.state h1').text()).toBe('Predictions');
    expect(wrapper.findAll('.state h1')).toHaveLength(1);
    expect(wrapper.find('.state p:not(.label)').text()).toBe('Predictions open when the first match is created.');
    expect(wrapper.find('table').exists()).toBe(false);
    wrapper.unmount();
  });
});
