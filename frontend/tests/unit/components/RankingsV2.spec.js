import { mount, flushPromises } from '@vue/test-utils';
import RankingsV2 from '@/components/v2/RankingsV2.vue';
import { getPointsStandings, getScorers, getMotmStandings, getSeasons, getToken } from '@/services/api';
import { resolveActiveGroup } from '@/services/activeGroup';

vi.mock('@/services/api', () => ({
  getPointsStandings: vi.fn(),
  getScorers: vi.fn(),
  getMotmStandings: vi.fn(),
  getSeasons: vi.fn(),
  getToken: vi.fn(() => '')
}));

vi.mock('@/services/activeGroup', () => ({
  resolveActiveGroup: vi.fn(),
  setActiveGroupId: vi.fn()
}));

function mountRankings({ seasons = [], points = [], route = { query: {} } } = {}) {
  resolveActiveGroup.mockResolvedValue({
    groups: [{ id: 'test-group', name: 'Test', role: 'admin' }],
    activeGroupId: 'test-group'
  });
  getSeasons.mockResolvedValue(seasons);
  getPointsStandings.mockResolvedValue(points);
  getScorers.mockResolvedValue([]);
  getMotmStandings.mockResolvedValue([]);
  return mount(RankingsV2, {
    global: {
      provide: { v2Context: { groups: [], seasons: [], groupId: '', season: '', hide: false } },
      mocks: { $route: route },
      stubs: { 'router-link': { template: '<a><slot /></a>', props: ['to'] } }
    }
  });
}

describe('RankingsV2.vue empty group', () => {
  beforeEach(() => {
    getToken.mockReturnValue('');
  });

  it('shows the home-style empty state and hides the stats selector when there is no season', async () => {
    const wrapper = mountRankings({ seasons: [] });
    await flushPromises();
    expect(wrapper.find('.subnav').exists()).toBe(false);
    expect(wrapper.find('table').exists()).toBe(false);
    expect(wrapper.find('.pcard').exists()).toBe(false);
    expect(wrapper.find('.state .label').text()).toBe('No matches played yet');
    expect(wrapper.find('.state h1').text()).toBe('Rankings');
    expect(wrapper.findAll('.state h1')).toHaveLength(1);
    expect(wrapper.find('.state p:not(.label)').text()).toBe('Standings appear once the first score is recorded.');
    expect(wrapper.text()).not.toContain('No standings yet');
    wrapper.unmount();
  });

  it('keeps the tab selector and the usual empty title when the group has a season but no rows', async () => {
    const wrapper = mountRankings({ seasons: ['2026-2027'] });
    await flushPromises();
    expect(wrapper.find('.subnav').exists()).toBe(true);
    expect(wrapper.find('h1').text()).toBe('No standings yet');
    expect(wrapper.text()).not.toContain('No matches played yet');
    wrapper.unmount();
  });
});

describe('RankingsV2.vue Star of the Match tab', () => {
  beforeEach(() => {
    getToken.mockReturnValue('');
  });

  it('names the motm tab Stars, and its empty title No stars yet', async () => {
    const wrapper = mountRankings({ seasons: ['2026-2027'], route: { query: { tab: 'motm' } } });
    await flushPromises();
    expect(wrapper.findAll('.subnav a').map((tab) => tab.text())).toEqual(['Points', 'Scorers', 'Stars']);
    expect(wrapper.find('h1').text()).toBe('No stars yet');
    wrapper.unmount();
  });
});
