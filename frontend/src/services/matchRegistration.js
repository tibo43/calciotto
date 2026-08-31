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
