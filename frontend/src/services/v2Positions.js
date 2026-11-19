// Preferred positions (design v2 only): what a player picks on their Account
// page, what a group admin sees next to names on the match page and the team
// editor, and what the editor's automatic split balances on. The v1 split,
// fillTeamsFromRegistrations in matchRegistration.js, does not use any of this.

// The four positions, in canonical order: every list is shown and stored in it.
export const POSITIONS = ['GK', 'DEF', 'MID', 'FWD'];

// A position list cleaned up: known values only, deduplicated, canonical order.
export function normalizePositions(list) {
  const picked = new Set((Array.isArray(list) ? list : []).map((value) => String(value || '').trim().toUpperCase()));
  return POSITIONS.filter((position) => picked.has(position));
}

// "DEF · MID", or '' for a player with no position.
export function positionsTag(list) {
  return normalizePositions(list).join(' · ');
}

// GET /matches/:id/positions' answer as { [playerId]: ['DEF', 'MID'] }.
export function positionsMap(entries) {
  const map = {};
  (Array.isArray(entries) ? entries : []).forEach((entry) => {
    if (!entry || !entry.PlayerID) return;
    const positions = normalizePositions(entry.PreferredPositions);
    if (positions.length) map[entry.PlayerID] = positions;
  });
  return map;
}

// How many players of a roster can play each position. A player with several
// positions counts once in each of them, so the four numbers can add up to more
// than the roster.
export function positionCounts(players, positionsById) {
  const counts = { GK: 0, DEF: 0, MID: 0, FWD: 0 };
  (Array.isArray(players) ? players : []).forEach((player) => {
    const positions = (positionsById && player && positionsById[player.ID]) || [];
    positions.forEach((position) => {
      if (position in counts) counts[position] += 1;
    });
  });
  return counts;
}

// "GK 1 · DEF 2 · MID 2 · FWD 1" under a team in the editor.
export function positionSummary(players, positionsById) {
  const counts = positionCounts(players, positionsById);
  return POSITIONS.map((position) => `${position} ${counts[position]}`).join(' · ');
}

