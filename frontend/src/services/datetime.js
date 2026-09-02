// Local-offset RFC3339 formatting for the scheduling fields of POST /matches.
//
// The backend derives a scheduled match's calendar day (and therefore its
// season) from `scheduled_at` **in the offset the client sent**, so the string
// has to carry the browser's own UTC offset.
//
// This is why nothing here goes through `Date.prototype.toISOString()`: that
// always returns UTC with a `Z` suffix, which silently moves a late-evening
// kick-off to the previous day for anyone east of Greenwich — 00:30 in Paris
// becomes `22:30Z` on the day before, and the match lands on the wrong date.
// `<input type="datetime-local">` has the mirror-image problem: its value
// (`YYYY-MM-DDTHH:MM`) carries no offset at all, so it goes through here too.

// Turns a `Date.prototype.getTimezoneOffset()` value into an RFC3339 offset.
//
// getTimezoneOffset() is minutes to *add to local time to get UTC*, i.e. the
// sign is the opposite of the one RFC3339 wants: Paris in summer reports -120
// and must be written `+02:00`. Half-hour and quarter-hour zones (India +05:30,
// Nepal +05:45, Newfoundland -03:30) are why the remainder is kept rather than
// dividing by 60 and dropping it.
export const formatUTCOffset = (offsetMinutes) => {
  const sign = offsetMinutes > 0 ? '-' : '+';
  const total = Math.abs(offsetMinutes);
  const hours = Math.floor(total / 60).toString().padStart(2, '0');
  const minutes = (total % 60).toString().padStart(2, '0');
  return `${sign}${hours}:${minutes}`;
};

// Combines a calendar day (`YYYY-MM-DD`, what the modal's hand-rolled picker
// produces) and a wall-clock time (`HH:MM` or `HH:MM:SS`, what an
// `<input type="time">` produces) into `YYYY-MM-DDTHH:MM:SS±HH:MM`.
//
// The offset is read from a Date built at that exact local instant rather than
// from `new Date()`, so a match scheduled across a DST boundary is stamped with
// the offset in force *then*, not the one in force today.
//
// Returns '' for missing/malformed input — callers validate before submitting,
// and an empty string is never a valid scheduling field.
export const toLocalRFC3339 = (day, time) => {
  if (!day || !time) return '';

  const dayParts = day.split('-').map(Number);
  const timeParts = time.split(':').map(Number);
  if (dayParts.length !== 3 || timeParts.length < 2) return '';
  if (dayParts.some(Number.isNaN) || timeParts.some(Number.isNaN)) return '';

  const [year, month, dayOfMonth] = dayParts;
  const [hours, minutes] = timeParts;
  const seconds = timeParts.length > 2 ? timeParts[2] : 0;

  const local = new Date(year, month - 1, dayOfMonth, hours, minutes, seconds);
  if (Number.isNaN(local.getTime())) return '';

  const pad = (value) => value.toString().padStart(2, '0');
  const datePart = `${year}-${pad(month)}-${pad(dayOfMonth)}`;
  const timePart = `${pad(hours)}:${pad(minutes)}:${pad(seconds)}`;

  return `${datePart}T${timePart}${formatUTCOffset(local.getTimezoneOffset())}`;
};

// Same thing for a raw `<input type="datetime-local">` value, which packs the
// day and the time into one offset-less string.
export const dateTimeLocalToRFC3339 = (value) => {
  if (!value) return '';
  const [day, time] = value.split('T');
  return toLocalRFC3339(day, time);
};

// ---------------------------------------------------------------------------
// Display formatting for the timestamps the backend hands back.
//
// The mirror image of the two functions above: those turn what the user typed
// into an RFC3339 string for the backend, these turn an RFC3339 string from the
// backend into something readable. Both directions live here so there is one
// date path in the app rather than a hand-rolled `toLocaleString` call in each
// component that happens to need one.
//
// Rendering is deliberately in the *browser's* zone (no `timeZone` option), not
// in the zone the timestamp was created in: a kick-off is a single instant, and
// a player travelling or living elsewhere wants to know when it happens for
// them. That is also why nothing here re-reads the offset the string carries —
// `Date` has already resolved it to an instant by then.

