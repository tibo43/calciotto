import {
  isScheduledMatch,
  deriveRegistrationState,
  registrationsAreOpen,
  fillTeamsFromRegistrations,
  REGISTRATION_UNSCHEDULED,
  REGISTRATION_NOT_OPEN_YET,
  REGISTRATION_OPEN,
  REGISTRATION_CLOSED_BY_ADMIN,
  REGISTRATION_CLOSED_AT_KICKOFF
} from '@/services/matchRegistration';

// Fixed instants with explicit offsets, so nothing here depends on the machine's
// zone: Date.parse resolves each to an absolute epoch value either way.
const OPENS_AT = '2026-09-01T12:00:00+02:00';
const KICKOFF = '2026-09-06T20:30:00+02:00';

const at = (rfc3339) => Date.parse(rfc3339);

const scheduled = (overrides = {}) => ({
  ScheduledAt: KICKOFF,
  RegistrationOpensAt: OPENS_AT,
  MaxPlayers: 16,
  RegistrationCount: 0,
  ...overrides
});

describe('isScheduledMatch', () => {
  it('is true for a match carrying a kick-off', () => {
    expect(isScheduledMatch(scheduled())).toBe(true);
  });

  it('is false for an ordinary match, which carries none of the scheduling keys', () => {
    expect(isScheduledMatch({ ID: 'm', Date: '2026-09-06', Teams: [] })).toBe(false);
  });

  it('is false for a null/undefined match rather than throwing', () => {
    expect(isScheduledMatch(null)).toBe(false);
    expect(isScheduledMatch(undefined)).toBe(false);
  });
});

