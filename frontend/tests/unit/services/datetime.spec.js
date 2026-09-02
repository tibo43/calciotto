import {
  formatUTCOffset,
  toLocalRFC3339,
  dateTimeLocalToRFC3339,
  formatDateTimeForDisplay,
  formatDateTimeShort,
  parseCalendarDay,
  formatCalendarDay,
  formatCalendarDayLong,
  formatCalendarDayShort
} from '@/services/datetime';

// The machine's own zone must never decide whether these pass — a test that
// only holds in Europe/Paris is worse than no test at all here.
const withOffset = (offsetMinutes, assertions) => {
  const spy = jest
    .spyOn(Date.prototype, 'getTimezoneOffset')
    .mockReturnValue(offsetMinutes);
  try {
    assertions();
  } finally {
    spy.mockRestore();
  }
};

describe('formatUTCOffset', () => {
  // getTimezoneOffset() is minutes to add to local time to reach UTC, so its
  // sign is the opposite of RFC3339's.
  it('flips the sign of a zone ahead of UTC', () => {
    expect(formatUTCOffset(-120)).toBe('+02:00');
  });

  it('flips the sign of a zone behind UTC', () => {
    expect(formatUTCOffset(240)).toBe('-04:00');
  });

  it('writes UTC itself as +00:00', () => {
    expect(formatUTCOffset(0)).toBe('+00:00');
  });

  it('keeps half-hour and quarter-hour zones intact', () => {
    expect(formatUTCOffset(-330)).toBe('+05:30'); // Asia/Kolkata
    expect(formatUTCOffset(210)).toBe('-03:30'); // America/St_Johns
    expect(formatUTCOffset(-345)).toBe('+05:45'); // Asia/Kathmandu
  });

  it('zero-pads a single-digit hour offset', () => {
    expect(formatUTCOffset(-60)).toBe('+01:00');
  });
});

describe('toLocalRFC3339', () => {
  it('stamps the local offset of a zone ahead of UTC', () => {
    withOffset(-120, () => {
      expect(toLocalRFC3339('2026-09-06', '20:30')).toBe('2026-09-06T20:30:00+02:00');
    });
  });

  it('stamps the local offset of a zone behind UTC', () => {
    withOffset(240, () => {
      expect(toLocalRFC3339('2026-09-06', '20:30')).toBe('2026-09-06T20:30:00-04:00');
    });
  });

  it('stamps UTC as +00:00', () => {
    withOffset(0, () => {
      expect(toLocalRFC3339('2026-09-06', '20:30')).toBe('2026-09-06T20:30:00+00:00');
    });
  });

  it('stamps a half-hour zone', () => {
    withOffset(-330, () => {
      expect(toLocalRFC3339('2026-09-06', '20:30')).toBe('2026-09-06T20:30:00+05:30');
    });
  });

  // The whole point of this helper: toISOString() would return
  // '2026-09-05T22:30:00.000Z' here, and the backend — which derives the
  // match's calendar day (and therefore its season) from this value — would
  // file the match on the 5th instead of the 6th.
  it('keeps a late-evening kick-off on the picked day for a zone ahead of UTC', () => {
    withOffset(-120, () => {
      const value = toLocalRFC3339('2026-09-06', '00:30');
      expect(value).toBe('2026-09-06T00:30:00+02:00');
      expect(value.startsWith('2026-09-06')).toBe(true);
      expect(new Date(value).toISOString()).toBe('2026-09-05T22:30:00.000Z');
    });
  });

  it('zero-pads month, day, hour and minute', () => {
    withOffset(-60, () => {
      expect(toLocalRFC3339('2026-3-5', '9:5')).toBe('2026-03-05T09:05:00+01:00');
    });
  });

  it('keeps seconds when the time input supplies them', () => {
    withOffset(-60, () => {
      expect(toLocalRFC3339('2026-03-05', '09:05:07')).toBe('2026-03-05T09:05:07+01:00');
    });
  });

  it('returns an empty string for missing or malformed input', () => {
    withOffset(-120, () => {
      expect(toLocalRFC3339('', '20:30')).toBe('');
      expect(toLocalRFC3339('2026-09-06', '')).toBe('');
      expect(toLocalRFC3339('not-a-date', '20:30')).toBe('');
      expect(toLocalRFC3339('2026-09-06', 'evening')).toBe('');
      expect(toLocalRFC3339('2026-09', '20:30')).toBe('');
    });
  });
});

describe('dateTimeLocalToRFC3339', () => {
  // <input type="datetime-local"> yields YYYY-MM-DDTHH:MM with no offset at
  // all, so it needs exactly the same treatment.
  it('adds the local offset to an offset-less datetime-local value', () => {
    withOffset(-120, () => {
      expect(dateTimeLocalToRFC3339('2026-09-01T12:00')).toBe('2026-09-01T12:00:00+02:00');
    });
  });

  it('returns an empty string for an empty or partial value', () => {
    withOffset(-120, () => {
      expect(dateTimeLocalToRFC3339('')).toBe('');
      expect(dateTimeLocalToRFC3339('2026-09-01')).toBe('');
    });
  });
});

