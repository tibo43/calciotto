import { scoreSides, teamScoreLine, sizeCounter, recapEntries, shareSubtitle } from '@/services/v2Edit';

const player = (name, goals, own = 0) => ({ Name: name, GoalNumber: goals, OwnGoals: own });

describe('scoreSides', () => {
  it('adds the other team\'s own goals and ignores how many players each side has', () => {
    const sides = scoreSides([
      [player('Paolo Greco', 0, 2), player('Marco Rossi', 2), player('Andrea Conti', 1), player('Alessandro Bartoli', 1)],
      [player('Matteo Ricci', 1), player('Thibaut Fabre', 0), player('Camille Durand', 0), player('Giulia Neri', 0)],
    ]);
    expect(sides[0].score).toBe(4);
    expect(sides[1].score).toBe(3);
    expect(sides[1].concededOwnGoals).toBe(2);
    expect(teamScoreLine(sides[0])).toBe('4 goals');
    expect(teamScoreLine(sides[1])).toBe('3 goals · 2 OG');
  });

  it('accepts 5 v 4 and 12 v 11 without a special case', () => {
    const five = Array.from({ length: 5 }, (_, i) => player(`A${i}`, i === 0 ? 1 : 0));
    const four = Array.from({ length: 4 }, (_, i) => player(`B${i}`, 0));
    expect(scoreSides([five, four])[0].score).toBe(1);
    const twelve = Array.from({ length: 12 }, (_, i) => player(`C${i}`, 0));
    const eleven = Array.from({ length: 11 }, (_, i) => player(`D${i}`, 0, i === 0 ? 1 : 0));
    expect(scoreSides([twelve, eleven]).map((side) => side.score)).toEqual([1, 0]);
  });
});

describe('sizeCounter', () => {
  it('is a neutral headcount, including when the sides differ', () => {
    expect(sizeCounter(7, 4)).toBe('7 v 4');
    expect(sizeCounter(12, 11)).toBe('12 v 11');
    expect(sizeCounter(0, 0)).toBe('0 v 0');
  });
});

describe('recapEntries', () => {
  it('puts an own goal on the side that benefits, under the scorer\'s last name', () => {
    const lines = recapEntries([
      [player('Marco Rossi', 2), player('Paolo Greco', 0, 2)],
      [player('Matteo Ricci', 1)],
    ]);
    expect(lines.map((line) => line.text)).toEqual(['Rossi ×2', 'Greco (OG ×2)', 'Ricci']);
    expect(lines.find((line) => line.text.startsWith('Greco')).away).toBe(true);
  });
});

describe('shareSubtitle', () => {
  it('announces the match until someone is placed, then the line-ups', () => {
    const empty = [{ Name: 'Les Bleus', Players: [] }, { Name: 'Les Rouges', Players: [] }];
    expect(shareSubtitle(empty)).toMatch(/announcement/);
    const made = [{ Name: 'Les Bleus', Players: [player('A', 0)] }, { Name: 'Les Rouges', Players: [player('B', 0), player('C', 0)] }];
    expect(shareSubtitle(made)).toBe('The line-ups will be sent — Les Bleus 1 · Les Rouges 2.');
  });
});
