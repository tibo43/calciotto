import { formatUTCOffset, toLocalRFC3339, dateTimeLocalToRFC3339 } from '@/services/datetime';

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
