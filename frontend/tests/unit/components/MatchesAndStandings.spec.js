import { shallowMount, flushPromises } from '@vue/test-utils';
import MatchesAndStandings from '@/components/MatchesAndStandings.vue';
import { getSeasons } from '@/services/api';
import { resolveActiveGroup, setActiveGroupId } from '@/services/activeGroup';
import { decodeMatchId } from '@/services/shortLink';

vi.mock('@/services/api', () => ({
  getSeasons: vi.fn()
}));

vi.mock('@/services/activeGroup', () => ({
  resolveActiveGroup: vi.fn(),
  setActiveGroupId: vi.fn()
}));

// decodeMatchId has its own spec (shortLink.spec.js); this file only cares
// that the page sends a decoded link to the match page.
vi.mock('@/services/shortLink', () => ({
  decodeMatchId: vi.fn()
}));

const mountPage = async (seasons, { routeQuery = {}, router = { replace: vi.fn() }, provide = {} } = {}) => {
  resolveActiveGroup.mockResolvedValue({
    groups: [{ id: 'group-uuid', name: 'Test Group', role: 'admin' }],
    activeGroupId: 'group-uuid'
  });
  getSeasons.mockResolvedValue(seasons);

  const wrapper = shallowMount(MatchesAndStandings, {
    global: { mocks: { $route: { query: routeQuery }, $router: router }, provide }
  });
  await flushPromises();
  return wrapper;
};

// The reported scenario: an admin schedules a match far enough out that it
// introduces a season nobody has played in yet. The list the page receives
// already contains that future season; the page must still open on today's.
describe('MatchesAndStandings.vue season preselection', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('defaults to the season containing today, not the last season in the list', async () => {
    vi.useFakeTimers().setSystemTime(new Date(2026, 7, 15)); // Aug 15, 2026 — still season 2025-2026
    const wrapper = await mountPage(['2024-2025', '2025-2026', '2026-2027']);
    expect(wrapper.vm.selectedSeason).toBe('2025-2026');
  });

  it('still opens on the current season in the ordinary case where it is also the last one', async () => {
    vi.useFakeTimers().setSystemTime(new Date(2026, 9, 1)); // Oct 1, 2026 — season 2026-2027
    const wrapper = await mountPage(['2025-2026', '2026-2027']);
    expect(wrapper.vm.selectedSeason).toBe('2026-2027');
  });

  it("falls back to the last season on record when today's own season has no matches yet", async () => {
    vi.useFakeTimers().setSystemTime(new Date(2026, 9, 1));
    const wrapper = await mountPage(['2024-2025', '2025-2026']);
    expect(wrapper.vm.selectedSeason).toBe('2025-2026');
  });

  it('degrades to no filtering when the group has no seasons at all', async () => {
    vi.useFakeTimers().setSystemTime(new Date(2026, 9, 1));
    const wrapper = await mountPage([]);
    expect(wrapper.vm.selectedSeason).toBe('');
  });
});

describe('MatchesAndStandings.vue shell context', () => {
  beforeEach(() => {
    sessionStorage.clear();
    setActiveGroupId.mockReset();
  });

  it('fills the shell\'s group and season selects once the page has loaded', async () => {
    const v2Context = { groups: [], seasons: [], groupId: '', season: '' };
    vi.useFakeTimers().setSystemTime(new Date(2026, 9, 1));
    await mountPage(['2026-2027'], { provide: { v2Context } });
    vi.useRealTimers();

    expect(v2Context.groupId).toBe('group-uuid');
    expect(v2Context.groups).toEqual([{ id: 'group-uuid', name: 'Test Group', role: 'admin' }]);
    expect(v2Context.seasons).toEqual(['2026-2027']);
    expect(v2Context.season).toBe('2026-2027');
  });

  it('keeps a season picked earlier in the session', async () => {
    sessionStorage.setItem('calciotto-v2-season', '2025-2026');
    const v2Context = { groups: [], seasons: [], groupId: '', season: '' };
    const wrapper = await mountPage(['2025-2026', '2026-2027'], { provide: { v2Context } });

    expect(wrapper.vm.selectedSeason).toBe('2025-2026');
    expect(v2Context.season).toBe('2025-2026');
  });
});

describe('MatchesAndStandings.vue shared match deep link (?match=<code>)', () => {
  beforeEach(() => {
    decodeMatchId.mockReset();
    resolveActiveGroup.mockReset();
  });

  it('opens a ?match= link on the match page, without resolving the group here', async () => {
    decodeMatchId.mockReturnValue('match-uuid');
    const router = { replace: vi.fn() };

    await mountPage(['2026-2027'], { routeQuery: { match: 'abc123' }, router });

    expect(router.replace).toHaveBeenCalledWith({ name: 'MatchView', params: { id: 'match-uuid' } });
    expect(resolveActiveGroup).not.toHaveBeenCalled();
  });

  it('stays on the home page when the code cannot be decoded', async () => {
    decodeMatchId.mockImplementation(() => { throw new Error('bad code'); });
    const router = { replace: vi.fn() };

    const wrapper = await mountPage(['2026-2027'], { routeQuery: { match: 'garbage' }, router });

    expect(router.replace).not.toHaveBeenCalled();
    expect(wrapper.vm.isInitializing).toBe(false);
  });

  it('does nothing special without a ?match= param', async () => {
    const router = { replace: vi.fn() };
    await mountPage(['2026-2027'], { router });
    expect(decodeMatchId).not.toHaveBeenCalled();
    expect(router.replace).not.toHaveBeenCalled();
  });
});
