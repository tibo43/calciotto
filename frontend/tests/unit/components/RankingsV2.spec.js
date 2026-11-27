import { mount, flushPromises } from '@vue/test-utils';
import RankingsV2 from '@/components/v2/RankingsV2.vue';
import { getPointsStandings, getScorers, getMotmStandings, getAgoranking, getSeasons, getToken } from '@/services/api';
import { resolveActiveGroup } from '@/services/activeGroup';

vi.mock('@/services/api', () => ({
  getPointsStandings: vi.fn(),
  getScorers: vi.fn(),
  getMotmStandings: vi.fn(),
  getAgoranking: vi.fn(),
  getSeasons: vi.fn(),
  getToken: vi.fn(() => '')
}));

vi.mock('@/services/activeGroup', () => ({
  resolveActiveGroup: vi.fn(),
  setActiveGroupId: vi.fn()
}));

function mountRankings({ seasons = [], points = [], agoranking = null, agorankingError = null, route = { query: {} } } = {}) {
  resolveActiveGroup.mockResolvedValue({
    groups: [{ id: 'test-group', name: 'Test', role: 'admin' }],
    activeGroupId: 'test-group'
  });
  getSeasons.mockResolvedValue(seasons);
  getPointsStandings.mockResolvedValue(points);
  getScorers.mockResolvedValue([]);
  getMotmStandings.mockResolvedValue([]);
  if (agorankingError) getAgoranking.mockRejectedValue(agorankingError);
  else getAgoranking.mockResolvedValue(agoranking ?? { Matches: 0, N: 0, GroupAverage: 0, Ranked: [], Unranked: [] });
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
    expect(wrapper.findAll('.subnav a').map((tab) => tab.text())).toEqual(['Points', 'Scorers', 'Stars', 'Agoranking']);
    expect(wrapper.find('h1').text()).toBe('No stars yet');
    wrapper.unmount();
  });
});

// v-show hides with an inline display: none.
function shown(wrapper, selector) {
  return !(wrapper.find(selector).attributes('style') || '').includes('display: none');
}

// A token whose payload carries player_id "me".
const ME_TOKEN = `x.${btoa(JSON.stringify({ player_id: 'me' }))}.y`;

const AGORANKING = {
  Matches: 11,
  N: 4,
  GroupAverage: 1.318,
  Ranked: [
    { PlayerID: 'p1', Name: 'mathias', Played: 4, Won: 3, Drawn: 1, Lost: 0, Points: 10, Rating: 1.9147, IsMember: true },
    { PlayerID: 'me', Name: 'thibaut fabre', Played: 8, Won: 4, Drawn: 2, Lost: 2, Points: 14, Rating: 1.6059, IsMember: true },
    { PlayerID: 'p3', Name: 'old timer', Played: 5, Won: 1, Drawn: 0, Lost: 4, Points: 3, Rating: 0.92, IsMember: false },
  ],
  Unranked: [
    { PlayerID: 'u1', Name: 'nick', Played: 2, Won: 2, Drawn: 0, Lost: 0, Points: 6, Rating: 1.88, IsMember: true },
    { PlayerID: 'u2', Name: 'brent', Played: 1, Won: 0, Drawn: 1, Lost: 0, Points: 1, Rating: 1.25, IsMember: true },
  ],
};

