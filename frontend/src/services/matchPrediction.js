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

// What a match's card and detail view show, one step further than
// predictionState: once predictions have closed, a match is either still
// waiting for its result or settled. Settlement is the admin saving the
// score after kick-off (Match.ResultRecordedAt, set by
// MatchService.UpdateMatch), mirroring StandingsService.GetPredictionStandings'
// FilterResultRecorded — not the calendar, since until the score is in a
// 0-0 could just mean nobody has entered it yet.
export const PREDICTIONS_AWAITING_RESULT = 'awaiting-result';
export const PREDICTIONS_SETTLED = 'settled';

export const predictionDisplayState = (match, nowMs) => {
  const state = predictionState(match, nowMs);
  if (state !== PREDICTIONS_CLOSED) {
    return state;
  }
  return match.ResultRecordedAt ? PREDICTIONS_SETTLED : PREDICTIONS_AWAITING_RESULT;
};

export const predictionDisplayLabel = (displayState) => {
  switch (displayState) {
    case PREDICTIONS_OPEN:
      return 'Predictions open';
    case PREDICTIONS_AWAITING_RESULT:
      return 'Awaiting result';
    case PREDICTIONS_SETTLED:
      return 'Settled';
    default:
      return 'Not available';
  }
};

// The winning team's id, or null for a draw — read off Team.Score exactly
// the way ComputePredictionStandings does on the Go side.
export const matchWinnerId = (match) => {
  const [teamA, teamB] = match.Teams;
  if (teamA.Score > teamB.Score) {
    return teamA.ID;
  }
  if (teamB.Score > teamA.Score) {
    return teamB.ID;
  }
  return null;
};

// One settled prediction's outcome at the one implicit token staked: a hit
// nets odds - 1, a miss -1 — the same arithmetic as
// ComputePredictionStandings, so a row here always adds up to what the
// leaderboard shows. predictedTeamId is null for a draw; odds is the
// summary's { TeamAOdds, DrawOdds, TeamBOdds }. Returns null when the match
// isn't settled or there are no odds to settle against.
export const settlePrediction = (match, odds, predictedTeamId) => {
  if (!match || !match.ResultRecordedAt || !odds || !rosterFullyComposed(match)) {
    return null;
  }
  const picked = predictedTeamId || null;
  const correct = picked === matchWinnerId(match);
  if (!correct) {
    return { correct, points: -1 };
  }
  let pickedOdds = odds.DrawOdds;
  if (picked === match.Teams[0].ID) {
    pickedOdds = odds.TeamAOdds;
  } else if (picked === match.Teams[1].ID) {
    pickedOdds = odds.TeamBOdds;
  }
  return { correct, points: pickedOdds - 1 };
};