// The display formatters render in the *browser's* zone, so a test must not
// hand them a fixed absolute instant and assert a fixed wall clock — that only
// holds in one zone. Every case below either builds the Date from local
// components (so the local getters give those components back, whatever the
// zone) or round-trips through toLocalRFC3339, which stamps the machine's own
// offset. Assertions are on substrings rather than the whole string because the
// exact separator between date and time ("," vs " at ") depends on the ICU
// version bundled with Node, not on anything this code controls.
describe('formatDateTimeForDisplay', () => {
  it('renders the local weekday, date and time of a Date built from local components', () => {
    const result = formatDateTimeForDisplay(new Date(2026, 8, 6, 20, 30));

    expect(result).toContain('Sun');
    expect(result).toContain('Sep 6, 2026');
    expect(result).toContain('8:30 PM');
  });

  it('parses an RFC3339 string carrying the local offset to the same instant', () => {
    // toLocalRFC3339 is the producer side of this pair, so the round trip is
    // the real contract: what the create modal sent must come back reading the
    // same way.
    const rfc3339 = toLocalRFC3339('2026-09-06', '20:30');

    expect(formatDateTimeForDisplay(rfc3339)).toBe(formatDateTimeForDisplay(new Date(2026, 8, 6, 20, 30)));
  });

  it('renders a midnight kick-off as 12:00 AM on its own day, not the day before', () => {
    const result = formatDateTimeForDisplay(new Date(2026, 8, 6, 0, 30));

    expect(result).toContain('Sep 6, 2026');
    expect(result).toContain('12:30 AM');
  });

  it('returns an empty string rather than "Invalid Date" for unusable input', () => {
    expect(formatDateTimeForDisplay('')).toBe('');
    expect(formatDateTimeForDisplay(null)).toBe('');
    expect(formatDateTimeForDisplay(undefined)).toBe('');
    expect(formatDateTimeForDisplay('not a date at all')).toBe('');
  });
});

describe('formatDateTimeShort', () => {
  it('drops the weekday but keeps the date and time, for a compact match card', () => {
    const result = formatDateTimeShort(new Date(2026, 8, 6, 20, 30));

    expect(result).toContain('Sep 6, 2026');
    expect(result).toContain('8:30 PM');
    expect(result).not.toContain('Sun');
  });

  it('returns an empty string for unusable input', () => {
    expect(formatDateTimeShort('')).toBe('');
    expect(formatDateTimeShort('nonsense')).toBe('');
  });
});

// The whole point of these: `new Date('2026-09-06')` parses as UTC midnight, so
// the old hand-rolled formatters rendered Sep 5 for anyone west of Greenwich.
// Every assertion below is written to hold in any zone — comparing against a
// locally-constructed Date rather than a hardcoded string wherever the zone
// could otherwise decide the outcome.
describe('parseCalendarDay', () => {
  it('reads YYYY-MM-DD as local midnight, not UTC midnight', () => {
    const date = parseCalendarDay('2026-09-06');
    // Local getters: these are 2026/8/6 in every zone only if the parse was
    // local. A UTC parse gives Sep 5 west of Greenwich.
    expect(date.getFullYear()).toBe(2026);
    expect(date.getMonth()).toBe(8);
    expect(date.getDate()).toBe(6);
    expect(date.getHours()).toBe(0);
  });

  it('tolerates surrounding whitespace', () => {
    expect(parseCalendarDay('  2026-09-06 ').getDate()).toBe(6);
  });

  it('passes a Date through and rejects an invalid one', () => {
    const date = new Date(2026, 8, 6);
    expect(parseCalendarDay(date)).toBe(date);
    expect(parseCalendarDay(new Date('nope'))).toBeNull();
  });

  it('leaves a full timestamp to Date, which already parses it as local', () => {
    const parsed = parseCalendarDay('2026-09-06T21:00:00+02:00');
    expect(parsed.getTime()).toBe(Date.parse('2026-09-06T21:00:00+02:00'));
  });

  it('rejects a day that does not exist instead of rolling it over', () => {
    // Date would turn this into March 3rd and display a day the API never sent.
    expect(parseCalendarDay('2026-02-31')).toBeNull();
    expect(parseCalendarDay('2026-13-01')).toBeNull();
  });

  it('returns null for empty, malformed and non-string input', () => {
    expect(parseCalendarDay('')).toBeNull();
    expect(parseCalendarDay(null)).toBeNull();
    expect(parseCalendarDay(undefined)).toBeNull();
    expect(parseCalendarDay('not a date')).toBeNull();
    expect(parseCalendarDay(20260906)).toBeNull();
  });
});

describe('the calendar-day formatters', () => {
  // Built from local components, so the expectation carries the same zone the
  // formatter will use — the comparison is about the *day*, not the zone.
  const sameDayLocally = (options) =>
    new Date(2026, 8, 6).toLocaleDateString('en-US', options);

  it('formats the day the string names, in any zone', () => {
    expect(formatCalendarDay('2026-09-06')).toBe(sameDayLocally({
      weekday: 'short', year: 'numeric', month: 'short', day: 'numeric'
    }));
    expect(formatCalendarDayLong('2026-09-06')).toBe(sameDayLocally({
      weekday: 'long', year: 'numeric', month: 'long', day: 'numeric'
    }));
    expect(formatCalendarDayShort('2026-09-06')).toBe(sameDayLocally({
      year: 'numeric', month: 'short', day: 'numeric'
    }));
  });

  it('renders the expected shapes', () => {
    // Zone-independent: Sep 6 2026 is a Sunday everywhere, and the parse is
    // local, so no formatter can land on a neighbouring day.
    expect(formatCalendarDay('2026-09-06')).toBe('Sun, Sep 6, 2026');
    expect(formatCalendarDayLong('2026-09-06')).toBe('Sunday, September 6, 2026');
    expect(formatCalendarDayShort('2026-09-06')).toBe('Sep 6, 2026');
  });

  it('echoes an unusable string rather than blanking a match card', () => {
    expect(formatCalendarDay('not a date')).toBe('not a date');
    expect(formatCalendarDayShort('2026-02-31')).toBe('2026-02-31');
  });

  it('returns an empty string for non-string input', () => {
    expect(formatCalendarDay(null)).toBe('');
    expect(formatCalendarDayShort(undefined)).toBe('');
  });
});
