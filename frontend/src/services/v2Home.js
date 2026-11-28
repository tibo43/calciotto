import { t } from '@/i18n';
import en from '@/locales/en.json';
import { parseCalendarDay } from './datetime';
import { activeTag, dateLabel, monthName, weekdayName } from './intl';
import { getLocale } from './locale';
import { deriveRegistrationState, REGISTRATION_NOT_OPEN_YET, REGISTRATION_OPEN } from './matchRegistration';
import { isMotmVotingOpen, motmVotingDeadlineMs } from './motmVoting';

// The award's short name, the same in every language (glossary.mvp, never
// translated, never a plural « s »). The code keeps the motm identifiers
// (MotmAwards, tab=motm, motm_announced…). public/sw.js is served as is
// from public/, outside the bundle, so it cannot import this: its
// MOTM_TEXT.name is a copy, and tests/unit/public/sw.spec.js fails if the
// two differ. Every sentence around it is a message (match.vote.*…).
export const MVP = en.glossary.mvp;

// « MVP vote open »: a Latest matches row's sub-line while its vote is open.
export const voteOpenLine = () => t('match.vote.open');

const OLD_AWARD_NAME = /\bm[ae]n[\s_-]+of[\s_-]+the[\s_-]+match\b|\bstars?\s+of\s+the\s+match\b/gi;
const OLD_AWARD_LEFT = /\bmotm\b|\bm[ae]n[\s_-]+of[\s_-]+the[\s_-]+match\b/i;

// The toast text after a failed vote. The backend's own reason is kept; an
// older "man of the match" / "star of the match" wording is rewritten to
// MVP. A reason that would still show the award the old way (MOTM…), or no
// reason at all, gives way to the generic message. The reason is shown as a
// sentence: a capital first letter, full stop at the end.
export function voteErrorMessage(error) {
  const reason = error?.response?.data?.error;
  if (typeof reason !== 'string' || !reason.trim()) return t('match.vote.errorFallback');
  const renamed = reason.trim().replace(OLD_AWARD_NAME, MVP);
  if (OLD_AWARD_LEFT.test(renamed)) return t('match.vote.errorFallback');
  // As a sentence: a capital first letter and a closing full stop.
  const sentence = renamed.charAt(0).toUpperCase() + renamed.slice(1);
  return /[.!?]$/.test(sentence) ? sentence : `${sentence}.`;
}

// Weekday and month names come from Intl in the active language ('short':
// « Sun », « Sep » in English); the order of a label is its date.* message.

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

const WEEKDAY_TIME = { weekday: 'long', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' };

function localParts(value) {
  const at = new Date(value);
  if (Number.isNaN(at.getTime())) return null;
  // No timeZone option, on purpose: a kick-off is one instant shown in the
  // viewer's own zone, the same rule as formatDateTimeShort in datetime.js.
  return new Intl.DateTimeFormat(activeTag(), WEEKDAY_TIME).formatToParts(at);
}

export function heroDateLabel(date) {
  const day = parseCalendarDay(date);
  if (!day) return '';
  return dateLabel(day, { weekday: 'short', day: 'numeric', month: 'short' },
    () => t('date.hero', { weekday: weekdayName(day), day: day.getDate(), month: monthName(day) }));
}

export function rowDateLabel(date) {
  const day = parseCalendarDay(date);
  if (!day) return '';
  return dateLabel(day, { day: '2-digit', month: 'short' },
    () => t('date.row', { day: String(day.getDate()).padStart(2, '0'), month: monthName(day) }));
}

// « 4 Sep »: the day as read out in a Latest matches row's name.
export function dayMonthLabel(date) {
  const day = parseCalendarDay(date);
  if (!day) return '';
  return dateLabel(day, { day: 'numeric', month: 'short' },
    () => t('date.dayMonth', { day: day.getDate(), month: monthName(day) }));
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
  if (getLocale() !== 'en') return dateLabel(scheduledAt, WEEKDAY_TIME);
  const parts = localParts(scheduledAt);
  const weekday = parts && parts.find((part) => part.type === 'weekday');
  return weekday ? t('date.weekdayTime', { weekday: weekday.value, time }) : time;
}

export function registrationMeta(match, nowMs) {
  const state = deriveRegistrationState(match, nowMs);
  if (state === REGISTRATION_OPEN) return t('home.signups.open');
  if (state === REGISTRATION_NOT_OPEN_YET) return t('home.signups.notOpenYet');
  if (state === 'closed-by-admin' || state === 'closed-at-kickoff') return t('home.signups.closed');
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
    return dateLabel(at, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' },
      () => t('date.deadline', { day: at.getDate(), month: monthName(at), time }));
  }
  const last = new Date(deadline - 1);
  const date = dateLabel(last, { day: 'numeric', month: 'short' },
    () => t('date.dayMonth', { day: last.getDate(), month: monthName(last) }));
  return t('date.deadlineMidnight', { date });
}

export function motmWinners(votes) {
  const rows = (votes && votes.Tally) || [];
  const max = rows.reduce((best, row) => Math.max(best, Number(row.Votes) || 0), 0);
  if (max <= 0) return [];
  return rows.filter((row) => Number(row.Votes) === max);
}

// The home's MVP vote line: only for a match that is over and whose vote is
// still open. A closed vote, or another match's, says nothing there.
export function homeVoteLine(match, nowMs) {
  if (!match || !isMatchOver(match, nowMs)) return null;
  return isMotmVotingOpen(match, nowMs) ? t('match.vote.openHome') : null;
}

// A Latest matches row's grey sub-line: voteOpenLine() (« MVP vote open »)
// while that match's vote is still open, in place of its
// weekday and kick-off time (never both); else that time, or nothing.
export function rowSubLine(match, nowMs) {
  if (!match) return '';
  if (isMatchOver(match, nowMs) && isMotmVotingOpen(match, nowMs)) return voteOpenLine();
  return weekdayTimeLabel(match.ScheduledAt);
}

export function matchVoteCopy(match, votes, nowMs) {
  if (!match) return '';
  if (isMotmVotingOpen(match, nowMs)) {
    const until = voteDeadlineLabel(match);
    return until
      ? t('match.vote.openUntil', { until })
      : t('match.vote.openChange');
  }
  const winners = motmWinners(votes);
  if (!winners.length) return t('match.vote.closedNoVotes');
  if (winners.length > 1) {
    return winners.length === 2 ? t('match.vote.closedTieBoth') : t('match.vote.closedTieAll');
  }
  return t('match.vote.closed');
}

export function goalLine(player) {
  const goals = Number(player.GoalNumber || player.GoalsScored || 0);
  const own = Number(player.OwnGoals || 0);
  const parts = [];
  if (goals > 0) parts.push(t('match.goals', goals));
  if (own > 0) parts.push(t('match.ownGoals', own));
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
