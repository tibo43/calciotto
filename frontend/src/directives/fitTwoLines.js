// v-fit-two-lines, on a scoreboard's .board-teams: when either team name
// would take more than two lines at its normal size, the board gets the
// is-step-down class and both names drop one font-size step together, so
// they always share one size. Measured on the names themselves, so it
// follows the real font and column width: again once the web fonts are in
// and on every resize.

export const MAX_LINES = 2;
const NAMES = '.board-team .name';

// How many lines a box of this height holds at this line height.
export function lineCount(height, lineHeight) {
  if (!(height > 0) || !(lineHeight > 0)) return 0;
  return Math.round(height / lineHeight);
}

export function needsStepDown(height, lineHeight, maxLines = MAX_LINES) {
  return lineCount(height, lineHeight) > maxLines;
}

// The pair rule: one name over the limit steps both down.
export function pairNeedsStepDown(names, maxLines = MAX_LINES) {
  return names.some(({ height, lineHeight }) => needsStepDown(height, lineHeight, maxLines));
}

function lineHeightOf(el) {
  const style = window.getComputedStyle(el);
  const value = parseFloat(style.lineHeight);
  if (value > 0) return value;
  return parseFloat(style.fontSize) || 0;
}

function fit(board) {
  board.classList.remove('is-step-down');
  const names = [...board.querySelectorAll(NAMES)].map((el) => ({ height: el.getBoundingClientRect().height, lineHeight: lineHeightOf(el) }));
  if (pairNeedsStepDown(names)) board.classList.add('is-step-down');
}

export default {
  mounted(el) {
    const run = () => fit(el);
    el.__fitTwoLines = run;
    run();
    if (document.fonts && document.fonts.ready) document.fonts.ready.then(run).catch(() => {});
    window.addEventListener('resize', run);
  },
  updated(el) {
    fit(el);
  },
  unmounted(el) {
    if (el.__fitTwoLines) window.removeEventListener('resize', el.__fitTwoLines);
    delete el.__fitTwoLines;
  },
};
