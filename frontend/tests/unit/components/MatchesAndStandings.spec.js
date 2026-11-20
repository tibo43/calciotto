import { ref } from 'vue';
import { shallowMount, flushPromises } from '@vue/test-utils';
import MatchesAndStandings from '@/components/MatchesAndStandings.vue';
import { getPointsStandings, getScorers, getMotmStandings, getSeasons } from '@/services/api';
import { resolveActiveGroup, setActiveGroupId } from '@/services/activeGroup';
import { decodeMatchId } from '@/services/shortLink';
import { findGroupForMatch } from '@/router/index';
import { loadDesignV2 } from '@/services/designV2';

vi.mock('@/services/api', () => ({
  getPointsStandings: vi.fn(),
  getScorers: vi.fn(),
  getMotmStandings: vi.fn(),
  getSeasons: vi.fn()
}));

vi.mock('@/services/activeGroup', () => ({
  resolveActiveGroup: vi.fn(),
  setActiveGroupId: vi.fn()
}));

// Both mocked wholesale for the same reason: real findGroupForMatch would
// hit the real getMatchDetailsByID (not stubbed here), and real
// decodeMatchId has its own dedicated spec (shortLink.spec.js) already
// covering the base62 round-trip — this file only cares that the page wires
// their results into the right tab/group/season/prop.
vi.mock('@/services/shortLink', () => ({
  decodeMatchId: vi.fn()
}));

vi.mock('@/router/index', () => ({
  findGroupForMatch: vi.fn()
}));

// Design v2 off unless a test says otherwise: these tests are about today's
// page.
vi.mock('@/services/designV2', () => ({
  loadDesignV2: vi.fn(() => Promise.resolve(false))
}));

const mountPage = async (seasons, { routeQuery = {}, router = { replace: vi.fn() }, provide = {} } = {}) => {
  resolveActiveGroup.mockResolvedValue({
    groups: [{ id: 'group-uuid', name: 'Test Group', role: 'admin' }],
    activeGroupId: 'group-uuid'
  });
  getSeasons.mockResolvedValue(seasons);
  getPointsStandings.mockResolvedValue([]);
  getScorers.mockResolvedValue([]);
  getMotmStandings.mockResolvedValue([]);

  // Children are stubbed: this file is only about the shell's own season
  // preselection, not about what MatchesPanel/the standings tables render —
  // each of those already has its own test file.
  const wrapper = shallowMount(MatchesAndStandings, {
    global: { mocks: { $route: { query: routeQuery }, $router: router }, provide }
  });
  await flushPromises();
  return wrapper;
};

// This is the exact scenario reported from real testing: an admin schedules a
// match far enough out that it introduces a season nobody has played in yet.
// ComputeSeasons (backend) counts it regardless, so the list the page receives
// already contains that future season — the fix under test is which one gets
// preselected from it.
describe('MatchesAndStandings.vue season preselection', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('defaults to the season containing today, not the last season in the list', async () => {
    vi.useFakeTimers().setSystemTime(new Date(2026, 7, 15)); // Aug 15, 2026 — still season 2025-2026
    const wrapper = await mountPage(['2024-2025', '2025-2026', '2026-2027']);

    // Without the fix this would be '2026-2027' (the last entry) even though
    // today is still in 2025-2026 — exactly the reported bug: a match
    // scheduled into next season silently hid the ongoing one's history.
    expect(wrapper.vm.selectedSeason).toBe('2025-2026');
  });

  it('still opens on the current season in the ordinary case where it is also the last one', async () => {
    vi.useFakeTimers().setSystemTime(new Date(2026, 9, 1)); // Oct 1, 2026 — season 2026-2027
    const wrapper = await mountPage(['2025-2026', '2026-2027']);

    expect(wrapper.vm.selectedSeason).toBe('2026-2027');
  });

  it("falls back to the last season on record when today's own season has no matches yet", async () => {
    vi.useFakeTimers().setSystemTime(new Date(2026, 9, 1)); // season 2026-2027, but no matches logged for it yet
    const wrapper = await mountPage(['2024-2025', '2025-2026']);

    expect(wrapper.vm.selectedSeason).toBe('2025-2026');
  });

  it('degrades to no filtering when the group has no seasons at all', async () => {
    vi.useFakeTimers().setSystemTime(new Date(2026, 9, 1));
    const wrapper = await mountPage([]);

    expect(wrapper.vm.selectedSeason).toBe('');
  });
});