describe('deriveRegistrationState', () => {
  it('reports an unscheduled match as such, whatever the clock says', () => {
    expect(deriveRegistrationState({ ID: 'm' }, at('2026-09-03T12:00:00+02:00')))
      .toBe(REGISTRATION_UNSCHEDULED);
  });

  it('is not-open-yet before the sign-ups open', () => {
    expect(deriveRegistrationState(scheduled(), at('2026-09-01T11:59:00+02:00')))
      .toBe(REGISTRATION_NOT_OPEN_YET);
  });

  it('is open from the opening instant itself', () => {
    expect(deriveRegistrationState(scheduled(), at(OPENS_AT))).toBe(REGISTRATION_OPEN);
  });

  it('is open between the opening and kick-off', () => {
    expect(deriveRegistrationState(scheduled(), at('2026-09-04T09:00:00+02:00')))
      .toBe(REGISTRATION_OPEN);
  });

  it('is closed-by-admin once RegistrationsClosedAt is present, even mid-window', () => {
    const match = scheduled({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' });
    expect(deriveRegistrationState(match, at('2026-09-05T19:00:00+02:00')))
      .toBe(REGISTRATION_CLOSED_BY_ADMIN);
  });

  // The two closed states have to stay distinguishable: only the admin's one is
  // reversible, and Reopen is offered on exactly that one.
  it('prefers closed-by-admin over not-open-yet, so the copy never promises an opening that was already cancelled', () => {
    const match = scheduled({ RegistrationsClosedAt: '2026-08-31T10:00:00+02:00' });
    expect(deriveRegistrationState(match, at('2026-08-31T11:00:00+02:00')))
      .toBe(REGISTRATION_CLOSED_BY_ADMIN);
  });

  it('is closed-at-kickoff at the kick-off instant, with no admin close at all', () => {
    expect(deriveRegistrationState(scheduled(), at(KICKOFF)))
      .toBe(REGISTRATION_CLOSED_AT_KICKOFF);
  });

  it('is closed-at-kickoff after kick-off', () => {
    expect(deriveRegistrationState(scheduled(), at('2026-09-06T22:00:00+02:00')))
      .toBe(REGISTRATION_CLOSED_AT_KICKOFF);
  });

  it('treats a scheduled match with an unparseable opening time as having no lower bound', () => {
    const match = scheduled({ RegistrationOpensAt: undefined });
    expect(deriveRegistrationState(match, at('2026-09-02T12:00:00+02:00')))
      .toBe(REGISTRATION_OPEN);
  });
});

describe('registrationsAreOpen', () => {
  // Both Participate and Withdraw hang off this one predicate, because the
  // backend refuses both outside the same window.
  it('is true only for the open state', () => {
    expect(registrationsAreOpen(REGISTRATION_OPEN)).toBe(true);
    expect(registrationsAreOpen(REGISTRATION_NOT_OPEN_YET)).toBe(false);
    expect(registrationsAreOpen(REGISTRATION_CLOSED_BY_ADMIN)).toBe(false);
    expect(registrationsAreOpen(REGISTRATION_CLOSED_AT_KICKOFF)).toBe(false);
    expect(registrationsAreOpen(REGISTRATION_UNSCHEDULED)).toBe(false);
  });
});

describe('fillTeamsFromRegistrations', () => {
  const confirmed = (count, from = 1) => Array.from({ length: count }, (unused, index) => ({
    PlayerID: `p${from + index}`,
    Name: `player${from + index}`,
    Position: from + index,
    IsWaiting: false
  }));

  const empty = () => [[], []];
  const names = (roster) => roster.map(player => player.Name);

  it('splits an even confirmed roster equally, alternating in sign-up order', () => {
    const { rosters, addedCount, skippedCount } = fillTeamsFromRegistrations(confirmed(6), empty());

    expect(addedCount).toBe(6);
    expect(skippedCount).toBe(0);
    // Position 1, 3, 5 to the first team; 2, 4, 6 to the second.
    expect(names(rosters[0])).toEqual(['player1', 'player3', 'player5']);
    expect(names(rosters[1])).toEqual(['player2', 'player4', 'player6']);
  });

  it('puts the extra player on the first team for an odd roster', () => {
    const { rosters } = fillTeamsFromRegistrations(confirmed(5), empty());

    expect(rosters[0]).toHaveLength(3);
    expect(rosters[1]).toHaveLength(2);
    expect(names(rosters[0])).toEqual(['player1', 'player3', 'player5']);
  });

  it('places nobody from the waiting list — they did not get a place', () => {
    const registrations = [
      ...confirmed(2),
      { PlayerID: 'p3', Name: 'reserve-a', Position: 3, IsWaiting: true },
      { PlayerID: 'p4', Name: 'reserve-b', Position: 4, IsWaiting: true }
    ];

    const { rosters, addedCount } = fillTeamsFromRegistrations(registrations, empty());

    expect(addedCount).toBe(2);
    expect([...names(rosters[0]), ...names(rosters[1])]).toEqual(['player1', 'player2']);
  });

  it('never duplicates a player already in a team, and reports them as skipped', () => {
    const rostersIn = [
      [{ ID: 'p1', Name: 'player1', GoalNumber: 3 }],
      [{ ID: 'p2', Name: 'player2', GoalNumber: 0 }]
    ];

    const { rosters, addedCount, skippedCount } = fillTeamsFromRegistrations(confirmed(4), rostersIn);

    expect(addedCount).toBe(2);
    expect(skippedCount).toBe(2);
    expect(names(rosters[0])).toEqual(['player1', 'player3']);
    expect(names(rosters[1])).toEqual(['player2', 'player4']);
    // And the goals already recorded against the player who was already there
    // survive untouched — this action must never cost an admin their edits.
    expect(rosters[0][0].GoalNumber).toBe(3);
  });

  it('matches an already-placed player on the name when ids do not line up', () => {
    // A roster entry typed in by hand can hold a player whose id we were not
    // given; the duplicate check still has to catch them.
    const rostersIn = [[{ Name: 'PLAYER1' }], []];

    const { addedCount, skippedCount } = fillTeamsFromRegistrations(confirmed(2), rostersIn);

    expect(addedCount).toBe(1);
    expect(skippedCount).toBe(1);
  });

  it('sends the newcomers to the smaller side rather than keeping an existing gap', () => {
    // An admin who had already put three players on one team gets the sides
    // evened out, not handed back a 6 v 3.
    const rostersIn = [
      [
        { ID: 'a1', Name: 'already-a', GoalNumber: 0 },
        { ID: 'a2', Name: 'already-b', GoalNumber: 0 },
        { ID: 'a3', Name: 'already-c', GoalNumber: 0 }
      ],
      []
    ];

    const { rosters, addedCount, skippedCount } = fillTeamsFromRegistrations(confirmed(5), rostersIn);

    expect(addedCount).toBe(5);
    expect(skippedCount).toBe(0);
    expect(rosters[0]).toHaveLength(4);
    expect(rosters[1]).toHaveLength(4);
    // The first three sign-ups fill the empty side until it draws level, and
    // only then does the alternation resume (tie to the first team).
    expect(names(rosters[1])).toEqual(['player1', 'player2', 'player3', 'player5']);
    expect(names(rosters[0])).toEqual(['already-a', 'already-b', 'already-c', 'player4']);
  });

  it('is a no-op on an empty sign-up list, leaving the rosters as they were', () => {
    const rostersIn = [[{ ID: 'p1', Name: 'player1', GoalNumber: 1 }], []];

    const { rosters, addedCount, skippedCount } = fillTeamsFromRegistrations([], rostersIn);

    expect(addedCount).toBe(0);
    expect(skippedCount).toBe(0);
    expect(rosters[0]).toEqual([{ ID: 'p1', Name: 'player1', GoalNumber: 1 }]);
    expect(rosters[1]).toEqual([]);
  });

  it('is a no-op on a list that is entirely waiting', () => {
    const registrations = [{ PlayerID: 'p1', Name: 'reserve', Position: 1, IsWaiting: true }];

    const { rosters, addedCount } = fillTeamsFromRegistrations(registrations, empty());

    expect(addedCount).toBe(0);
    expect(rosters).toEqual([[], []]);
  });

  it('starts every newly placed player at 0 goals, like the Add Player path', () => {
    const { rosters } = fillTeamsFromRegistrations(confirmed(2), empty());

    expect(rosters[0][0]).toEqual({ ID: 'p1', Name: 'player1', GoalNumber: 0 });
    expect(rosters[1][0]).toEqual({ ID: 'p2', Name: 'player2', GoalNumber: 0 });
  });

  it('never mutates the rosters it was given', () => {
    const rostersIn = [[], []];

    const { rosters } = fillTeamsFromRegistrations(confirmed(2), rostersIn);

    expect(rostersIn[0]).toHaveLength(0);
    expect(rostersIn[1]).toHaveLength(0);
    expect(rosters[0]).toHaveLength(1);
  });

  it('alternates on Position, not on the order the array happens to be in', () => {
    const shuffled = [confirmed(1, 3)[0], confirmed(1, 1)[0], confirmed(1, 2)[0]];

    const { rosters } = fillTeamsFromRegistrations(shuffled, empty());

    expect(names(rosters[0])).toEqual(['player1', 'player3']);
    expect(names(rosters[1])).toEqual(['player2']);
  });

  it('tolerates a null list and null rosters rather than throwing', () => {
    expect(fillTeamsFromRegistrations(null, null).addedCount).toBe(0);
    expect(fillTeamsFromRegistrations(undefined, empty()).addedCount).toBe(0);
  });
});
