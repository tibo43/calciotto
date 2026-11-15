import { mount } from '@vue/test-utils';
import fitTwoLines, { lineCount, needsStepDown, pairNeedsStepDown, MAX_LINES } from '@/directives/fitTwoLines';

describe('fitTwoLines', () => {
  it('counts lines from the height and the line height', () => {
    expect(lineCount(88, 44)).toBe(2);
    expect(lineCount(132, 44)).toBe(3);
    expect(lineCount(0, 44)).toBe(0);
    expect(lineCount(44, 0)).toBe(0);
  });

  it('steps a name down only beyond two lines', () => {
    expect(MAX_LINES).toBe(2);
    expect(needsStepDown(44, 44)).toBe(false);
    expect(needsStepDown(88, 44)).toBe(false);
    expect(needsStepDown(132, 44)).toBe(true);
  });

  it('steps the pair down as soon as one name needs it', () => {
    expect(pairNeedsStepDown([{ height: 44, lineHeight: 44 }, { height: 88, lineHeight: 44 }])).toBe(false);
    expect(pairNeedsStepDown([{ height: 132, lineHeight: 44 }, { height: 44, lineHeight: 44 }])).toBe(true);
    expect(pairNeedsStepDown([{ height: 44, lineHeight: 44 }, { height: 132, lineHeight: 44 }])).toBe(true);
    expect(pairNeedsStepDown([])).toBe(false);
  });

  it('puts is-step-down on the board when either name is taller than two lines, and takes it off again', () => {
    const heights = { home: 44, away: 132 };
    const wrapper = mount({
      template: '<div v-fit-two-lines class="board-teams"><div class="board-team"><span class="name home">A</span></div><div class="board-score">1–0</div><div class="board-team"><span class="name away">B</span></div></div>',
      directives: { fitTwoLines },
    }, { attachTo: document.body });
    const board = wrapper.element;
    for (const side of ['home', 'away']) {
      const el = board.querySelector(`.${side}`);
      el.style.lineHeight = '44px';
      el.getBoundingClientRect = () => ({ height: heights[side] });
    }
    window.dispatchEvent(new Event('resize'));
    expect(board.classList.contains('is-step-down')).toBe(true);
    expect(board.querySelector('.home').classList.contains('is-step-down')).toBe(false);
    heights.away = 88;
    window.dispatchEvent(new Event('resize'));
    expect(board.classList.contains('is-step-down')).toBe(false);
    heights.home = 176;
    window.dispatchEvent(new Event('resize'));
    expect(board.classList.contains('is-step-down')).toBe(true);
    wrapper.unmount();
  });
});
