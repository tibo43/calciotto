// The Man of the Match voting window, mirrored from the Go side
// (services.MotmVotingDeadline / VotingWindowError,
// app/internal/services/matchVotes.go).
//
// A scheduled match (one with a kick-off, ScheduledAt) closes 24 hours after
// kick-off: a Thursday 19:45 kick-off closes on Friday at 19:45. A match
// created without a kick-off (recorded after the fact) keeps the older
// Date-based rule: open through the day after its Date, closing at the stroke
// of Date+2. The backend stays the authority and re-checks this on every
// call; this module is what lets a star be greyed out *before* a 409.
//
// Reading the tally (GET /matches/:id/votes) is never gated on this — only
// casting, changing, or removing a vote is.

import { parseCalendarDay } from './datetime';

export const MOTM_VOTING_WINDOW_MS = 24 * 60 * 60 * 1000;

// A Match.Date alone (YYYY-MM-DD) is a calendar day, not a kick-off.
const kickoffMs = (match) => {
  const value = match && match.ScheduledAt;
  if (!value || /^\d{4}-\d{2}-\d{2}$/.test(String(value))) return NaN;
  return Date.parse(value);
};

// The instant voting closes, in epoch milliseconds: kick-off + 24h for a
// scheduled match, else local midnight two days after the match's own Date.
// NaN when neither can be read. The Date-based deadline is constructed from
// the parsed date's own year/month/day (not by adding 48 hours) so a DST
// transition in between doesn't shift it by an hour.
export const motmVotingDeadlineMs = (match) => {
  const kickoff = kickoffMs(match);
  if (!Number.isNaN(kickoff)) return kickoff + MOTM_VOTING_WINDOW_MS;
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
