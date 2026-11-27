import { splitHomeMatches, heroDateLabel, placesLeft, registrationMeta, kickoffLabel, homeVoteLine, rowSubLine, weekdayTimeLabel, dayMonthLabel, matchVoteCopy, goalLine, myTeam, nextWaitingPosition, voteErrorMessage } from '@/services/v2Home';

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

  it('leaves the hero empty when nothing is ahead, the finished match going under Latest', () => {
    const split = splitHomeMatches([played], NOW);
    expect(split.next).toBeNull();
    expect(split.recent[0].ID).toBe('p');
  });

  it('counts a match as over once its score is recorded, even before its day ends', () => {
    const today = { ...next, ID: 'today', ResultRecordedAt: new Date(NOW).toISOString() };
    const split = splitHomeMatches([today, later], NOW);
    expect(split.next.ID).toBe('l');
    expect(split.recent.map((m) => m.ID)).toEqual(['today']);
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

describe('Star of the Match copy', () => {
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
    expect(homeVoteLine(openMatch, NOW)).toBe('Star of the Match vote open · Vote in match details');
  });

  it('says nothing once voting has closed, or for a match still ahead', () => {
    expect(homeVoteLine(closedMatch, NOW)).toBeNull();
    expect(homeVoteLine({ ID: 'n', Date: '2026-09-06', ScheduledAt: '2026-09-06T20:30:00+02:00' }, NOW)).toBeNull();
    expect(homeVoteLine(null, NOW)).toBeNull();
  });

  it('reads a row day as « 4 Sep »', () => {
    expect(dayMonthLabel('2026-09-04')).toBe('4 Sep');
    expect(dayMonthLabel('2026-08-30T00:00:00Z')).toBe('30 Aug');
    expect(dayMonthLabel('')).toBe('');
  });

  it('puts MOTM vote open in place of a Latest row kick-off time while voting is open', () => {
    const friday = { ...openMatch, ScheduledAt: '2026-09-04T08:00:00+02:00', ResultRecordedAt: '2026-09-04T09:30:00+02:00', Date: '2026-09-04' };
    expect(rowSubLine(friday, NOW)).toBe('Star of the Match vote open');
    expect(rowSubLine({ ...friday, ScheduledAt: null }, NOW)).toBe('Star of the Match vote open');
    const at = '2026-08-28T08:00:00+02:00';
    expect(rowSubLine({ ...closedMatch, ScheduledAt: at }, NOW)).toBe(weekdayTimeLabel(at));
    expect(weekdayTimeLabel(at)).not.toBe('');
    expect(rowSubLine(closedMatch, NOW)).toBe('');
    expect(rowSubLine(null, NOW)).toBe('');
  });

  it('gives the deadline, a tie, and a no-vote close on the match page', () => {
    expect(matchVoteCopy(openMatch, tally, NOW)).toBe('Star of the Match vote open until 4 Sep midnight · tap a player to change');
    // A scheduled match closes 24 hours after kick-off, shown in the viewer's zone.
    const kickoff = '2026-09-03T19:45:00+02:00';
    const deadline = new Date(Date.parse(kickoff) + 24 * 60 * 60 * 1000);
    const time = `${String(deadline.getHours()).padStart(2, '0')}:${String(deadline.getMinutes()).padStart(2, '0')}`;
    expect(matchVoteCopy({ ...openMatch, ScheduledAt: kickoff }, tally, NOW))
      .toBe(`Star of the Match vote open until ${deadline.getDate()} Sep ${time} · tap a player to change`);
    expect(matchVoteCopy(closedMatch, tie, NOW)).toBe('Star of the Match vote closed · tie, both receive the award');
    expect(matchVoteCopy(closedMatch, { Tally: [] }, NOW)).toBe('Star of the Match vote closed · no votes, no winner');
  });
});

describe('myTeam', () => {
  const match = { Teams: [{ Name: 'Black', Players: [{ ID: 'me' }] }, { Name: 'White', Players: [{ ID: 'q' }] }] };
  it("names the player's team, or nothing when they are in neither", () => {
    expect(myTeam(match, 'me')).toBe('Black');
    expect(myTeam(match, 'other')).toBeNull();
    expect(myTeam(match, '')).toBeNull();
  });
});

describe('nextWaitingPosition', () => {
  it('lands right after the players already waiting', () => {
    expect(nextWaitingPosition([{ IsWaiting: false }, { IsWaiting: true }, { IsWaiting: true }])).toBe(3);
    expect(nextWaitingPosition([])).toBe(1);
  });
});


describe('voteErrorMessage', () => {
  const refusal = (error) => ({ response: { data: { error } } });
  const GENERIC = 'Error updating your Star of the Match vote.';

  it("renames the award in each of the backend's vote refusals, whatever the case", () => {
    expect(voteErrorMessage(refusal('you cannot vote for yourself for man of the match')))
      .toBe('You cannot vote for yourself for Star of the Match.');
    expect(voteErrorMessage(refusal("only players on this match's roster can vote for man of the match")))
      .toBe("Only players on this match's roster can vote for Star of the Match.");
    expect(voteErrorMessage(refusal('voting for man of the match is closed for this match')))
      .toBe('Voting for Star of the Match is closed for this match.');
    expect(voteErrorMessage(refusal('Man Of The Match: MAN OF THE MATCH, men of the match')))
      .toBe('Star of the Match: Star of the Match, Star of the Match.');
  });

  it('writes the reason as a sentence: capital first letter, one closing full stop', () => {
    expect(voteErrorMessage(refusal('voting for man of the match is closed for this match')))
      .toBe('Voting for Star of the Match is closed for this match.');
    expect(voteErrorMessage(refusal('Already a sentence.'))).toBe('Already a sentence.');
    expect(voteErrorMessage(refusal('  is it closed?  '))).toBe('Is it closed?');
    expect(voteErrorMessage(refusal('closed!'))).toBe('Closed!');
  });

  it('keeps a reason that never names the award', () => {
    expect(voteErrorMessage(refusal('match not found'))).toBe('Match not found.');
  });

  it('gives the generic message for a reason still naming the award the old way, or no reason', () => {
    expect(voteErrorMessage(refusal('MOTM voting is closed'))).toBe(GENERIC);
    expect(voteErrorMessage(refusal('motm'))).toBe(GENERIC);
    expect(voteErrorMessage(refusal('man-of-the-match vote closed'))).toBe('Star of the Match vote closed.');
    expect(voteErrorMessage(refusal(''))).toBe(GENERIC);
    expect(voteErrorMessage(refusal({ code: 1 }))).toBe(GENERIC);
    expect(voteErrorMessage(new Error('Failed to cast Man of the Match vote'))).toBe(GENERIC);
    expect(voteErrorMessage(undefined)).toBe(GENERIC);
  });
});
