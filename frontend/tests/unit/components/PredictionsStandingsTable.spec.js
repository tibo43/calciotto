import { mount } from '@vue/test-utils';
import PredictionsStandingsTable from '@/components/PredictionsStandingsTable.vue';

const settled = (day, correct, overrides = {}) => ({
  MatchID: `m-${day}`,
  Date: `2026-10-${String(day).padStart(2, '0')}`,
  TeamAID: 'ta',
  TeamAName: 'Black',
  TeamBID: 'tb',
  TeamBName: 'White',
  ScoreA: 2,
  ScoreB: 1,
  PredictedTeamID: 'ta',
  Odds: 2.5,
  Correct: correct,
  Points: correct ? 1.5 : -1,
  ...overrides
});

const ROWS = [
  {
    PlayerID: 'p1',
    Name: 'alice',
    Points: 2.0,
    IsMember: true,
    // Oldest first, as the backend sends it: six entries, so only the last
    // five show as squares.
    History: [settled(1, false), settled(2, true), settled(3, true), settled(4, false), settled(5, true),
      settled(6, false, { PredictedTeamID: null, Odds: 3.4 })]
  },
  { PlayerID: 'p2', Name: 'bob', Points: -1, IsMember: true, History: [settled(6, false, { PredictedTeamID: 'tb' })] }
];

const squareClasses = (row) => row.findAll('.form-square').map(sq => ['won', 'lost', 'empty'].find(c => sq.classes(c)));

describe('PredictionsStandingsTable', () => {
  it('shows the last five predictions as squares, oldest left', () => {
    const wrapper = mount(PredictionsStandingsTable, { props: { rows: ROWS } });
    const [alice] = wrapper.findAll('.player-row');

    expect(squareClasses(alice)).toEqual(['won', 'won', 'lost', 'won', 'lost']);
  });

  it('pads a short history with empty squares on the left', () => {
    const wrapper = mount(PredictionsStandingsTable, { props: { rows: ROWS } });
    const bob = wrapper.findAll('.player-row')[1];

    expect(squareClasses(bob)).toEqual(['empty', 'empty', 'empty', 'empty', 'lost']);
  });

  it('unfolds a player\'s whole history on click, newest first, and folds it again', async () => {
    const wrapper = mount(PredictionsStandingsTable, { props: { rows: ROWS } });
    expect(wrapper.find('.history-row').exists()).toBe(false);

    await wrapper.findAll('.player-row')[0].trigger('click');
    const entries = wrapper.findAll('.history-entry');
    expect(entries).toHaveLength(6);
    expect(entries[0].find('.history-pick').text()).toBe('Draw · 3.40x');
    expect(entries[0].find('.history-points').text()).toBe('-1.00');
    expect(entries[4].find('.history-score').text()).toBe('Black 2 – 1 White');
    expect(entries[4].find('.history-points').text()).toBe('+1.50');

    await wrapper.findAll('.player-row')[0].trigger('click');
    expect(wrapper.find('.history-row').exists()).toBe(false);
  });

  it('shows one player\'s history at a time, naming the team that was picked', async () => {
    const wrapper = mount(PredictionsStandingsTable, { props: { rows: ROWS } });

    await wrapper.findAll('.player-row')[0].trigger('click');
    await wrapper.findAll('.player-row')[1].trigger('click');

    expect(wrapper.findAll('.history-row')).toHaveLength(1);
    expect(wrapper.find('.history-pick').text()).toBe('White · 2.50x');
  });
});
