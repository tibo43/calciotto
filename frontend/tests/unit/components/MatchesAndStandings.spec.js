import { shallowMount, flushPromises } from '@vue/test-utils';
import MatchesAndStandings from '@/components/MatchesAndStandings.vue';
import { getPointsStandings, getScorers, getSeasons } from '@/services/api';
import { resolveActiveGroup } from '@/services/activeGroup';

jest.mock('@/services/api', () => ({
  getPointsStandings: jest.fn(),
  getScorers: jest.fn(),
  getSeasons: jest.fn()
}));

jest.mock('@/services/activeGroup', () => ({
  resolveActiveGroup: jest.fn(),
  setActiveGroupId: jest.fn()
}));

const mountPage = async (seasons) => {
  resolveActiveGroup.mockResolvedValue({
    groups: [{ id: 'group-uuid', name: 'Test Group', role: 'admin' }],
    activeGroupId: 'group-uuid'
  });
  getSeasons.mockResolvedValue(seasons);
  getPointsStandings.mockResolvedValue([]);
  getScorers.mockResolvedValue([]);

  // Children are stubbed: this file is only about the shell's own season
  // preselection, not about what MatchesPanel/the standings tables render —
  // each of those already has its own test file.
  const wrapper = shallowMount(MatchesAndStandings);
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
    jest.useRealTimers();
  });

  it('defaults to the season containing today, not the last season in the list', async () => {
    jest.useFakeTimers().setSystemTime(new Date(2026, 7, 15)); // Aug 15, 2026 — still season 2025-2026
    const wrapper = await mountPage(['2024-2025', '2025-2026', '2026-2027']);

    // Without the fix this would be '2026-2027' (the last entry) even though
    // today is still in 2025-2026 — exactly the reported bug: a match
    // scheduled into next season silently hid the ongoing one's history.
    expect(wrapper.vm.selectedSeason).toBe('2025-2026');
  });

  it('still opens on the current season in the ordinary case where it is also the last one', async () => {
    jest.useFakeTimers().setSystemTime(new Date(2026, 9, 1)); // Oct 1, 2026 — season 2026-2027
    const wrapper = await mountPage(['2025-2026', '2026-2027']);

    expect(wrapper.vm.selectedSeason).toBe('2026-2027');
  });

  it("falls back to the last season on record when today's own season has no matches yet", async () => {
    jest.useFakeTimers().setSystemTime(new Date(2026, 9, 1)); // season 2026-2027, but no matches logged for it yet
    const wrapper = await mountPage(['2024-2025', '2025-2026']);

    expect(wrapper.vm.selectedSeason).toBe('2025-2026');
  });

  it('degrades to no filtering when the group has no seasons at all', async () => {
    jest.useFakeTimers().setSystemTime(new Date(2026, 9, 1));
    const wrapper = await mountPage([]);

    expect(wrapper.vm.selectedSeason).toBe('');
  });
});
