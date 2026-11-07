// The win/draw/loss prediction window, mirrored from the Go side
// (MatchPredictionService.PredictionWindowError,
// app/internal/services/matchPredictions.go).
//
// A prediction can be cast or changed once the match's roster is fully
// composed on both sides, through the instant of kick-off — a match with no
// schedule at all never opens a window, and neither does one whose roster
// isn't composed yet (there is nothing to predict a winner between). The
// backend stays the authority and re-checks this on every call; this module
// is what lets the three pick buttons be greyed out with an explanatory
// title *before* a 409/400, the same reasoning
// matchRegistration.js/deriveRegistrationState already follows for the
// sign-up window.
//
// Reading predictions (GET /matches/:id/predictions) is never gated on
// this — only casting one is.

import { isScheduledMatch } from './matchRegistration';

export const PREDICTIONS_NOT_AVAILABLE = 'not-available';
export const PREDICTIONS_OPEN = 'open';
export const PREDICTIONS_CLOSED = 'closed';

// Predicting a winner needs both sides to actually exist — stricter than
// teamsAreComposed's own "either team has a player" (enough to start
// showing a roster UI), the same distinction
// MatchPredictionService.rosterFullyComposed draws on the Go side.
export const rosterFullyComposed = (match) => Boolean(
  match
  && Array.isArray(match.Teams)
  && match.Teams.length === 2
  && match.Teams.every(team => Array.isArray(team.Players) && team.Players.length > 0)
);

// One of PREDICTIONS_NOT_AVAILABLE/PREDICTIONS_OPEN/PREDICTIONS_CLOSED —
// collapsing the rule into a single state the way
// matchRegistration.js's deriveRegistrationState does for sign-ups, rather
// than a bare boolean, so a caller can render *why* predictions aren't
// available without re-deriving the reason itself.
export const predictionState = (match, nowMs) => {
  if (!isScheduledMatch(match) || !rosterFullyComposed(match)) {
    return PREDICTIONS_NOT_AVAILABLE;
  }
  const kickoffMs = Date.parse(match.ScheduledAt);
  if (!Number.isNaN(kickoffMs) && nowMs >= kickoffMs) {
    return PREDICTIONS_CLOSED;
  }
  return PREDICTIONS_OPEN;
};

export const predictionsAreOpen = (match, nowMs) => predictionState(match, nowMs) === PREDICTIONS_OPEN;
