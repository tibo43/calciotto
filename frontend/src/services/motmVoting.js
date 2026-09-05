// The Man of the Match voting window, mirrored from the Go side
// (MatchVoteService.VotingWindowError, app/internal/services/matchVotes.go).
//
// A vote can be cast, changed, or removed through the end of the calendar
// day *after* the match was played — a match on Date D stays votable through
// D+1, closing at the stroke of D+2. This is deliberately Date-based, not
// tied to ScheduledAt/kick-off or CreatedAt at all: an earlier version
// anchored differently for a scheduled match (kick-off) than an unscheduled
// one (CreatedAt), which fed into a status badge that ended up looking
// inconsistent from one match to the next — see MatchesPanel.vue's
// matchStatus, which uses this exact same Date-based rule for the same
// reason. The backend stays the authority and re-checks this on every call;
// this module is what lets a star be greyed out with an explanatory tooltip
// *before* a 409, the same reasoning MatchRegistrationService/
// deriveRegistrationState already follow for the sign-up window.
//
// Reading the tally (GET /matches/:id/votes) is never gated on this — only
// casting, changing, or removing a vote is.

import { parseCalendarDay } from './datetime';

// The instant voting closes, in epoch milliseconds: local midnight two days
// after the match's own Date. NaN when Date is missing or unparseable —
// which should not happen in practice (every match carries one), but is
// handled explicitly rather than silently coercing into a wrong instant.
// Constructed from the parsed date's own year/month/day (not by adding
// 48 hours in milliseconds) so a DST transition in between doesn't shift the
// deadline by an hour — JS normalizes the day-overflow using the local
// calendar, not fixed-width time arithmetic.
export const motmVotingDeadlineMs = (match) => {
  const date = match && parseCalendarDay(match.Date);
  if (!date) return NaN;
  return new Date(date.getFullYear(), date.getMonth(), date.getDate() + 2).getTime();
};

// Whether a vote can still be cast, changed, or removed at nowMs. An
// unparseable Date fails OPEN rather than closed: this check is a
// defence-in-depth UI convenience only, and the backend refuses the call
// outright the moment this would ever disagree with it — better to offer a
// star that turns out to be a step too late than to hide one that was still
// valid because of a payload this app doesn't recognise.
export const isMotmVotingOpen = (match, nowMs) => {
  const deadline = motmVotingDeadlineMs(match);
  if (Number.isNaN(deadline)) return true;
  return nowMs < deadline;
};