function shuffle(list, rng) {
  const out = [...list];
  for (let i = out.length - 1; i > 0; i -= 1) {
    const j = Math.floor(rng() * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}

const isOnRoster = (roster, entry) => roster.some((player) => {
  if (!player) return false;
  if (player.ID && entry.PlayerID) return player.ID === entry.PlayerID;
  const name = String(player.Name || '').toLowerCase();
  return Boolean(name) && name === String(entry.Name || '').toLowerCase();
});

// The automatic split of the v2 team editor. Pure: the randomness comes from
// `rng` (Math.random by default, a seeded one in the tests), nothing is
// mutated, and the result has fillTeamsFromRegistrations' shape so the editor
// treats both alike.
//
//   - Additive, like the v1 split: only CONFIRMED sign-ups are placed, and a
//     player already on a team stays where they are.
//   - Team sizes end up at most one apart (when the players already placed
//     allow it; otherwise every newcomer goes to the smaller team). Which team
//     gets the odd player out is random.
//   - Coverage first, in canonical order: for GK, then DEF, MID and FWD, each
//     team that has nobody able to play that position gets one, drawn at random
//     among those who play only that position, or failing that among those who
//     play the fewest positions (a multi-position player is the joker that
//     fills the gap). Players already placed count: a team that has a
//     goalkeeper does not get another one here.
//   - Then the rest of the players with a position, by main position (the first
//     of theirs in canonical order), shuffled within each position: each goes
//     to the team where, over all of their positions, the two teams' counts
//     end up closest (a multi-position player counts in each), then the team
//     with more room left, then at random.
//   - Then the players with no position, shuffled, to the team with more room
//     left (a tie is random).
//   - Last, newcomers on opposite teams are swapped while a swap makes the
//     teams more even position by position (see step 4 below).
export function splitByPositions(registrations, currentRosters, positionsById = {}, rng = Math.random) {
  const rosters = (Array.isArray(currentRosters) ? currentRosters : [])
    .map((roster) => (Array.isArray(roster) ? [...roster] : []));
  if (rosters.length < 2) return { rosters, addedCount: 0, placed: [] };

  const positionsOf = (id) => (positionsById && positionsById[id]) || [];
  const pool = shuffle((Array.isArray(registrations) ? registrations : [])
    .filter((entry) => entry && !entry.IsWaiting)
    .filter((entry) => !rosters.some((roster) => isOnRoster(roster, entry))), rng);
  if (!pool.length) return { rosters, addedCount: 0, placed: [] };

  // Room left on each team, so that the final sizes are at most one apart.
  const total = rosters[0].length + rosters[1].length + pool.length;
  const targets = [Math.floor(total / 2), Math.floor(total / 2)];
  if (total % 2) {
    if (rosters[0].length > targets[0]) targets[0] += 1;
    else if (rosters[1].length > targets[1]) targets[1] += 1;
    else targets[rng() < 0.5 ? 0 : 1] += 1;
  }
  const room = targets.map((target, index) => Math.max(0, target - rosters[index].length));

  const covers = (index, position) => rosters[index]
    .filter((player) => player && positionsOf(player.ID).includes(position)).length;
  const placedEntries = [];
  const remaining = new Set(pool);
  const place = (entry, index) => {
    rosters[index].push({ ID: entry.PlayerID, Name: entry.Name, GoalNumber: 0, OwnGoals: 0 });
    room[index] -= 1;
    remaining.delete(entry);
    placedEntries.push(entry);
  };
  const teamOrder = () => (rng() < 0.5 ? [0, 1] : [1, 0]);

  // 1. Coverage: one player per missing position and team.
  POSITIONS.forEach((position) => {
    teamOrder().forEach((index) => {
      if (room[index] <= 0 || covers(index, position) > 0) return;
      const candidates = [...remaining].filter((entry) => positionsOf(entry.PlayerID).includes(position));
      if (!candidates.length) return;
      const fewest = Math.min(...candidates.map((entry) => positionsOf(entry.PlayerID).length));
      // The pool is already shuffled, so the first of the fewest is a random one.
      place(candidates.find((entry) => positionsOf(entry.PlayerID).length === fewest), index);
    });
  });

  // How uneven the two teams would be, position by position, with `entry`
  // on team `index`: only the entry's own positions change, so only they count.
  const unevenness = (entry, index) => positionsOf(entry.PlayerID).reduce((sum, position) => {
    const mine = covers(index, position) + 1;
    const theirs = covers(1 - index, position);
    return sum + Math.abs(mine - theirs);
  }, 0);
  const pickTeam = (entry) => {
    const open = teamOrder().filter((index) => room[index] > 0);
    if (open.length < 2) return open[0];
    const [a, b] = open;
    const diff = unevenness(entry, a) - unevenness(entry, b);
    if (diff !== 0) return diff < 0 ? a : b;
    if (room[a] !== room[b]) return room[a] > room[b] ? a : b;
    return a;
  };

  // 2. The other players with a position, by main position.
  POSITIONS.forEach((position) => {
    [...remaining]
      .filter((entry) => positionsOf(entry.PlayerID)[0] === position)
      .forEach((entry) => {
        const index = pickTeam(entry);
        if (index !== undefined) place(entry, index);
      });
  });

  // 3. The players with no position.
  [...remaining].forEach((entry) => {
    const index = pickTeam(entry);
    if (index !== undefined) place(entry, index);
  });

  // 4. Polish: swap two newcomers from opposite teams while that makes the
  // teams more even (sizes never change, players already placed never move).
  // A team losing its only player at a position the other team has twice is
  // what weighs most, then the gap at each position. The greedy steps above
  // can leave a gap a single swap closes, mostly when multi-position players
  // are many.
  const score = () => POSITIONS.reduce((sum, position) => {
    const a = covers(0, position);
    const b = covers(1, position);
    const stranded = (a === 0 && b >= 2) || (b === 0 && a >= 2) ? 10 : 0;
    return sum + stranded + Math.abs(a - b);
  }, 0);
  const fresh = new Set(pool.map((entry) => entry.PlayerID));
  const isFresh = (player) => player && fresh.has(player.ID);
  for (let pass = 0, improved = true; improved && pass < 50; pass += 1) {
    improved = false;
    const current = score();
    for (let i = 0; i < rosters[0].length && !improved; i += 1) {
      if (!isFresh(rosters[0][i])) continue;
      for (let j = 0; j < rosters[1].length && !improved; j += 1) {
        if (!isFresh(rosters[1][j])) continue;
        [rosters[0][i], rosters[1][j]] = [rosters[1][j], rosters[0][i]];
        if (score() < current) improved = true;
        else [rosters[0][i], rosters[1][j]] = [rosters[1][j], rosters[0][i]];
      }
    }
  }
  const teamOf = new Map();
  rosters.forEach((roster, index) => roster.forEach((player) => teamOf.set(player.ID, index)));
  const placed = placedEntries.map((entry) => ({ teamIndex: teamOf.get(entry.PlayerID), name: entry.Name }));

  return { rosters, addedCount: placed.length, placed };
}
