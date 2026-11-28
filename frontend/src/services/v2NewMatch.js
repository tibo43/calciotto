import { t } from '@/i18n';
import { toLocalRFC3339, dateTimeLocalToRFC3339 } from '@/services/datetime';

// The design v2 "New match" page (NewMatchV2.vue). Only scheduled matches
// are created there: kick-off, when sign-ups open, how many places, and
// whether the admin signs up straight away. The rules mirror the backend's
// (MatchSpec.validate) so the reason shows next to the field instead of
// coming back as a 400; the backend still checks everything.

// A calciotto is 8 a side.
export const DEFAULT_MAX_PLAYERS = 16;

const pad = (value) => String(value).padStart(2, '0');

// `<input type="datetime-local">`'s value for a moment, in the viewer's zone,
// to the minute: what "sign-ups open" starts at (right away).
export function localDateTimeValue(date = new Date()) {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

// Either { field, error } for the first problem, or the arguments for
// api.createMatch: { matchData, scheduling }. Times carry the browser's own
// UTC offset (datetime.js): the backend files the match under the kick-off's
// calendar day in that offset.
export function buildNewMatch({ day, time, opensAt, maxPlayers, registerCreator }, now = new Date()) {
  if (!day) return { field: 'day', error: t('newMatch.errors.day') };
  if (!time) return { field: 'time', error: t('newMatch.errors.time') };
  if (!opensAt) return { field: 'opensAt', error: t('newMatch.errors.opensAt') };

  const places = Number(maxPlayers);
  if (!Number.isInteger(places) || places < 1) {
    return { field: 'maxPlayers', error: t('newMatch.errors.places') };
  }

  const scheduledAt = toLocalRFC3339(day, time);
  if (!scheduledAt) return { field: 'time', error: t('newMatch.errors.checkKickoff') };
  if (Date.parse(scheduledAt) <= now.getTime()) {
    return { field: 'time', error: t('newMatch.errors.kickoffPast') };
  }
  const registrationOpensAt = dateTimeLocalToRFC3339(opensAt);
  if (!registrationOpensAt) return { field: 'opensAt', error: t('newMatch.errors.checkOpensAt') };
  if (Date.parse(registrationOpensAt) >= Date.parse(scheduledAt)) {
    return { field: 'opensAt', error: t('newMatch.errors.opensAfterKickoff') };
  }

  const scheduling = { scheduledAt, registrationOpensAt, maxPlayers: places };
  if (registerCreator) scheduling.registerCreator = true;
  return { matchData: { Date: day }, scheduling };
}
