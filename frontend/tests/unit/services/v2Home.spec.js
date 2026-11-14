import { splitHomeMatches, heroDateLabel, placesLeft, registrationMeta, kickoffLabel, homeVoteLine, matchVoteCopy, goalLine } from '@/services/v2Home';

const NOW = new Date('2026-09-04T10:00:00+02:00').getTime();

const played = { ID: 'p', Date: '2026-08-30', ScheduledAt: '2026-08-30T20:30:00+02:00' };
const next = {
  ID: 'n', Date: '2026-09-06', ScheduledAt: '2026-09-06T20:30:00+02:00',
  RegistrationOpensAt: '2026-09-01T18:00:00+02:00', MaxPlayers: 16,
};
const later = {
  ID: 'l', Date: '2026-09-13', ScheduledAt: '2026-09-13T19:45:00+02:00',
  RegistrationOpensAt: '2026-09-10T18:00:00+02:00', MaxPlayers: 16,
};

describe('splitHomeMatches', () => {
  it('puts the soonest match still ahead in the hero and the rest under Later', () => {
    const split = splitHomeMatches([played, later, next], NOW);
    expect(split.next.ID).toBe('n');
    expect(split.later.map((m) => m.ID)).toEqual(['l']);
    expect(split.recent.map((m) => m.ID)).toEqual(['p']);
  });

  it('uses the most recent finished match as the hero when nothing is ahead', () => {
    const split = splitHomeMatches([played], NOW);
    expect(split.next).toBeNull();
    expect(split.recent[0].ID).toBe('p');
  });
});

describe('v2 labels', () => {
  it('formats the hero date in English', () => {
    expect(heroDateLabel('2026-11-13')).toBe('Fri 13 Nov');
  });

  it('shows the kick-off in the viewer local zone, and nothing when there is no kick-off', () => {
    const iso = '2026-09-06T20:30:00+02:00';
    const at = new Date(iso);
    const hours = String(at.getHours()).padStart(2, '0');
    const minutes = String(at.getMinutes()).padStart(2, '0');
    expect(kickoffLabel(iso)).toBe(`${hours}:${minutes}`);
    expect(kickoffLabel('2026-09-06')).toBe('');
    expect(kickoffLabel('')).toBe('');
  });

  it('describes an open list and the places still free', () => {
    expect(registrationMeta(next, NOW)).toBe('Sign-ups open');
    expect(placesLeft(next, 13)).toBe(3);
  });

  it('writes goals and own goals as plain text', () => {
    expect(goalLine({ GoalNumber: 3, OwnGoals: 2 })).toBe('3 goals · 2 OG');
    expect(goalLine({ GoalNumber: 1, OwnGoals: 0 })).toBe('1 goal');
  });
});

describe('MOTM copy', () => {
  const openMatch = { ID: 'm', Date: '2026-09-03' };
  const closedMatch = { ID: 'm', Date: '2026-08-30' };
  const tally = { Tally: [{ PlayerID: 'a', Name: 'Marco Rossi', Votes: 4 }] };
  const tie = {
    Tally: [
      { PlayerID: 'a', Name: 'Marco Rossi', Votes: 3 },
      { PlayerID: 'b', Name: 'Andrea Conti', Votes: 3 },
    ],
  };

  it('points home at the match while voting is open', () => {
    expect(homeVoteLine(openMatch, tally, NOW)).toBe('MOTM vote open · Vote in match details');
  });

  it('names the winner once voting has closed, and stays quiet when nobody voted', () => {
    expect(homeVoteLine(closedMatch, tally, NOW)).toBe('MOTM: Marco Rossi');
    expect(homeVoteLine(closedMatch, { Tally: [] }, NOW)).toBeNull();
  });

  it('gives the deadline, a tie, and a no-vote close on the match page', () => {
    expect(matchVoteCopy(openMatch, tally, NOW)).toBe('MOTM vote open until 4 Sep midnight · tap a player to change');
    expect(matchVoteCopy(closedMatch, tie, NOW)).toBe('MOTM vote closed · tie, both receive the award');
    expect(matchVoteCopy(closedMatch, { Tally: [] }, NOW)).toBe('MOTM vote closed · no votes, no winner');
  });
});
