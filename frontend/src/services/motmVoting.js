// The Man of the Match voting window, mirrored from the Go side
// (MatchVoteService.VotingWindowError, app/internal/services/matchVotes.go).
//
// A vote can be cast, changed, or removed only until 24 hours after "when the
// match happened" — kick-off (ScheduledAt) for a scheduled match, or the
// match row's own CreatedAt for one recorded after the fact. The latter is
// the best available proxy for "when it was played" for a match with no
// kick-off at all, since these are normally logged right after playing — an
// explicit product decision, not a default worth reconsidering here. The
// backend stays the authority and re-checks this on every call; this module
// is what lets a star be greyed out with an explanatory tooltip *before* a
// 409, the same reasoning MatchRegistrationService/deriveRegistrationState
// already follow for the sign-up window.
//
// Reading the tally (GET /matches/:id/votes) is never gated on this — only
// casting, changing, or removing a vote is.

import { isScheduledMatch } from './matchRegistration';

export const MOTM_VOTING_WINDOW_MS = 24 * 60 * 60 * 1000;

// The instant "the match happened", in epoch milliseconds: kick-off for a
// scheduled match, the row's own creation instant otherwise. NaN when neither
// is present or parseable — which should not happen in practice (every match
// this app's backend serves carries a CreatedAt), but is handled explicitly
// rather than silently coercing into a wrong instant.
export const motmVotingAnchorMs = (match) => {
  if (!match) return NaN;
  const raw = isScheduledMatch(match) ? match.ScheduledAt : match.CreatedAt;
  return Date.parse(raw);
};

// Whether a vote can still be cast, changed, or removed at nowMs. An
// unparseable anchor fails OPEN rather than closed: this check is a
// defence-in-depth UI convenience only, and the backend refuses the call
// outright the moment this would ever disagree with it — better to offer a
// star that turns out to be a step too late than to hide one that was still
// valid because of a payload this app doesn't recognise.
export const isMotmVotingOpen = (match, nowMs) => {
  const anchor = motmVotingAnchorMs(match);
  if (Number.isNaN(anchor)) return true;
  return nowMs < anchor + MOTM_VOTING_WINDOW_MS;
};
