import {
  POSITIONS, normalizePositions, positionsTag, positionsMap, positionCounts, positionSummary, splitByPositions,
} from '@/services/v2Positions';
import { fillTeamsFromRegistrations } from '@/services/matchRegistration';

// A small seeded generator, so every run of a test draws the same numbers.
function seeded(seed) {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6D2B79F5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

const entry = (id, position = id, waiting = false) => ({ PlayerID: `p${id}`, Name: `Player ${id}`, Position: position, IsWaiting: waiting });
const onTeam = (id) => ({ ID: `p${id}`, Name: `Player ${id}`, GoalNumber: 2, OwnGoals: 0 });
const ids = (roster) => roster.map((player) => player.ID);

describe('position helpers', () => {
  it('normalizes to known values, deduplicated, in canonical order', () => {
    expect(POSITIONS).toEqual(['GK', 'DEF', 'MID', 'FWD']);
    expect(normalizePositions(['FWD', 'gk', 'FWD', 'ST', null])).toEqual(['GK', 'FWD']);
    expect(normalizePositions(undefined)).toEqual([]);
  });

  it('writes the tag in canonical order with a middle dot, nothing for no position', () => {
    expect(positionsTag(['MID', 'DEF'])).toBe('DEF · MID');
    expect(positionsTag([])).toBe('');
  });

  it('maps the admin route answer by player id, leaving out empty lists', () => {
    expect(positionsMap([
      { PlayerID: 'a', PreferredPositions: ['MID', 'GK'] },
      { PlayerID: 'b', PreferredPositions: [] },
      null,
    ])).toEqual({ a: ['GK', 'MID'] });
    expect(positionsMap(undefined)).toEqual({});
  });

  it('counts a multi-position player once in each of their positions', () => {
    const map = { p1: ['GK'], p2: ['DEF', 'MID'], p3: ['DEF'], p4: ['FWD'] };
    const roster = [onTeam(1), onTeam(2), onTeam(3), onTeam(4), onTeam(5)];
    expect(positionCounts(roster, map)).toEqual({ GK: 1, DEF: 2, MID: 1, FWD: 1 });
    expect(positionSummary(roster, map)).toBe('GK 1 · DEF 2 · MID 1 · FWD 1');
    expect(positionSummary([], {})).toBe('GK 0 · DEF 0 · MID 0 · FWD 0');
  });
});

describe('splitByPositions', () => {
  const positions = {
    p1: ['GK'], p2: ['GK'], p3: ['DEF'], p4: ['DEF'], p5: ['MID'], p6: ['MID'],
    p7: ['FWD'], p8: ['FWD'], p9: ['DEF', 'MID'], p10: [],
  };
  const regs = Array.from({ length: 10 }, (_, i) => entry(i + 1));

  it('gives each team one goalkeeper and balances every position, for any draw', () => {
    for (let seed = 1; seed <= 200; seed += 1) {
      const { rosters, addedCount } = splitByPositions(regs, [[], []], positions, seeded(seed));
      expect(addedCount).toBe(10);
      expect(rosters.map((r) => r.length)).toEqual([5, 5]);
      const counts = rosters.map((r) => positionCounts(r, positions));
      expect(counts[0].GK).toBe(1);
      expect(counts[1].GK).toBe(1);
      expect(counts[0].FWD).toBe(1);
      expect(counts[1].FWD).toBe(1);
      expect(Math.abs(counts[0].DEF - counts[1].DEF)).toBeLessThanOrEqual(1);
      expect(Math.abs(counts[0].MID - counts[1].MID)).toBeLessThanOrEqual(1);
      expect(new Set([...ids(rosters[0]), ...ids(rosters[1])]).size).toBe(10);
    }
  });

  it('keeps every position at most one apart when multi-position players are many, counting them in each', () => {
    const map = {
      p1: ['MID', 'FWD'], p2: ['FWD'], p3: ['GK'], p4: ['FWD'], p5: ['DEF'], p6: ['DEF', 'MID'],
      p7: ['GK', 'DEF'], p8: ['MID'], p9: ['DEF'], p11: ['MID'],
    };
    const eleven = Array.from({ length: 11 }, (_, i) => entry(i + 1));
    for (let seed = 1; seed <= 300; seed += 1) {
      const { rosters, placed } = splitByPositions(eleven, [[], []], map, seeded(seed));
      const counts = rosters.map((r) => positionCounts(r, map));
      POSITIONS.forEach((position) => expect(Math.abs(counts[0][position] - counts[1][position])).toBeLessThanOrEqual(1));
      expect(Math.abs(rosters[0].length - rosters[1].length)).toBe(1);
      // What it reports matches where everyone ended up.
      placed.forEach(({ teamIndex, name }) => expect(rosters[teamIndex].some((p) => p.Name === name)).toBe(true));
    }
  });

  it('is random: different draws give different teams, the same seed the same teams', () => {
    const a = splitByPositions(regs, [[], []], positions, seeded(1)).rosters;
    const again = splitByPositions(regs, [[], []], positions, seeded(1)).rosters;
    expect(again).toEqual(a);
    const distinct = new Set();
    for (let seed = 1; seed <= 30; seed += 1) {
      distinct.add(ids(splitByPositions(regs, [[], []], positions, seeded(seed)).rosters[0]).sort().join(','));
    }
    expect(distinct.size).toBeGreaterThan(5);
  });

  it('only places confirmed players and leaves placed players where they are, goals included', () => {
    const list = [entry(1), entry(2), entry(3), entry(4), entry(5, 5, true)];
    const current = [[onTeam(1)], []];
    const { rosters, addedCount } = splitByPositions(list, current, positions, seeded(3));
    expect(addedCount).toBe(3);
    expect(rosters[0][0]).toEqual(onTeam(1));
    expect([...ids(rosters[0]), ...ids(rosters[1])]).not.toContain('p5');
    expect(current).toEqual([[onTeam(1)], []]);
    // The goalkeeper already placed counts: the other one goes across.
    expect(ids(rosters[1])).toContain('p2');
    rosters.flat().filter((p) => p.ID !== 'p1').forEach((p) => expect(p.GoalNumber).toBe(0));
  });

  it('keeps team sizes at most one apart, counting the players already placed', () => {
    for (let seed = 1; seed <= 50; seed += 1) {
      const current = [[onTeam(1), onTeam(3), onTeam(5)], []];
      const { rosters } = splitByPositions(regs, current, positions, seeded(seed));
      expect(rosters.map((r) => r.length).sort()).toEqual([5, 5]);
      const odd = splitByPositions(regs.slice(0, 9), [[], []], positions, seeded(seed)).rosters;
      expect(Math.abs(odd[0].length - odd[1].length)).toBe(1);
    }
    // More already placed on one side than there are newcomers: all go across.
    const lopsided = splitByPositions([entry(7), entry(8)], [[onTeam(1), onTeam(3), onTeam(5), onTeam(6)], []], positions, seeded(1));
    expect(lopsided.rosters.map((r) => r.length)).toEqual([4, 2]);
  });

  it('uses a multi-position player to fill a missing position, before single-position ones elsewhere', () => {
    // Two defenders only on paper: p3 (DEF) and p9 (DEF, MID); one midfielder p5.
    const map = { p3: ['DEF'], p4: ['DEF'], p5: ['MID'], p9: ['DEF', 'MID'] };
    for (let seed = 1; seed <= 100; seed += 1) {
      const { rosters } = splitByPositions([entry(3), entry(4), entry(5), entry(9)], [[], []], map, seeded(seed));
      const counts = rosters.map((r) => positionCounts(r, map));
      // Each team can field a defender and a midfielder: p9 is the joker.
      expect(counts[0].DEF).toBeGreaterThanOrEqual(1);
      expect(counts[1].DEF).toBeGreaterThanOrEqual(1);
      expect(counts[0].MID).toBe(1);
      expect(counts[1].MID).toBe(1);
    }
  });

  it('spreads the players with no position at random, after the others', () => {
    const none = Array.from({ length: 6 }, (_, i) => entry(i + 1));
    const seen = new Set();
    for (let seed = 1; seed <= 30; seed += 1) {
      const { rosters } = splitByPositions(none, [[], []], {}, seeded(seed));
      expect(rosters.map((r) => r.length)).toEqual([3, 3]);
      seen.add(ids(rosters[0]).sort().join(','));
    }
    expect(seen.size).toBeGreaterThan(3);
  });

  it('answers like the v1 split when there is nothing to place', () => {
    const current = [[onTeam(1)], [onTeam(2)]];
    expect(splitByPositions([entry(1), entry(2)], current, positions)).toEqual({ rosters: current, addedCount: 0, placed: [] });
    expect(splitByPositions([], [[]], {})).toEqual({ rosters: [[]], addedCount: 0, placed: [] });
  });

  it('leaves the v1 split untouched: still sign-up order, no positions', () => {
    const { rosters } = fillTeamsFromRegistrations(regs.slice(0, 4), [[], []]);
    expect(ids(rosters[0])).toEqual(['p1', 'p3']);
    expect(ids(rosters[1])).toEqual(['p2', 'p4']);
  });
});
