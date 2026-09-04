import { MOTM_VOTING_WINDOW_MS, motmVotingAnchorMs, isMotmVotingOpen } from '@/services/motmVoting';

// Mirrors matchRegistration.spec.js's own style: fixed instants with explicit
// offsets, so nothing here depends on the machine's zone.
const KICKOFF = '2026-09-06T20:30:00+02:00';
const CREATED_AT = '2026-09-01T12:00:00+02:00';

const at = (rfc3339) => Date.parse(rfc3339);

describe('motmVotingAnchorMs', () => {
  it('anchors on ScheduledAt for a scheduled match, ignoring CreatedAt entirely', () => {
    const match = { ScheduledAt: KICKOFF, CreatedAt: '2020-01-01T00:00:00Z' };
    expect(motmVotingAnchorMs(match)).toBe(at(KICKOFF));
  });

  it('anchors on CreatedAt for a match with no kick-off at all', () => {
    const match = { CreatedAt: CREATED_AT };
    expect(motmVotingAnchorMs(match)).toBe(at(CREATED_AT));
  });

  it('is NaN for a null/undefined match rather than throwing', () => {
    expect(Number.isNaN(motmVotingAnchorMs(null))).toBe(true);
    expect(Number.isNaN(motmVotingAnchorMs(undefined))).toBe(true);
  });
});

describe('isMotmVotingOpen', () => {
  it('is open right at the anchor instant, for a scheduled match', () => {
    const match = { ScheduledAt: KICKOFF };
    expect(isMotmVotingOpen(match, at(KICKOFF))).toBe(true);
  });

  it('is still open one second before the 24h deadline, for a scheduled match', () => {
    const match = { ScheduledAt: KICKOFF };
    expect(isMotmVotingOpen(match, at(KICKOFF) + MOTM_VOTING_WINDOW_MS - 1000)).toBe(true);
  });

  it('is closed exactly at the 24h deadline, for a scheduled match', () => {
    const match = { ScheduledAt: KICKOFF };
    expect(isMotmVotingOpen(match, at(KICKOFF) + MOTM_VOTING_WINDOW_MS)).toBe(false);
  });

  it('is closed well past the 24h deadline, for a scheduled match', () => {
    const match = { ScheduledAt: KICKOFF };
    expect(isMotmVotingOpen(match, at(KICKOFF) + 30 * 24 * 60 * 60 * 1000)).toBe(false);
  });

  it('anchors on CreatedAt, not ScheduledAt, for a match with no kick-off', () => {
    const match = { CreatedAt: CREATED_AT };
    expect(isMotmVotingOpen(match, at(CREATED_AT) + 1000)).toBe(true);
    expect(isMotmVotingOpen(match, at(CREATED_AT) + MOTM_VOTING_WINDOW_MS)).toBe(false);
  });

  it('a scheduled match ignores a long-past CreatedAt: the window is still open shortly after kick-off', () => {
    const match = { ScheduledAt: KICKOFF, CreatedAt: '2020-01-01T00:00:00Z' };
    expect(isMotmVotingOpen(match, at(KICKOFF) + 60 * 1000)).toBe(true);
  });

  it('fails open when no anchor can be determined at all', () => {
    expect(isMotmVotingOpen({}, Date.now())).toBe(true);
    expect(isMotmVotingOpen(null, Date.now())).toBe(true);
  });
});
