import { motmVotingDeadlineMs, isMotmVotingOpen } from '@/services/motmVoting';

// A match played 2026-09-03 accepts votes through 2026-09-04, closing at the
// stroke of 2026-09-05 — the exact example the product rule was phrased
// around. Local-midnight instants, so nothing here depends on the machine's
// zone beyond what parseCalendarDay itself already commits to.
const PLAYED_ON = '2026-09-03';
const at = (y, m, d) => new Date(y, m - 1, d).getTime();

describe('motmVotingDeadlineMs', () => {
  it('is local midnight two days after the match Date', () => {
    const match = { Date: PLAYED_ON };
    expect(motmVotingDeadlineMs(match)).toBe(at(2026, 9, 5));
  });

  it('ignores ScheduledAt/CreatedAt entirely — only Date matters', () => {
    const match = { Date: PLAYED_ON, ScheduledAt: '2020-01-01T00:00:00Z', CreatedAt: '2020-01-01T00:00:00Z' };
    expect(motmVotingDeadlineMs(match)).toBe(at(2026, 9, 5));
  });

  it('is NaN for a null/undefined match, or one with no Date, rather than throwing', () => {
    expect(Number.isNaN(motmVotingDeadlineMs(null))).toBe(true);
    expect(Number.isNaN(motmVotingDeadlineMs(undefined))).toBe(true);
    expect(Number.isNaN(motmVotingDeadlineMs({}))).toBe(true);
  });
});

describe('isMotmVotingOpen', () => {
  it('is open on the match day itself', () => {
    const match = { Date: PLAYED_ON };
    expect(isMotmVotingOpen(match, at(2026, 9, 3))).toBe(true);
  });

  it('is still open the day after, right up to the last millisecond', () => {
    const match = { Date: PLAYED_ON };
    expect(isMotmVotingOpen(match, at(2026, 9, 5) - 1)).toBe(true);
  });

  it('is closed exactly at the stroke of the second day after', () => {
    const match = { Date: PLAYED_ON };
    expect(isMotmVotingOpen(match, at(2026, 9, 5))).toBe(false);
  });

  it('is closed well past the deadline', () => {
    const match = { Date: PLAYED_ON };
    expect(isMotmVotingOpen(match, at(2026, 10, 1))).toBe(false);
  });

  it('a scheduled match\'s window still depends only on Date, ignoring kick-off entirely', () => {
    const match = { Date: PLAYED_ON, ScheduledAt: '2020-01-01T00:00:00Z' };
    expect(isMotmVotingOpen(match, at(2026, 9, 4))).toBe(true);
    expect(isMotmVotingOpen(match, at(2026, 9, 5))).toBe(false);
  });

  it('fails open when no Date can be determined at all', () => {
    expect(isMotmVotingOpen({}, Date.now())).toBe(true);
    expect(isMotmVotingOpen(null, Date.now())).toBe(true);
  });
});