// The `/m/:code` tinylink (router/index.js) now lands here with `?match=`
// rather than on the admin-only MatchDetails.vue — see CLAUDE.md. This is
// what actually resolves that query param: decode it, find which of the
// caller's groups owns it (findGroupForMatch — the exact search
// canEditMatch already does, reused rather than duplicated), force the
// Matches tab, preselect that match's own season, and hand its id down to
// MatchesPanel. Every failure mode degrades silently, mirroring the
// `/m/:code` redirect's own "malformed code falls back home" contract.
describe('MatchesAndStandings.vue shared match deep link (?match=<code>)', () => {
  beforeEach(() => {
    decodeMatchId.mockReset();
    findGroupForMatch.mockReset();
    setActiveGroupId.mockReset();
  });

  it('selects the Matches tab, preselects the match\'s own season, and passes its id to MatchesPanel', async () => {
    decodeMatchId.mockReturnValue('match-uuid');
    findGroupForMatch.mockResolvedValue({
      group: { id: 'group-uuid', role: 'admin' },
      details: { ID: 'match-uuid', Date: '2026-09-06' }
    });

    const wrapper = await mountPage(['2025-2026', '2026-2027'], { routeQuery: { match: 'abc123' } });

    expect(decodeMatchId).toHaveBeenCalledWith('abc123');
    expect(findGroupForMatch).toHaveBeenCalledWith(
      'match-uuid',
      [{ id: 'group-uuid', name: 'Test Group', role: 'admin' }]
    );
    expect(wrapper.vm.activeSubTab).toBe('matches');
    // 2026-09-06 falls in 2026-2027, the *later* of the two seasons on offer
    // — proof this overrides the ordinary seasonOf(now) default rather than
    // coincidentally matching it.
    expect(wrapper.vm.selectedSeason).toBe('2026-2027');
    expect(wrapper.findComponent({ name: 'MatchesPanel' }).props('deepLinkMatchId')).toBe('match-uuid');
    // Already the active group — no reload needed.
    expect(setActiveGroupId).not.toHaveBeenCalled();
  });

  it('switches into the match\'s own group (and reloads) when it differs from the active one', async () => {
    decodeMatchId.mockReturnValue('match-uuid');
    findGroupForMatch.mockResolvedValue({
      group: { id: 'other-group-uuid', role: 'member' },
      details: { ID: 'match-uuid', Date: '2026-09-06' }
    });
    const reload = vi.fn();
    const originalLocation = window.location;
    delete window.location;
    window.location = { ...originalLocation, reload };

    await mountPage(['2025-2026'], { routeQuery: { match: 'abc123' } });

    expect(setActiveGroupId).toHaveBeenCalledWith('other-group-uuid');
    expect(reload).toHaveBeenCalledTimes(1);

    window.location = originalLocation;
  });

  it('degrades silently when the match belongs to no group the caller is in', async () => {
    decodeMatchId.mockReturnValue('match-uuid');
    findGroupForMatch.mockResolvedValue(null);

    const wrapper = await mountPage(['2025-2026'], { routeQuery: { match: 'abc123' } });

    expect(wrapper.vm.deepLinkMatchId).toBe('');
    expect(wrapper.vm.selectedSeason).toBe('2025-2026');
  });

  it('degrades silently when the code cannot be decoded', async () => {
    decodeMatchId.mockImplementation(() => {
      throw new Error('invalid character in short link code');
    });

    const wrapper = await mountPage(['2025-2026'], { routeQuery: { match: 'not-a-real-code' } });

    expect(findGroupForMatch).not.toHaveBeenCalled();
    expect(wrapper.vm.deepLinkMatchId).toBe('');
  });

  it('does nothing when there is no ?match= param at all', async () => {
    const wrapper = await mountPage(['2025-2026']);

    expect(decodeMatchId).not.toHaveBeenCalled();
    expect(findGroupForMatch).not.toHaveBeenCalled();
    expect(wrapper.vm.deepLinkMatchId).toBe('');
  });
});

// Design v2 has a page per match: a shared or notification link opens it
// instead of preselecting the match on the home list.
describe('MatchesAndStandings.vue under design v2', () => {
  beforeEach(() => {
    decodeMatchId.mockReset();
    findGroupForMatch.mockReset();
    loadDesignV2.mockReset();
  });

  it('opens a ?match= link on the match page, without resolving it here', async () => {
    loadDesignV2.mockResolvedValue(true);
    decodeMatchId.mockReturnValue('match-uuid');
    const router = { replace: vi.fn() };

    await mountPage(['2026-2027'], { routeQuery: { match: 'abc123' }, router });

    expect(router.replace).toHaveBeenCalledWith({ name: 'MatchView', params: { id: 'match-uuid' } });
    expect(findGroupForMatch).not.toHaveBeenCalled();
  });

  it('keeps today\'s behaviour for a ?match= link when the flag is off', async () => {
    loadDesignV2.mockResolvedValue(false);
    decodeMatchId.mockReturnValue('match-uuid');
    findGroupForMatch.mockResolvedValue(null);
    const router = { replace: vi.fn() };

    await mountPage(['2026-2027'], { routeQuery: { match: 'abc123' }, router });

    expect(router.replace).not.toHaveBeenCalled();
    expect(findGroupForMatch).toHaveBeenCalled();
  });

  it('fills the shell\'s group and season selects when the flag lands after the page loaded', async () => {
    const designV2On = ref(false);
    const v2Context = { groups: [], seasons: [], groupId: '', season: '' };

    const wrapper = await mountPage(['2026-2027'], { provide: { designV2On, v2Context } });
    expect(v2Context.groups).toEqual([]);

    designV2On.value = true;
    await flushPromises();

    expect(v2Context.groupId).toBe('group-uuid');
    expect(v2Context.groups).toEqual([{ id: 'group-uuid', name: 'Test Group', role: 'admin' }]);
    expect(v2Context.seasons).toEqual(['2026-2027']);
    wrapper.unmount();
  });
});
