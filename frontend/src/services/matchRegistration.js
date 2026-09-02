// Sign-up state of a scheduled match, derived from its timestamps.
//
// This is a pure function of a match's JSON plus a "now" the caller passes in,
// deliberately kept out of the components: it is the piece of this feature most
// easily got wrong (four states, two of which look identical to the user but
// aren't to the backend), and testing it through a mounted component would mean
// re-mounting once per state.
//
// It mirrors MatchRegistrationService.ensureRegistrationWindowOpen on the Go
// side — the backend stays the authority, and every action here is re-checked
// there, but a UI that only found out by getting a 409 would show a Participate
// button that always fails and, worse, a Withdraw button that has silently
// stopped working (withdrawing is gated on the *same* window as signing up, so
// closing the list refuses both).

// A match is scheduled iff it carries a kick-off. Every other scheduling field
// is absent on an unscheduled match (they are all `omitempty` on
// models.MatchWithDetails), so this one check is the whole test.
export const isScheduledMatch = (match) => Boolean(match && match.ScheduledAt);

export const REGISTRATION_UNSCHEDULED = 'unscheduled';
export const REGISTRATION_NOT_OPEN_YET = 'not-open-yet';
export const REGISTRATION_OPEN = 'open';
export const REGISTRATION_CLOSED_BY_ADMIN = 'closed-by-admin';
export const REGISTRATION_CLOSED_AT_KICKOFF = 'closed-at-kickoff';

// `nowMs` is a plain epoch-milliseconds number rather than being read from
// Date.now() in here, so the caller decides when "now" is — which is what makes
// this testable without faking timers, and what lets a view sample the clock
// exactly once (see the callers: there is no polling timer anywhere in this
// feature).
//
// The two "closed" states are separate because only one of them is reversible:
// an admin can reopen a list they closed, but nobody can un-pass kick-off.
export const deriveRegistrationState = (match, nowMs) => {
  if (!isScheduledMatch(match)) {
    return REGISTRATION_UNSCHEDULED;
  }

  // An explicit close wins over everything else, including "hasn't opened yet".
  // The backend checks the window in the opposite order (it reports "not open
  // yet" first), but that ordering only decides which *error message* a doomed
  // request gets; here it decides what the user is told, and "sign-ups open on
  // Friday" would be a promise the closed flag has already broken.
  if (match.RegistrationsClosedAt) {
    return REGISTRATION_CLOSED_BY_ADMIN;
  }

  // Kick-off is a hard backstop: past it, the backend refuses both signing up
  // and withdrawing even if no admin ever closed the list.
  const kickoff = Date.parse(match.ScheduledAt);
  if (!Number.isNaN(kickoff) && nowMs >= kickoff) {
    return REGISTRATION_CLOSED_AT_KICKOFF;
  }

  // A missing or unparseable opening time can't happen through the API — the
  // backend treats the three scheduling fields as all-or-nothing — so it is
  // treated as "no lower bound" rather than as an error state to render.
  const opensAt = Date.parse(match.RegistrationOpensAt);
  if (!Number.isNaN(opensAt) && nowMs < opensAt) {
    return REGISTRATION_NOT_OPEN_YET;
  }

  return REGISTRATION_OPEN;
};

// Both Participate and Withdraw hang off this single predicate, because the
// backend gates both on the same window.
export const registrationsAreOpen = (state) => state === REGISTRATION_OPEN;

// --- Composing the teams from the sign-up list --------------------------------
//
// The admin flow the product asked for is: close sign-ups, *then* compose the
// teams. This is the mechanical half of that — a plain alternating split the
// admin then adjusts by hand. It is explicitly NOT balanced allocation (by
// historical points or goals): that was considered and rejected.
//
// It lives here, as a pure function, for the same reason deriveRegistrationState
// does: the edges (odd roster, waiting list, a player already placed, an empty
// list) are what actually break, and testing them through a mounted component
// would mean one mount per case.
//
// It also does no I/O and knows nothing about the API: the caller assigns the
// returned rosters into its local match state and leaves the existing "Save
// Changes" button to persist them through the usual PUT /matches/:id diff. There
// is no endpoint for this and there should not be one.