// Accepts an RFC3339 string, a Date, or anything Date can parse; returns null
// for empty/unparseable input so the callers below can degrade to '' instead of
// rendering "Invalid Date" at the user.
const toDate = (value) => {
  if (!value) return null;
  const date = value instanceof Date ? value : new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
};

// Full kick-off / sign-ups-open line, e.g. "Sun, Sep 6, 2026, 8:30 PM".
// Weekday included on purpose: a calciotto is a recurring weekly fixture, so
// "which day of the week" is the first thing a player checks.
export const formatDateTimeForDisplay = (value) => {
  const date = toDate(value);
  if (!date) return '';
  return date.toLocaleString('en-US', {
    weekday: 'short',
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
};

// The compact variant for a match card in the carousel, e.g.
// "Sep 6, 2026, 8:30 PM" — no weekday, since the card has room for one line.
export const formatDateTimeShort = (value) => {
  const date = toDate(value);
  if (!date) return '';
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
};

// ---------------------------------------------------------------------------
// Calendar days (`Match.Date`, the `YYYY-MM-DD` the Go `Date` type marshals to)
// — a different kind of value from the timestamps above, with a trap of its own.
//
// `new Date('2026-09-06')` is specified to parse a date-*only* string as UTC
// midnight, so formatting the result in the browser's zone renders the
// *previous* day for anyone west of Greenwich: a match played on Sep 6 shows up
// as Sep 5 in New York. (A date-*time* string with no offset parses as local
// instead — the inconsistency is in the language, not in the caller.)
//
// The reason the fix is "parse it locally" rather than "render it in UTC" is
// that a calendar day is not an instant. `Match.Date` is the day the group
// played, the same day for every viewer, which is the exact opposite of
// ScheduledAt above: that one is a single instant deliberately shown in each
// viewer's own zone, this one is a label that must not move at all.

// Returns a Date at local midnight on the day `value` names, or null if it
// names none. A full timestamp is handed to Date unchanged — those forms carry
// a time and already parse as local, so the trap above doesn't apply to them.
export const parseCalendarDay = (value) => {
  if (!value) return null;
  if (value instanceof Date) return Number.isNaN(value.getTime()) ? null : value;
  if (typeof value !== 'string') return null;

  const parts = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value.trim());
  if (parts) {
    const [, year, month, day] = parts.map(Number);
    const date = new Date(year, month - 1, day);
    // Rejected rather than accepted: Date rolls an out-of-range day over (Feb
    // 31st becomes Mar 3rd), which would display a day the API never sent.
    if (date.getMonth() !== month - 1 || date.getDate() !== day) return null;
    return date;
  }

  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? null : parsed;
};

// Unparseable input echoes the original string instead of returning '' the way
// the timestamp formatters above do. That is deliberate and the difference is
// the field, not an oversight: a kick-off line can be dropped from a card
// harmlessly, whereas a match's date is the card's identity — showing the raw
// value at least stays diagnosable where a blank would just look broken.
const formatCalendarDayWith = (value, options) => {
  const date = parseCalendarDay(value);
  if (!date) return typeof value === 'string' ? value : '';
  return date.toLocaleDateString('en-US', options);
};

// "Sun, Sep 6, 2026" — the default: weekday first, since a calciotto is a
// recurring weekly fixture.
export const formatCalendarDay = (value) => formatCalendarDayWith(value, {
  weekday: 'short',
  year: 'numeric',
  month: 'short',
  day: 'numeric',
});

// "Sunday, September 6, 2026" — for the one place with room for it, the match
// detail page's header.
export const formatCalendarDayLong = (value) => formatCalendarDayWith(value, {
  weekday: 'long',
  year: 'numeric',
  month: 'long',
  day: 'numeric',
});

// "Sep 6, 2026" — no weekday, for a match card in the carousel, matching
// formatDateTimeShort's role for a scheduled one.
export const formatCalendarDayShort = (value) => formatCalendarDayWith(value, {
  year: 'numeric',
  month: 'short',
  day: 'numeric',
});
