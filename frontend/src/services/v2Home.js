import { parseCalendarDay } from './datetime';
import { deriveRegistrationState, REGISTRATION_NOT_OPEN_YET, REGISTRATION_OPEN } from './matchRegistration';
import { isMotmVotingOpen, motmVotingDeadlineMs } from './motmVoting';

// What the design v2 calls the Man of the Match award, everywhere it shows
// it (the code keeps the motm names, and v1 keeps "Man of the Match" and
// "MOTM"). One place, so the name can change here alone: the settings
// label (notificationLabels.js) imports it. public/sw.js, outside the
// bundle, keeps a copy (MOTM_TEXT.name) that tests/unit/public/sw.spec.js
// pins to this one.
export const STAR_OF_THE_MATCH = 'Star of the Match';
// The short label, where space is tight: the Rankings tab and the profile
// stat. Sentences keep the full name.
export const STARS = 'Stars';
export const VOTE_OPEN_LINE = `${STAR_OF_THE_MATCH} vote open`;

const OLD_AWARD_NAME = /\bm[ae]n[\s_-]+of[\s_-]+the[\s_-]+match\b/gi;
const OLD_AWARD_LEFT = /\bmotm\b|\bm[ae]n[\s_-]+of[\s_-]+the[\s_-]+match\b/i;
const VOTE_ERROR_FALLBACK = `Error updating your ${STAR_OF_THE_MATCH} vote.`;

// The toast text after a failed vote. The backend's own reason is kept, but
// its messages say "man of the match" (shared with v1, so not changed
// there): that name becomes STAR_OF_THE_MATCH, and a reason that would
// still show the old name in any other form (MOTM…) gives way to the
// generic message, as does no reason at all. The reason is shown as a
// sentence: capital first letter, full stop at the end.
export function voteErrorMessage(error) {
  const reason = error?.response?.data?.error;
  if (typeof reason !== 'string' || !reason.trim()) return VOTE_ERROR_FALLBACK;
  const renamed = reason.trim().replace(OLD_AWARD_NAME, STAR_OF_THE_MATCH);
  if (OLD_AWARD_LEFT.test(renamed)) return VOTE_ERROR_FALLBACK;
  // As a sentence: a capital first letter and a closing full stop.
  const sentence = renamed.charAt(0).toUpperCase() + renamed.slice(1);
  return /[.!?]$/.test(sentence) ? sentence : `${sentence}.`;
}

const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

const completedAt = (match) => {
  const day = parseCalendarDay(match && match.Date);
  if (!day) return NaN;
  return new Date(day.getFullYear(), day.getMonth(), day.getDate() + 1).getTime();
};

// A match is over once its score is recorded (ResultRecordedAt, set by the
// server on the first save after kick-off), or else from the day after it.
export const isMatchOver = (match, nowMs) => {
  if (match && match.ResultRecordedAt) return true;
  const at = completedAt(match);
  return !Number.isNaN(at) && nowMs >= at;
};

// The home hero is the soonest match that is not over yet. Everything else
// still ahead goes under "Later", and the matches that are over, newest
// first, under "Latest matches".
export function splitHomeMatches(matches, nowMs) {
  const list = Array.isArray(matches) ? matches : [];
  const upcoming = [];
  const recent = [];
  list.forEach((match) => {
    if (isMatchOver(match, nowMs)) recent.push(match);
    else upcoming.push(match);
  });
  const byKickoff = (match) => Date.parse(match.ScheduledAt || '') || Date.parse(match.Date) || 0;
  upcoming.sort((a, b) => byKickoff(a) - byKickoff(b));
  recent.sort((a, b) => byKickoff(b) - byKickoff(a));
  return {
    next: upcoming[0] || null,
    later: upcoming.slice(1),
    recent,
  };
}

const hasKickoff = (value) => Boolean(value) && !/^\d{4}-\d{2}-\d{2}$/.test(String(value));

