import {
  rosterFullyComposed,
  predictionState,
  predictionsAreOpen,
  PREDICTIONS_NOT_AVAILABLE,
  PREDICTIONS_OPEN,
  PREDICTIONS_CLOSED
} from '@/services/matchPrediction';

const KICKOFF = '2026-09-06T21:00:00+02:00';
const KICKOFF_MS = Date.parse(KICKOFF);

const team = (id, players) => ({ ID: id, Name: id, Players: players.map(name => ({ ID: name, Name: name })) });
const composed = () => [team('a', ['alice']), team('b', ['bob'])];

describe('rosterFullyComposed', () => {
  it('is true once both teams have at least one player', () => {
    expect(rosterFullyComposed({ Teams: composed() })).toBe(true);
  });

  // Stricter than teamsAreComposed: one populated side is enough to show a
  // roster, not enough to predict a winner between.
  it('is false when only one side has players', () => {
    expect(rosterFullyComposed({ Teams: [team('a', ['alice']), team('b', [])] })).toBe(false);
  });

  it('is false for anything other than exactly two teams', () => {
    expect(rosterFullyComposed({ Teams: [team('a', ['alice'])] })).toBe(false);
    expect(rosterFullyComposed({ Teams: [...composed(), team('c', ['carol'])] })).toBe(false);
  });

  it('is false for a null match, missing Teams or a team with no Players array', () => {
    expect(rosterFullyComposed(null)).toBe(false);
    expect(rosterFullyComposed({})).toBe(false);
    expect(rosterFullyComposed({ Teams: [{ ID: 'a' }, team('b', ['bob'])] })).toBe(false);
  });
});

describe('predictionState', () => {
  it('is not available for an unscheduled match, even with a composed roster', () => {
    expect(predictionState({ Teams: composed() }, KICKOFF_MS - 1000)).toBe(PREDICTIONS_NOT_AVAILABLE);
  });

  it('is not available for a scheduled match whose roster is not fully composed', () => {
    const match = { ScheduledAt: KICKOFF, Teams: [team('a', ['alice']), team('b', [])] };
    expect(predictionState(match, KICKOFF_MS - 1000)).toBe(PREDICTIONS_NOT_AVAILABLE);
  });

  it('is open before kick-off, right up to the last millisecond', () => {
    const match = { ScheduledAt: KICKOFF, Teams: composed() };
    expect(predictionState(match, KICKOFF_MS - 1)).toBe(PREDICTIONS_OPEN);
  });

  // Mirrors the Go side's !now.Before(kickoff): kick-off itself is closed.
  it('is closed exactly at kick-off and after', () => {
    const match = { ScheduledAt: KICKOFF, Teams: composed() };
    expect(predictionState(match, KICKOFF_MS)).toBe(PREDICTIONS_CLOSED);
    expect(predictionState(match, KICKOFF_MS + 60000)).toBe(PREDICTIONS_CLOSED);
  });
});

describe('predictionsAreOpen', () => {
  it('is true only in the open state', () => {
    const match = { ScheduledAt: KICKOFF, Teams: composed() };
    expect(predictionsAreOpen(match, KICKOFF_MS - 1)).toBe(true);
    expect(predictionsAreOpen(match, KICKOFF_MS)).toBe(false);
    expect(predictionsAreOpen({ Teams: composed() }, KICKOFF_MS - 1)).toBe(false);
  });
});
