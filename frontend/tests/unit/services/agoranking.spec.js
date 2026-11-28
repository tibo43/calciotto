import { formatRating, pointsLabel, recordLine, rankedLine, normalizeAgoranking, emptyAgoranking } from '@/services/agoranking';

describe('agoranking display helpers', () => {
  it('shows the rating with two decimals', () => {
    expect(formatRating(1.6149)).toBe('1.61');
    expect(formatRating(1.665)).toBe('1.67');
    expect(formatRating(2)).toBe('2.00');
    expect(formatRating(0)).toBe('0.00');
    expect(formatRating(undefined)).toBe('–');
  });

  it('says pt for a single point and pts otherwise', () => {
    expect(pointsLabel(0)).toBe('0 pts');
    expect(pointsLabel(1)).toBe('1 pt');
    expect(pointsLabel(14)).toBe('14 pts');
  });

  it('builds the mobile lines', () => {
    const row = { Played: 8, Won: 4, Drawn: 2, Lost: 2, Points: 14 };
    expect(rankedLine(row)).toBe('8 M · 4W 2D 2L · 14 pts');
    expect(recordLine({ Played: 1, Won: 0, Drawn: 1, Lost: 0, Points: 1 })).toBe('0W 1D 0L · 1 pt');
  });

  it('always returns the full shape', () => {
    expect(normalizeAgoranking(null)).toEqual(emptyAgoranking());
    expect(normalizeAgoranking({ Matches: 3, N: 1, Ranked: null })).toEqual({ Matches: 3, N: 1, GroupAverage: 0, Ranked: [], Unranked: [] });
    const ranked = [{ PlayerID: 'a' }];
    expect(normalizeAgoranking({ Ranked: ranked }).Ranked).toBe(ranked);
  });
});