// Identity is checked on the id first and the name second. Both are needed: a
// registration entry carries PlayerID while a team roster entry carries ID (two
// different Go structs), and a roster that was typed in by hand can hold a
// player whose id is absent from what we were given. Names are compared
// case-insensitively, matching isPlayerInAnyTeam in MatchDetails.vue.
const samePlayer = (rosterEntry, registration) => {
  if (!rosterEntry || !registration) return false;

  // Two ids present is a decisive answer either way. Falling through to the
  // name comparison here would be a real bug rather than a nicety: this app
  // deliberately lets two players share a display name — Player.Name carries
  // no unique index and SignupNewPlayer checks no collision, so two "Marco"s
  // can join one group through the same invite code — and the second one would
  // then be silently left out of the teams, counted only as "skipped".
  if (rosterEntry.ID && registration.PlayerID) {
    return rosterEntry.ID === registration.PlayerID;
  }

  // Only when an id is missing on one side is the name all there is to go on.
  const rosterName = (rosterEntry.Name || '').toLowerCase();
  const registrationName = (registration.Name || '').toLowerCase();
  return Boolean(rosterName) && rosterName === registrationName;
};

// `registrations` is the list exactly as GET /matches/:id/registrations sent it;
// `currentRosters` is an array of the two teams' current player arrays, in team
// order. Returns the new rosters (fresh arrays — nothing is mutated) plus what
// happened, which the caller turns into its confirmation message.
//
// Rules, all four of them product decisions rather than implementation detail:
//   1. Only the CONFIRMED entries are placed. The waiting list is by definition
//      the players who did not get a place; putting them in a team would
//      contradict the cap they fell outside of.
//   2. They are placed in sign-up order (Position), each going to whichever
//      team is smaller at that point (ties to the first). On the usual empty
//      match that is a plain alternation — an even roster splits equally, an
//      odd one leaves the extra player on the first team — and on a match an
//      admin has already partly filled it closes the gap rather than keeping
//      it.
//   3. A player already in EITHER team is left exactly where they are and never
//      placed again — so this can be run on a partly-filled match without
//      duplicating anyone and without discarding an admin's manual work
//      (including goals already recorded against them). It is deliberately
//      additive, not a replacement; see the UI copy, which says so.
//   4. A newly placed player starts at 0 goals, like the "Add Player" path.
export const fillTeamsFromRegistrations = (registrations, currentRosters) => {
  const rosters = (Array.isArray(currentRosters) ? currentRosters : [])
    .map(roster => (Array.isArray(roster) ? [...roster] : []));

  const entries = Array.isArray(registrations) ? registrations : [];
  // Sorted on Position rather than trusting array order. The server already
  // sends them in that order, so in practice this is a no-op; doing it here is
  // what makes the alternation a function of Position, as specified, instead of
  // a function of however the caller happened to hold the list. The sort is
  // stable, so entries with an equal or absent Position keep their order.
  const confirmed = entries
    .filter(entry => entry && !entry.IsWaiting)
    .sort((a, b) => (a.Position || 0) - (b.Position || 0));

  // Fewer than two teams is not a shape this component ever renders (a match
  // always has exactly two), but returning the rosters untouched beats throwing
  // on the way to a confirmation dialog.
  if (rosters.length < 2) {
    return { rosters, addedCount: 0, skippedCount: 0, placed: [] };
  }

  const alreadyPlaced = (registration) =>
    rosters.some(roster => roster.some(player => samePlayer(player, registration)));

  const placed = [];
  let skippedCount = 0;

  // Each newcomer goes to whichever team is currently smaller, ties to the
  // first — which on two empty rosters is exactly the alternation described
  // above (1st, 3rd, 5th to the first team), and on a partly-filled match
  // evens the sides out instead of preserving the gap. A fixed alternation
  // would take a match an admin had already put 3 players into on one side and
  // hand it back at 8 v 5, for no reason a reader could guess.
  const smallerTeamIndex = () => (rosters[0].length <= rosters[1].length ? 0 : 1);

  confirmed.forEach(entry => {
    if (alreadyPlaced(entry)) {
      skippedCount += 1;
      return;
    }
    const teamIndex = smallerTeamIndex();
    rosters[teamIndex].push({ ID: entry.PlayerID, Name: entry.Name, GoalNumber: 0 });
    placed.push({ teamIndex, name: entry.Name });
  });

  return { rosters, addedCount: placed.length, skippedCount, placed };
};
