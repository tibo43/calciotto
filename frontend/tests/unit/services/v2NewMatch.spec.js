import { buildNewMatch, localDateTimeValue, DEFAULT_MAX_PLAYERS } from '@/services/v2NewMatch';

// Wednesday 2 September 2026, 10:00 local time.
const NOW = new Date(2026, 8, 2, 10, 0);
const valid = { day: '2026-09-06', time: '20:30', opensAt: '2026-09-02T10:00', maxPlayers: '16', registerCreator: false };

describe('buildNewMatch', () => {
  it('builds a scheduled match, with times in the local offset', () => {
    const { matchData, scheduling } = buildNewMatch(valid, NOW);
    expect(matchData).toEqual({ Date: '2026-09-06' });
    expect(scheduling.maxPlayers).toBe(16);
    expect(scheduling.scheduledAt).toMatch(/^2026-09-06T20:30:00[+-]\d{2}:\d{2}$/);
    expect(scheduling.registrationOpensAt).toMatch(/^2026-09-02T10:00:00[+-]\d{2}:\d{2}$/);
    expect(scheduling).not.toHaveProperty('registerCreator');
  });

  it('adds register_creator only when asked', () => {
    expect(buildNewMatch({ ...valid, registerCreator: true }, NOW).scheduling.registerCreator).toBe(true);
  });

  it.each([
    [{ day: '' }, 'day'],
    [{ time: '' }, 'time'],
    [{ opensAt: '' }, 'opensAt'],
    [{ maxPlayers: '0' }, 'maxPlayers'],
    [{ maxPlayers: '8.5' }, 'maxPlayers'],
    [{ day: '2026-09-01' }, 'time'],
    [{ opensAt: '2026-09-06T20:30' }, 'opensAt'],
  ])('refuses %j, pointing at %s', (change, field) => {
    const result = buildNewMatch({ ...valid, ...change }, NOW);
    expect(result.field).toBe(field);
    expect(result.error).toBeTruthy();
  });
});

describe('localDateTimeValue', () => {
  it('formats a moment for a datetime-local input, to the minute', () => {
    expect(localDateTimeValue(new Date(2026, 0, 5, 7, 4, 59))).toBe('2026-01-05T07:04');
  });

  it('defaults to 16 places', () => {
    expect(DEFAULT_MAX_PLAYERS).toBe(16);
  });
});