function localParts(value) {
  const at = new Date(value);
  if (Number.isNaN(at.getTime())) return null;
  // No timeZone option, on purpose: a kick-off is one instant shown in the
  // viewer's own zone, the same rule as formatDateTimeShort in datetime.js.
  return new Intl.DateTimeFormat('en-US', {
    weekday: 'long',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(at);
}

export function heroDateLabel(date) {
  const day = parseCalendarDay(date);
  if (!day) return '';
  return `${WEEKDAYS[day.getDay()]} ${day.getDate()} ${MONTHS[day.getMonth()]}`;
}

export function rowDateLabel(date) {
  const day = parseCalendarDay(date);
  if (!day) return '';
  return `${String(day.getDate()).padStart(2, '0')} ${MONTHS[day.getMonth()]}`;
}

// « 4 Sep »: the day as read out in a Latest matches row's name.
export function dayMonthLabel(date) {
  const day = parseCalendarDay(date);
  if (!day) return '';
  return `${day.getDate()} ${MONTHS[day.getMonth()]}`;
}

// A calendar day with no kick-off returns nothing, so a played match never
// prints 00:00. The clock is the viewer's local zone (see localParts).
export function kickoffLabel(scheduledAt) {
  if (!hasKickoff(scheduledAt)) return '';
  const parts = localParts(scheduledAt);
  if (!parts) return '';
  const hour = parts.find((part) => part.type === 'hour');
  const minute = parts.find((part) => part.type === 'minute');
  if (!hour || !minute) return '';
  return `${hour.value}:${minute.value}`;
}

export function weekdayTimeLabel(scheduledAt) {
  const time = kickoffLabel(scheduledAt);
  if (!time) return '';
  const parts = localParts(scheduledAt);
  const weekday = parts && parts.find((part) => part.type === 'weekday');
  return weekday ? `${weekday.value} ${time}` : time;
}

export function registrationMeta(match, nowMs) {
  const state = deriveRegistrationState(match, nowMs);
  if (state === REGISTRATION_OPEN) return 'Sign-ups open';
  if (state === REGISTRATION_NOT_OPEN_YET) return 'Sign-ups not open yet';
  if (state === 'closed-by-admin' || state === 'closed-at-kickoff') return 'Sign-ups closed';
  return '';
}

export function placesLeft(match, confirmedCount) {
  const cap = Number(match && match.MaxPlayers) || 0;
  if (!cap) return 0;
  return Math.max(0, cap - confirmedCount);
}

// When the vote closes, as read in « vote open until … ». A scheduled match
// closes 24 hours after kick-off, at a clock time in the viewer's zone
// (« 27 Nov 19:45 »); one without a kick-off is open through the day after
// its Date (« 4 Sep midnight »). See motmVotingDeadlineMs.
export function voteDeadlineLabel(match) {
  const deadline = motmVotingDeadlineMs(match);
  if (Number.isNaN(deadline)) return '';
  if (hasKickoff(match.ScheduledAt)) {
    const at = new Date(deadline);
    const time = `${String(at.getHours()).padStart(2, '0')}:${String(at.getMinutes()).padStart(2, '0')}`;
    return `${at.getDate()} ${MONTHS[at.getMonth()]} ${time}`;
  }
  const last = new Date(deadline - 1);
  return `${last.getDate()} ${MONTHS[last.getMonth()]} midnight`;
}

export function motmWinners(votes) {
  const rows = (votes && votes.Tally) || [];
  const max = rows.reduce((best, row) => Math.max(best, Number(row.Votes) || 0), 0);
  if (max <= 0) return [];
  return rows.filter((row) => Number(row.Votes) === max);
}

// The home's Star of the Match line: only for a match that is over and whose vote is
// still open. A closed vote, or another match's, says nothing there.
export function homeVoteLine(match, nowMs) {
  if (!match || !isMatchOver(match, nowMs)) return null;
  return isMotmVotingOpen(match, nowMs) ? `${STAR_OF_THE_MATCH} vote open · Vote in match details` : null;
}

// A Latest matches row's grey sub-line: VOTE_OPEN_LINE (« Star of the Match
// vote open ») while that match's vote is still open, in place of its
// weekday and kick-off time (never both); else that time, or nothing.
export function rowSubLine(match, nowMs) {
  if (!match) return '';
  if (isMatchOver(match, nowMs) && isMotmVotingOpen(match, nowMs)) return VOTE_OPEN_LINE;
  return weekdayTimeLabel(match.ScheduledAt);
}

export function matchVoteCopy(match, votes, nowMs) {
  if (!match) return '';
  if (isMotmVotingOpen(match, nowMs)) {
    const until = voteDeadlineLabel(match);
    return until
      ? `${STAR_OF_THE_MATCH} vote open until ${until} · tap a player to change`
      : `${STAR_OF_THE_MATCH} vote open · tap a player to change`;
  }
  const winners = motmWinners(votes);
  if (!winners.length) return `${STAR_OF_THE_MATCH} vote closed · no votes, no winner`;
  if (winners.length > 1) {
    const who = winners.length === 2 ? 'both' : 'all of them';
    return `${STAR_OF_THE_MATCH} vote closed · tie, ${who} receive the award`;
  }
  return `${STAR_OF_THE_MATCH} vote closed`;
}

export function goalLine(player) {
  const goals = Number(player.GoalNumber || player.GoalsScored || 0);
  const own = Number(player.OwnGoals || 0);
  const parts = [];
  if (goals === 1) parts.push('1 goal');
  else if (goals > 1) parts.push(`${goals} goals`);
  if (own === 1) parts.push('1 OG');
  else if (own > 1) parts.push(`${own} OG`);
  return parts.join(' · ');
}

// The name of the team the player is on once the hero match's teams are
// set; null when they aren't in either.
export function myTeam(match, playerId) {
  const teams = (match && match.Teams) || [];
  if (!playerId || teams.length !== 2) return null;
  const index = teams.findIndex((team) => (team.Players || []).some((player) => player.ID === playerId));
  if (index === -1) return null;
  return teams[index].Name;
}

// Where a new sign-up would land on a full list: right after the players
// already waiting.
export function nextWaitingPosition(registrations) {
  return (registrations || []).filter((entry) => entry.IsWaiting).length + 1;
}
