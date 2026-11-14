import { teamsAreComposed } from './matchRegistration';

// A side's score is its own goals plus the other side's own goals. Nothing
// here assumes the two sides are the same size.
export function scoreSides(rosters) {
  const sides = (Array.isArray(rosters) ? rosters : []).slice(0, 2);
  while (sides.length < 2) sides.push([]);
  const own = sides.map((players) => (players || []).reduce((sum, player) => sum + (Number(player.OwnGoals) || 0), 0));
  const goals = sides.map((players) => (players || []).reduce((sum, player) => sum + (Number(player.GoalNumber) || 0), 0));
  const totalOwn = own[0] + own[1];
  return [0, 1].map((index) => ({
    goals: goals[index],
    ownGoals: own[index],
    concededOwnGoals: totalOwn - own[index],
    score: goals[index] + (totalOwn - own[index]),
  }));
}

export function teamScoreLine(side) {
  const total = side.score === 1 ? '1 goal' : `${side.score} goals`;
  if (!side.concededOwnGoals) return total;
  const og = side.concededOwnGoals === 1 ? '1 OG' : `${side.concededOwnGoals} OG`;
  return `${total} · ${og}`;
}

// Headcount only. Uneven sides are allowed, so this never becomes a warning.
export function sizeCounter(countA, countB) {
  return `${countA} v ${countB}`;
}

export function lastName(name) {
  const parts = String(name || '').trim().split(/\s+/).filter(Boolean);
  return parts.length ? parts[parts.length - 1] : '';
}

// Goals stay with the scorer's side. An own goal is listed on the side that
// benefits, as "Greco (OG ×2)".
export function recapEntries(rosters) {
  const entries = [];
  (rosters || []).slice(0, 2).forEach((players, index) => {
    (players || []).forEach((player) => {
      const goals = Number(player.GoalNumber) || 0;
      const own = Number(player.OwnGoals) || 0;
      const name = lastName(player.Name);
      if (!name) return;
      if (goals === 1) entries.push({ text: name, away: index === 1 });
      else if (goals > 1) entries.push({ text: `${name} ×${goals}`, away: index === 1 });
      if (own > 0) entries.push({ text: `${name} (OG ×${own})`, away: index === 0 });
    });
  });
  return entries;
}

export function lineupsReady(teams) {
  return teamsAreComposed({ Teams: teams });
}

export function shareSubtitle(teams) {
  if (!lineupsReady(teams)) {
    return 'The match announcement will be sent — the date and the sign-up link.';
  }
  const counts = (teams || []).slice(0, 2).map((team) => (
    `${team.Name} ${((team && team.Players) || []).length}`
  ));
  return `The line-ups will be sent — ${counts.join(' · ')}.`;
}