describe('RankingsV2.vue Agoranking tab', () => {
  beforeEach(() => {
    getToken.mockReturnValue(ME_TOKEN);
  });

  async function mountAgoranking(options = {}) {
    const wrapper = mountRankings({ seasons: ['2026-2027'], agoranking: AGORANKING, route: { query: { tab: 'agoranking' } }, ...options });
    await flushPromises();
    return wrapper;
  }

  it('asks the backend for the selected season', async () => {
    const wrapper = await mountAgoranking();
    expect(getAgoranking).toHaveBeenCalledWith('2026-2027', 'test-group');
    wrapper.unmount();
  });

  it('shows the title, the lead sentence and the ranked table in order with two-decimal ratings', async () => {
    const wrapper = await mountAgoranking();
    expect(wrapper.find('.subnav a:last-child').text()).toBe('Agoranking');
    expect(wrapper.find('.agr-title').text()).toBe('Agoranking');
    expect(wrapper.find('.agr-lead').text()).toBe("Who wins the most points per match, not in total. A few lucky games aren't enough: the more you play, the more your own results count.");
    expect(wrapper.find('.agr-lead b').text()).toBe('per match');
    const headers = wrapper.findAll('table.agr-table:not(.agr-unranked-table) th').map((th) => th.text());
    expect(headers).toEqual(['#', 'Player', 'M', 'W-D-L', 'Pts', 'Rating']);
    const rows = wrapper.findAll('table.agr-table:not(.agr-unranked-table) tbody tr');
    expect(rows.map((row) => row.findAll('td').map((td) => td.text()))).toEqual([
      ['1', 'Mathias', '4', '3-1-0', '10', '1.91'],
      ['2', 'Thibaut FabreYOU', '8', '4-2-2', '14', '1.61'],
      ['3', 'Old Timer (left the group)', '5', '1-0-4', '3', '0.92'],
    ]);
    wrapper.unmount();
  });

  it('highlights the current player with YOU, in the table and on the mobile cards', async () => {
    const wrapper = await mountAgoranking();
    const me = wrapper.findAll('table.agr-table tbody tr.is-me');
    expect(me).toHaveLength(1);
    expect(me[0].find('.agr-you').text()).toBe('YOU');
    const card = wrapper.find('.pcard.agr-card.is-me');
    expect(card.find('.nm').text()).toBe('Thibaut FabreYOU');
    expect(card.find('.st').text()).toBe('8 M · 4W 2D 2L · 14 pts');
    expect(card.find('.pts').text()).toBe('1.61');
    wrapper.unmount();
  });

  it('keeps "How it\'s calculated" closed by default, with no fixed number inside', async () => {
    const wrapper = await mountAgoranking();
    const head = wrapper.find('.agr-how .agr-fold-head');
    expect(head.text()).toBe("How it's calculated");
    expect(head.attributes('aria-expanded')).toBe('false');
    expect(shown(wrapper, '#agr-how-body')).toBe(false);
    await head.trigger('click');
    expect(head.attributes('aria-expanded')).toBe('true');
    expect(shown(wrapper, '#agr-how-body')).toBe(true);
    const body = wrapper.find('#agr-how-body');
    const paragraphs = body.findAll('p').map((p) => p.text());
    expect(paragraphs[0]).toBe("A win is worth 3 points, a draw 1, a loss 0. Every player starts with a few “average” matches worth the group's average points per match. Your real matches are added on top, so a newcomer sits near the average and moves away from it as they play.");
    expect(body.find('.agr-formula').text()).toBe('rating = (points + N × group avg) ÷ (matches + N)');
    expect(paragraphs[1]).toBe("N is a third of the matches the group has played this season, rounded up. It's also the minimum number of matches you need to be ranked, so both grow as the season goes on.");
    // Nothing season-specific: neither the threshold, nor the average, nor the match count.
    expect(body.text()).not.toMatch(/1\.3|\b11\b|\b4 matches/);
    wrapper.unmount();
  });

  it('folds the unranked players under "Not ranked yet · count", in grey, with their match count', async () => {
    const wrapper = await mountAgoranking();
    const head = wrapper.find('.agr-unranked .agr-fold-head');
    expect(head.text()).toBe('Not ranked yet · 2');
    expect(head.attributes('aria-expanded')).toBe('false');
    expect(shown(wrapper, '#agr-unranked-body')).toBe(false);
    await head.trigger('click');
    expect(shown(wrapper, '#agr-unranked-body')).toBe(true);
    const rows = wrapper.findAll('table.agr-unranked-table tbody tr');
    expect(rows.map((row) => row.findAll('td').map((td) => td.text()))).toEqual([
      ['–', 'Nick', '2', '2-0-0', '6', '–'],
      ['–', 'Brent', '1', '0-1-0', '1', '–'],
    ]);
    const cards = wrapper.findAll('.agr-ucard');
    expect(cards.map((card) => [card.find('.nm').text(), card.find('.st').text(), card.find('.m').text()])).toEqual([
      ['Nick', '2W 0D 0L · 6 pts', '2 M'],
      ['Brent', '0W 1D 0L · 1 pt', '1 M'],
    ]);
    wrapper.unmount();
  });

  it('hides the unranked fold when everyone is ranked', async () => {
    const wrapper = await mountAgoranking({ agoranking: { ...AGORANKING, Unranked: [] } });
    expect(wrapper.find('.agr-unranked').exists()).toBe(false);
    wrapper.unmount();
  });

  it('shows the usual empty title when the season has no ranking', async () => {
    const wrapper = await mountAgoranking({ agoranking: null });
    expect(wrapper.find('h1').text()).toBe('No standings yet');
    wrapper.unmount();
  });

  it('only breaks its own tab when the Agoranking fails to load', async () => {
    const failing = await mountAgoranking({ agorankingError: new Error('boom') });
    expect(failing.find('.state h1').text()).toBe("Couldn't load the Agoranking");
    failing.unmount();

    const points = mountRankings({ seasons: ['2026-2027'], points: [{ PlayerID: 'p1', Name: 'a', Played: 1, Points: 3, IsMember: true }], agorankingError: new Error('boom') });
    await flushPromises();
    expect(points.find('table.stand').exists()).toBe(true);
    expect(points.text()).not.toContain("Couldn't load");
    points.unmount();
  });
});
