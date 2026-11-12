// The notification wording lives in public/sw.js (see its comment); the file
// exports it for exactly this test.
const { buildNotification } = require('../../../public/sw.js');

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

  it('settles a prediction with the score and the points earned', () => {
    expect(buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Black', goals: 3 }, { team: 'White', goals: 1 }],
      points: (1 / 0.3) - 1
    })).toEqual({
      title: 'Predictions settled',
      body: 'Final score: Black 3 – White 1. You earned +2.33 points.'
    });
    expect(buildNotification({
      type: 'predictions_settled',
      score: [{ team: 'Black', goals: 1 }, { team: 'White', goals: 1 }],
      points: -1
    }).body).toBe('Final score: Black 1 – White 1. You earned -1.00 points.');
    expect(buildNotification({ type: 'predictions_settled', points: 1.5 }).body)
      .toBe('You earned +1.50 points.');
  });

  it('still shows something for a type it does not know', () => {
    expect(buildNotification({ type: 'something_new' }).title).toBe('Calciotto');
  });
});
