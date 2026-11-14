import { parseCalendarDay } from './datetime';
import { deriveRegistrationState, REGISTRATION_NOT_OPEN_YET, REGISTRATION_OPEN } from './matchRegistration';
import { isMotmVotingOpen } from './motmVoting';

const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

const completedAt = (match) => {
  const day = parseCalendarDay(match && match.Date);
  if (!day) return NaN;
  return new Date(day.getFullYear(), day.getMonth(), day.getDate() + 1).getTime();
};

export const isMatchOver = (match, nowMs) => {
  const at = completedAt(match);
  return !Number.isNaN(at) && nowMs >= at;
};

// The home hero is the soonest match that is not over yet. Everything else
// still ahead goes under "Later". When nothing is ahead, the hero is the
// most recent match that is over.
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

// Last calendar day a vote can still be cast: the match date plus one,
// closing at midnight as that day ends.
export function voteDeadlineLabel(match) {
  const day = parseCalendarDay(match && match.Date);
  if (!day) return '';
  const last = new Date(day.getFullYear(), day.getMonth(), day.getDate() + 1);
  return `${last.getDate()} ${MONTHS[last.getMonth()]}`;
}

export function motmWinners(votes) {
  const rows = (votes && votes.Tally) || [];
  const max = rows.reduce((best, row) => Math.max(best, Number(row.Votes) || 0), 0);
  if (max <= 0) return [];
  return rows.filter((row) => Number(row.Votes) === max);
}

export function homeVoteLine(match, votes, nowMs) {
  if (!match || !isMatchOver(match, nowMs)) return null;
  if (isMotmVotingOpen(match, nowMs)) {
    return 'MOTM vote open · Vote in match details';
  }
  const winners = motmWinners(votes);
  if (!winners.length) return null;
  return `MOTM: ${winners.map((row) => row.Name).join(', ')}`;
}

export function matchVoteCopy(match, votes, nowMs) {
  if (!match) return '';
  if (isMotmVotingOpen(match, nowMs)) {
    const until = voteDeadlineLabel(match);
    return until
      ? `MOTM vote open until ${until} midnight · tap a player to change`
      : 'MOTM vote open · tap a player to change';
  }
  const winners = motmWinners(votes);
  if (!winners.length) return 'MOTM vote closed · no votes, no winner';
  if (winners.length > 1) {
    const who = winners.length === 2 ? 'both' : 'all of them';
    return `MOTM vote closed · tie, ${who} receive the award`;
  }
  return 'MOTM vote closed';
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
