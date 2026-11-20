import {
  rosterFullyComposed,
  predictionState,
  predictionsAreOpen,
  predictionDisplayState,
  predictionDisplayLabel,
  matchWinnerId,
  settlePrediction,
  PREDICTIONS_NOT_AVAILABLE,
  PREDICTIONS_OPEN,
  PREDICTIONS_CLOSED,
  PREDICTIONS_AWAITING_RESULT,
  PREDICTIONS_SETTLED
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

const scored = (scoreA, scoreB, extra = {}) => {
  const [a, b] = composed();
  return { ScheduledAt: KICKOFF, Teams: [{ ...a, Score: scoreA }, { ...b, Score: scoreB }], ...extra };
};
const ODDS = { TeamAOdds: 2.5, DrawOdds: 3.4, TeamBOdds: 2.1 };

describe('predictionDisplayState', () => {
  it('passes the open and not-available states through', () => {
    expect(predictionDisplayState(scored(0, 0), KICKOFF_MS - 1)).toBe(PREDICTIONS_OPEN);
    expect(predictionDisplayState({ Teams: composed() }, KICKOFF_MS - 1)).toBe(PREDICTIONS_NOT_AVAILABLE);
  });

  // Kick-off alone settles nothing: until the admin saves the score, a 0-0
  // may just mean nobody has entered it.
  it('is awaiting the result after kick-off until a result is recorded', () => {
    expect(predictionDisplayState(scored(0, 0), KICKOFF_MS + 1)).toBe(PREDICTIONS_AWAITING_RESULT);
  });

  it('is settled once the result is recorded', () => {
    const match = scored(2, 1, { ResultRecordedAt: '2026-09-06T23:00:00+02:00' });
    expect(predictionDisplayState(match, KICKOFF_MS + 1)).toBe(PREDICTIONS_SETTLED);
  });

  it('labels every state', () => {
    expect(predictionDisplayLabel(PREDICTIONS_OPEN)).toBe('Predictions open');
    expect(predictionDisplayLabel(PREDICTIONS_AWAITING_RESULT)).toBe('Awaiting result');
    expect(predictionDisplayLabel(PREDICTIONS_SETTLED)).toBe('Settled');
    expect(predictionDisplayLabel(PREDICTIONS_NOT_AVAILABLE)).toBe('Not available');
  });
});

describe('matchWinnerId', () => {
  it('is the higher-scoring team, or null for a draw', () => {
    expect(matchWinnerId(scored(2, 1))).toBe('a');
    expect(matchWinnerId(scored(0, 3))).toBe('b');
    expect(matchWinnerId(scored(1, 1))).toBeNull();
  });
});

describe('settlePrediction', () => {
  const settled = (scoreA, scoreB) => scored(scoreA, scoreB, { ResultRecordedAt: '2026-09-06T23:00:00+02:00' });

  it('nets the backed side\'s odds minus one on a hit', () => {
    expect(settlePrediction(settled(2, 1), ODDS, 'a')).toEqual({ correct: true, points: 1.5 });
    expect(settlePrediction(settled(0, 1), ODDS, 'b').points).toBeCloseTo(1.1);
  });

  it('pays the draw odds on a correct draw, null and undefined alike', () => {
    expect(settlePrediction(settled(1, 1), ODDS, null).points).toBeCloseTo(2.4);
    expect(settlePrediction(settled(1, 1), ODDS, undefined).correct).toBe(true);
  });

  it('costs one on a miss, including a draw backed on a decided match', () => {
    expect(settlePrediction(settled(2, 1), ODDS, 'b')).toEqual({ correct: false, points: -1 });
    expect(settlePrediction(settled(2, 1), ODDS, null)).toEqual({ correct: false, points: -1 });
  });

  it('settles nothing before the result is recorded, or without odds', () => {
    expect(settlePrediction(scored(2, 1), ODDS, 'a')).toBeNull();
    expect(settlePrediction(settled(2, 1), null, 'a')).toBeNull();
  });
});
