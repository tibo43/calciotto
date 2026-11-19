import { mount, flushPromises } from '@vue/test-utils';
import EditMatchV2 from '@/components/v2/EditMatchV2.vue';
import { getMatchRegistrations, getMatchPositions } from '@/services/api';
import { findMatchInMyGroups } from '@/services/v2Match';

vi.mock('@/services/api', () => ({
  getMatchRegistrations: vi.fn(),
  getMatchPositions: vi.fn(),
  closeMatchRegistrations: vi.fn(),
  reopenMatchRegistrations: vi.fn(),
  setMatchMaxPlayers: vi.fn(),
  updateMatch: vi.fn(),
  deleteMatch: vi.fn(),
  getInviteCode: vi.fn(() => Promise.resolve({ invite_code: 'ABC' })),
  getGroupMembers: vi.fn(() => Promise.resolve([])),
  getSeasons: vi.fn(() => Promise.resolve([])),
}));

vi.mock('@/services/v2Match', async (importOriginal) => ({
  ...(await importOriginal()),
  findMatchInMyGroups: vi.fn(),
}));

const MATCH_ID = 'cccccccc-0000-4000-8000-000000000004';
const DAY = 24 * 60 * 60 * 1000;
const iso = (offset) => new Date(Date.now() + offset).toISOString();

function closedMatch(teams = [[], []]) {
  return {
    ID: MATCH_ID, GroupID: 'g', Date: iso(DAY).slice(0, 10), ScheduledAt: iso(DAY), RegistrationOpensAt: iso(-DAY),
    RegistrationsClosedAt: iso(-60 * 1000), MaxPlayers: 6,
    Teams: [{ ID: 'a', Name: 'Blue', Players: teams[0] }, { ID: 'b', Name: 'Red', Players: teams[1] }],
  };
}

const regs = ['Gigi', 'Dino', 'Ugo', 'Pia', 'Rita', 'Ezio', 'Leo'].map((name, i) => ({
  PlayerID: `p${i + 1}`, Name: name, Position: i + 1, IsWaiting: i >= 6,
}));
const positions = [
  { PlayerID: 'p1', PreferredPositions: ['GK'] },
  { PlayerID: 'p2', PreferredPositions: ['GK'] },
  { PlayerID: 'p3', PreferredPositions: ['MID', 'DEF'] },
  { PlayerID: 'p4', PreferredPositions: ['FWD'] },
];

async function mountEdit({ match = closedMatch(), positionsAnswer = Promise.resolve(positions) } = {}) {
  const group = { id: 'g', role: 'admin' };
  findMatchInMyGroups.mockResolvedValue({ group, details: match, groups: [group] });
  getMatchRegistrations.mockResolvedValue(regs);
  getMatchPositions.mockReturnValue(positionsAnswer);
  const wrapper = mount(EditMatchV2, {
    props: { id: MATCH_ID },
    global: {
      provide: { v2Context: { groups: [], seasons: [] } },
      directives: { fitTwoLines: {} },
      mocks: { $route: { query: { step: 'teams' } }, $router: { replace: vi.fn(), push: vi.fn() } },
      stubs: { 'router-link': { template: '<a><slot /></a>' }, ShareButton: true },
    },
  });
  await flushPromises();
  return wrapper;
}

const teamCols = (wrapper) => wrapper.findAll('.compose-col').slice(1);

beforeEach(() => vi.clearAllMocks());

describe('EditMatchV2.vue preferred positions', () => {
  it('asks for the positions of the match, and tags each reserve row on the right', async () => {
    const wrapper = await mountEdit();
    expect(getMatchPositions).toHaveBeenCalledWith(MATCH_ID);
    const ugo = wrapper.findAll('.p-row').find((row) => row.text().includes('Ugo'));
    expect(ugo.classes()).toContain('has-tag');
    expect(ugo.find('.pos-tag').text()).toBe('DEF · MID');
    const rita = wrapper.findAll('.p-row').find((row) => row.text().includes('Rita'));
    expect(rita.classes()).not.toContain('has-tag');
    expect(rita.find('.pos-tag').exists()).toBe(false);
  });

  it('splits the confirmed players with one goalkeeper a side and a summary under each team', async () => {
    const wrapper = await mountEdit();
    await wrapper.find('.compose-toolbar .btn').trigger('click');
    const cols = teamCols(wrapper);
    expect(cols.map((col) => col.findAll('.p-row').length)).toEqual([3, 3]);
    cols.forEach((col) => expect(col.find('.pos-summary').text()).toMatch(/^GK 1 · DEF \d · MID \d · FWD \d$/));
    // The waiting list stays out.
    expect(wrapper.findAll('.compose-col .p-row').some((row) => row.text().includes('Leo'))).toBe(false);
    expect(wrapper.find('.compose-toolbar .hint').text()).toContain('split at random');
  });

  it("keeps an admin's placed players where they are", async () => {
    const wrapper = await mountEdit({ match: closedMatch([[{ ID: 'p1', Name: 'Gigi', GoalNumber: 0 }], []]) });
    await wrapper.find('.compose-toolbar .btn').trigger('click');
    const [blue, red] = teamCols(wrapper);
    expect(blue.findAll('.p-row')[0].text()).toContain('Gigi');
    expect(red.text()).toContain('Dino');
  });

  it('still splits, at random, when the positions cannot be read', async () => {
    const wrapper = await mountEdit({ positionsAnswer: Promise.reject(new Error('403')) });
    expect(wrapper.find('.pos-tag').exists()).toBe(false);
    await wrapper.find('.compose-toolbar .btn').trigger('click');
    expect(teamCols(wrapper).map((col) => col.findAll('.p-row').length)).toEqual([3, 3]);
    teamCols(wrapper).forEach((col) => expect(col.find('.pos-summary').text()).toBe('GK 0 · DEF 0 · MID 0 · FWD 0'));
  });
});
