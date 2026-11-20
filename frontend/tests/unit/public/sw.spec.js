import { createRequire } from 'node:module';

// The notification wording lives in public/sw.js (see its comment); the file
// exports it for exactly this test.
const { buildNotification } = createRequire(import.meta.url)('../../../public/sw.js');

const KICKOFF = '2026-09-06T19:00:00Z';
const kickoffText = new Date(KICKOFF).toLocaleString('en-US', {
  weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit'
});
const kickoffTime = new Date(KICKOFF).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });

describe('buildNotification', () => {
  it('announces open sign-ups with the group and kick-off', () => {
    expect(buildNotification({ type: 'registration_open', groupName: 'Calciotto Milano', kickoffAt: KICKOFF })).toEqual({
      title: 'Sign-ups are open',
      body: `Calciotto Milano: match on ${kickoffText}. Grab your spot.`
    });
  });

  it("names the player's team", () => {
    expect(buildNotification({ type: 'team_assigned', teamName: 'Black', groupName: 'Calciotto Milano', kickoffAt: KICKOFF })).toEqual({
      title: "You're playing for Black",
      body: `Calciotto Milano: match on ${kickoffText}.`
    });
  });

  it('reminds of kick-off, with the team once composed', () => {
    expect(buildNotification({ type: 'kickoff_reminder', kickoffAt: KICKOFF, teamName: 'White' }).body)
      .toBe(`Kick-off at ${kickoffTime}, you're with White.`);
    expect(buildNotification({ type: 'kickoff_reminder', kickoffAt: KICKOFF }).body)
      .toBe(`Kick-off at ${kickoffTime}.`);
  });

  it('gives the final score and asks for a Man of the Match vote', () => {
    expect(buildNotification({
      type: 'result_recorded',
      score: [{ team: 'Black', goals: 3 }, { team: 'White', goals: 1 }]
    })).toEqual({ title: 'Final score: Black 3 – White 1', body: 'Vote for your Man of the Match.' });
  });

  it('announces that predictions are open for the match', () => {
    expect(buildNotification({ type: 'predictions_open', groupName: 'Calciotto Milano', kickoffAt: KICKOFF })).toEqual({
      title: 'Predictions are open',
      body: `Calciotto Milano: match on ${kickoffText}. Make your pick.`
    });
  });

  it('settles a prediction with the winner and the points earned or lost', () => {
    expect(buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Bianco', goals: 5 }, { team: 'Nero', goals: 3 }],
      points: (1 / 0.3) - 1
    })).toEqual({
      title: 'Predictions settled',
      body: 'Bianco won! You earned 2.33 points.'
    });
    expect(buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Bianco', goals: 5 }, { team: 'Nero', goals: 3 }],
      points: -1
    }).body).toBe('Bianco won! You lost 1 point.');
    expect(buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Bianco', goals: 0 }, { team: 'Nero', goals: 2 }],
      points: 1.2
    }).body).toBe('Nero won! You earned 1.20 points.');
  });

  it('says it is a draw when the goals are level', () => {
    expect(buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Bianco', goals: 1 }, { team: 'Nero', goals: 1 }],
      points: 1.5
    }).body).toBe("It's a draw! You earned 1.50 points.");
    expect(buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Bianco', goals: 0 }, { team: 'Nero', goals: 0 }],
      points: -1
    }).body).toBe("It's a draw! You lost 1 point.");
  });

  it('never repeats the score in the settled notification', () => {
    const { body } = buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Bianco', goals: 5 }, { team: 'Nero', goals: 3 }],
      points: 2
    });
    expect(body).not.toContain('Final score');
    expect(body).not.toMatch(/\d+ – \d+|5|3/);
  });

  it('shows the points alone without a usable score', () => {
    expect(buildNotification({ type: 'predictions_settled', points: 1.5 }).body)
      .toBe('You earned 1.50 points.');
    expect(buildNotification({ type: 'predictions_settled', score: [], points: -1.5 }).body)
      .toBe('You lost 1.50 points.');
    expect(buildNotification({ type: 'predictions_settled', score: [{ team: 'Bianco', goals: 2 }], points: 1 }).body)
      .toBe('You earned 1 point.');
    expect(buildNotification({ type: 'predictions_settled', points: 2 }).body)
      .toBe('You earned 2 points.');
    expect(buildNotification({ type: 'predictions_settled', points: 0 }).body)
      .toBe('You earned 0 points.');
  });

  it('still shows something for a type it does not know', () => {
    expect(buildNotification({ type: 'something_new' }).title).toBe('Calciotto');
  });

  it('names who joined which group for an admin', () => {
    const notification = buildNotification({ type: 'member_joined', playerName: 'Sam', groupName: 'Thursday Football' });
    expect(notification.title).toBe('Sam joined Thursday Football');
    expect(notification.body).toContain('generate a new invite code');
  });
});
