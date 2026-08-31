import {
  isScheduledMatch,
  deriveRegistrationState,
  registrationsAreOpen,
  REGISTRATION_UNSCHEDULED,
  REGISTRATION_NOT_OPEN_YET,
  REGISTRATION_OPEN,
  REGISTRATION_CLOSED_BY_ADMIN,
  REGISTRATION_CLOSED_AT_KICKOFF
} from '@/services/matchRegistration';

// Fixed instants with explicit offsets, so nothing here depends on the machine's
// zone: Date.parse resolves each to an absolute epoch value either way.
const OPENS_AT = '2026-09-01T12:00:00+02:00';
const KICKOFF = '2026-09-06T20:30:00+02:00';

const at = (rfc3339) => Date.parse(rfc3339);

const scheduled = (overrides = {}) => ({
  ScheduledAt: KICKOFF,
  RegistrationOpensAt: OPENS_AT,
  MaxPlayers: 16,
  RegistrationCount: 0,
  ...overrides
});

describe('isScheduledMatch', () => {
  it('is true for a match carrying a kick-off', () => {
    expect(isScheduledMatch(scheduled())).toBe(true);
  });

  it('is false for an ordinary match, which carries none of the scheduling keys', () => {
    expect(isScheduledMatch({ ID: 'm', Date: '2026-09-06', Teams: [] })).toBe(false);
  });

  it('is false for a null/undefined match rather than throwing', () => {
    expect(isScheduledMatch(null)).toBe(false);
    expect(isScheduledMatch(undefined)).toBe(false);
  });
});

describe('deriveRegistrationState', () => {
  it('reports an unscheduled match as such, whatever the clock says', () => {
    expect(deriveRegistrationState({ ID: 'm' }, at('2026-09-03T12:00:00+02:00')))
      .toBe(REGISTRATION_UNSCHEDULED);
  });

  it('is not-open-yet before the sign-ups open', () => {
    expect(deriveRegistrationState(scheduled(), at('2026-09-01T11:59:00+02:00')))
      .toBe(REGISTRATION_NOT_OPEN_YET);
  });

  it('is open from the opening instant itself', () => {
    expect(deriveRegistrationState(scheduled(), at(OPENS_AT))).toBe(REGISTRATION_OPEN);
  });

  it('is open between the opening and kick-off', () => {
    expect(deriveRegistrationState(scheduled(), at('2026-09-04T09:00:00+02:00')))
      .toBe(REGISTRATION_OPEN);
  });

  it('is closed-by-admin once RegistrationsClosedAt is present, even mid-window', () => {
    const match = scheduled({ RegistrationsClosedAt: '2026-09-05T18:00:00+02:00' });
    expect(deriveRegistrationState(match, at('2026-09-05T19:00:00+02:00')))
      .toBe(REGISTRATION_CLOSED_BY_ADMIN);
  });

  // The two closed states have to stay distinguishable: only the admin's one is
  // reversible, and Reopen is offered on exactly that one.
  it('prefers closed-by-admin over not-open-yet, so the copy never promises an opening that was already cancelled', () => {
    const match = scheduled({ RegistrationsClosedAt: '2026-08-31T10:00:00+02:00' });
    expect(deriveRegistrationState(match, at('2026-08-31T11:00:00+02:00')))
      .toBe(REGISTRATION_CLOSED_BY_ADMIN);
  });

  it('is closed-at-kickoff at the kick-off instant, with no admin close at all', () => {
    expect(deriveRegistrationState(scheduled(), at(KICKOFF)))
      .toBe(REGISTRATION_CLOSED_AT_KICKOFF);
  });

  it('is closed-at-kickoff after kick-off', () => {
    expect(deriveRegistrationState(scheduled(), at('2026-09-06T22:00:00+02:00')))
      .toBe(REGISTRATION_CLOSED_AT_KICKOFF);
  });

  it('treats a scheduled match with an unparseable opening time as having no lower bound', () => {
    const match = scheduled({ RegistrationOpensAt: undefined });
    expect(deriveRegistrationState(match, at('2026-09-02T12:00:00+02:00')))
      .toBe(REGISTRATION_OPEN);
  });
});

describe('registrationsAreOpen', () => {
  // Both Participate and Withdraw hang off this one predicate, because the
  // backend refuses both outside the same window.
  it('is true only for the open state', () => {
    expect(registrationsAreOpen(REGISTRATION_OPEN)).toBe(true);
    expect(registrationsAreOpen(REGISTRATION_NOT_OPEN_YET)).toBe(false);
    expect(registrationsAreOpen(REGISTRATION_CLOSED_BY_ADMIN)).toBe(false);
    expect(registrationsAreOpen(REGISTRATION_CLOSED_AT_KICKOFF)).toBe(false);
    expect(registrationsAreOpen(REGISTRATION_UNSCHEDULED)).toBe(false);
  });
});
