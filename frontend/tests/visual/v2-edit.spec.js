// Admin editing with the design-v2 feature on. The same route renders
// MatchDetails.vue when the feature is off, which is what match-details.spec.js
// still photographs.

const { test, expect } = require('@playwright/test');
const data = require('./fixtures/data');
const { gotoApp, expectEverythingStubbed, expectNoHorizontalOverflow } = require('./fixtures/app');

const ID = data.scheduledClosedMatch.ID;
const EDIT = `/matches/${ID}/edit`;

const registrations = data.registrations.slice(0, 13).map((entry, index) => ({
  ...entry,
  Position: index + 1,
  IsWaiting: index >= 11,
}));

function closedMatch(extra = {}) {
  return {
    ...data.scheduledClosedMatch,
    MaxPlayers: 11,
    Teams: data.scheduledClosedMatch.Teams.map((team) => ({ ...team, Players: [], Score: 0 })),
    ...extra,
  };
}

function scoredMatch() {
  const base = closedMatch({ ResultRecordedAt: '2026-09-06T22:00:00+02:00' });
  return {
    ...base,
    Teams: base.Teams.map((team, index) => ({
      ...team,
      Players: data.playedMatch.Teams[index].Players.map((player) => ({
        ...player,
        OwnGoals: index === 0 && player.Name === 'Giovanni Esposito' ? 2 : (player.OwnGoals || 0),
      })),
    })),
  };
}

// The four stages the steps reflect, from what the server has saved.
function teamsSavedMatch() {
  const base = scoredMatch();
  return {
    ...base,
    ResultRecordedAt: undefined,
    Teams: base.Teams.map((team) => ({ ...team, Players: team.Players.map((player) => ({ ...player, GoalNumber: 0, OwnGoals: 0 })) })),
  };
}
const STAGES = {
  open: () => closedMatch({ RegistrationsClosedAt: undefined }),
  closed: () => closedMatch(),
  'teams saved': teamsSavedMatch,
  'score saved': scoredMatch,
};

async function openEdit(page, path, options = {}) {
  const { match, ...rest } = options;
  return gotoApp(page, path, {
    features: ['design-v2'],
    registrations,
    matchDetails: { [ID]: match || closedMatch() },
    ...rest,
  });
}

async function splitTeams(page) {
  await page.getByRole('button', { name: 'Split automatically' }).click();
  await expect(page.locator('.size-count')).toHaveText('6 v 5');
}

// Moves the first player of the second team onto the first through the
// player's menu, with real clicks (a tap on a phone), then picks them again
// so the menu is open for the screenshot.
async function pickUneven(page) {
  await splitTeams(page);
  const row = page.locator('.compose-col').nth(2).locator('.p-row').first();
  const name = (await row.locator('.nm').innerText()).trim();
  await row.click();
  const menu = page.getByRole('menu', { name: `Move ${name} to` });
  await expect(menu.getByRole('menuitem')).toHaveText(['Les Bleus', 'Reserve']);
  await menu.getByRole('menuitem', { name: 'Les Bleus' }).click();
  await expect(page.locator('.size-count')).toHaveText('7 v 4');
  await page.locator('.compose-col').nth(1).locator('.p-row', { hasText: name }).click();
  await expect(page.getByRole('menu', { name: `Move ${name} to` }).getByRole('menuitem')).toHaveText(['Les Rouges', 'Reserve']);
}

test.describe('design v2 admin editing', () => {
  test('sign-ups', async ({ page }) => {
    const { unstubbed } = await openEdit(page, EDIT);
    await expect(page.getByRole('button', { name: 'Reopen sign-ups' })).toBeVisible();
    await expect(page.getByText('Share on WhatsApp')).toBeVisible();
    // The section title says it; the rows carry only the name.
    await expect(page.getByText('Confirmed', { exact: true })).toHaveCount(1);
    await expect(page.getByText('Waiting list', { exact: true })).toHaveCount(1);
    await expect(page.locator('.sign-row .tog')).toHaveCount(0);
    await expect(page.locator('.sign-row').first()).toHaveText('Andrea Conti');
    await expect(page.getByRole('button', { name: 'Promote' })).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-edit-signups.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('sign-ups, dark', async ({ page }) => {
    const { unstubbed } = await openEdit(page, EDIT, { theme: 'dark' });
    await expect(page.locator('.design-v2.dark-mode')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Reopen sign-ups' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-signups-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('sign-ups @mobile', async ({ page }) => {
    const { unstubbed } = await openEdit(page, EDIT);
    await expect(page.getByRole('button', { name: 'Reopen sign-ups' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-signups-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  // The sign-up button and the places share one 44px line at every width,
  // 16px below the note. The label sits left of the field from 768px and
  // above it below; under 390px "Reopen sign-ups" reads "Reopen" but keeps its
  // full accessible name.
  for (const width of [1280, 768, 390, 360]) {
    test(`sign-ups, the controls on one row at ${width}px`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 });
      const { unstubbed } = await openEdit(page, EDIT);
      const status = page.getByRole('button', { name: 'Reopen sign-ups' });
      await expect(status).toBeVisible();
      // innerText, not the text content: the label not shown is still in the DOM.
      expect((await status.innerText()).trim()).toBe(width < 390 ? 'Reopen' : 'Reopen sign-ups');
      const places = page.locator('.places-row');
      await expect(places.locator('label')).toHaveText('Places');
      const field = page.getByRole('spinbutton', { name: 'Number of places' });
      await expect(field).toBeVisible();
      await expect(page.getByText('Confirmed places')).toHaveCount(0);
      const box = async (locator) => locator.evaluate((el) => {
        const r = el.getBoundingClientRect();
        return { top: r.top, bottom: r.bottom, left: r.left, right: r.right, height: r.height };
      });
      const [b, f, u, l, row, note] = await Promise.all([status, field, places.getByRole('button', { name: 'Update' }),
        places.locator('label'), page.locator('.signup-controls'), page.locator('.signup-controls').locator('xpath=preceding-sibling::p[1]')].map(box));
      for (const target of [b, f, u]) {
        expect(target.height).toBeGreaterThanOrEqual(44);
        expect(Math.abs((target.top + target.bottom) / 2 - (b.top + b.bottom) / 2)).toBeLessThanOrEqual(1);
      }
      expect(b.right).toBeLessThanOrEqual(f.left - 8);
      expect(f.right).toBeLessThanOrEqual(u.left);
      expect(row.top - note.bottom).toBeGreaterThanOrEqual(16);
      if (width >= 768) {
        expect(l.right).toBeLessThanOrEqual(f.left);
        // The status button keeps its own width, the places sit on the right.
        expect(b.right - b.left).toBeLessThan((row.right - row.left) / 2);
        expect(Math.abs(u.right - row.right)).toBeLessThanOrEqual(1);
      } else {
        expect(l.bottom).toBeLessThanOrEqual(f.top);
        expect(Math.abs(l.left - f.left)).toBeLessThanOrEqual(1);
        expect(f.right - f.left).toBe(64);
        expect(f.left - b.right).toBe(8);
      }
      await expectNoHorizontalOverflow(page);
      expectEverythingStubbed(unstubbed);
    });
  }

  test('teams, nobody placed', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=teams`);
    await expect(page.locator('.size-count')).toHaveText('0 v 0');
    await expect(page.locator('.compose-col input')).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Promote' })).toHaveCount(0);
    await expect(page.getByRole('heading', { name: 'Les Bleus' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-teams.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('teams, split automatically', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=teams`);
    await splitTeams(page);
    await expect(page.getByText('The teams are uneven')).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-edit-teams-auto.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('teams, a player picked', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=teams`);
    await pickUneven(page);
    await expect(page.getByText('The teams are uneven')).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-edit-teams-choice.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('teams, a player picked @mobile', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=teams`);
    await pickUneven(page);
    await expect(page.getByRole('menuitem', { name: 'Reserve' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-teams-choice-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  for (const [stage, todo] of [['open', null], ['closed', 'Teams'], ['teams saved', 'Score and goals'], ['score saved', null]]) {
    test(`steps, ${stage}`, async ({ page }) => {
      const { unstubbed } = await openEdit(page, EDIT, { match: STAGES[stage]() });
      const steps = page.getByRole('navigation', { name: 'Steps' }).getByRole('button');
      await expect(steps).toHaveCount(3);
      await expect(page.locator('.step-i.is-todo')).toHaveCount(todo ? 1 : 0);
      if (todo) await expect(page.locator('.step-i.is-todo')).toContainText(todo);
      // No second set of buttons to move on: the steps are the only way.
      await expect(page.getByRole('button', { name: 'Compose the teams' })).toHaveCount(0);
      await expect(page.getByRole('button', { name: 'Enter the score' })).toHaveCount(0);
      for (const box of await steps.evaluateAll((items) => items.map((item) => item.getBoundingClientRect().height))) {
        expect(box).toBeGreaterThanOrEqual(44);
      }
      expectEverythingStubbed(unstubbed);
    });
  }

  // The step on screen is outlined in the text colour; when it is also the
  // next one to do, it keeps the accent fill as well. Orange never outlines.
  for (const [stage, path, shown] of [['closed', EDIT, '1 Sign-ups'], ['closed', `${EDIT}?step=teams`, '2 Teams'], ['teams saved', `${EDIT}?step=score`, '3 Score and goals']]) {
    test(`steps, ${stage}, showing ${shown}`, async ({ page }) => {
      const { unstubbed } = await openEdit(page, path, { match: STAGES[stage]() });
      const current = page.getByRole('button', { name: shown });
      await expect(current).toHaveAttribute('aria-current', 'step');
      await expect(page.locator('.step-i[aria-current]')).toHaveCount(1);
      const looks = await page.locator('.step-i').evaluateAll((items) => {
        // The resolved text colour of the theme, to compare outlines with.
        const probe = document.createElement('span');
        probe.style.color = 'var(--v2-text)';
        items[0].parentElement.appendChild(probe);
        const text = getComputedStyle(probe).color;
        probe.remove();
        return items.map((item) => {
          const style = getComputedStyle(item);
          return { on: item.classList.contains('is-on'), todo: item.classList.contains('is-todo'), border: style.borderTopColor, color: style.color, text };
        });
      });
      for (const look of looks) {
        expect(look.border).not.toBe('rgb(232, 116, 59)');
        expect(look.color).not.toBe('rgb(232, 116, 59)');
        if (look.on) expect(look.border).toBe(look.text);
        else expect(look.border).not.toBe(look.text);
      }
      expect(looks.filter((look) => look.on && look.todo)).toHaveLength(shown === '1 Sign-ups' ? 0 : 1);
      expectEverythingStubbed(unstubbed);
    });
  }

  // Every step links back to the match page, with the same label and style.
  for (const [step, path] of [['sign-ups', EDIT], ['teams', `${EDIT}?step=teams`], ['score', `${EDIT}?step=score`]]) {
    test(`back link, ${step}`, async ({ page }) => {
      const { unstubbed } = await openEdit(page, path, { match: STAGES['teams saved']() });
      const back = page.locator('a.back');
      await expect(back).toHaveCount(1);
      await expect(back).toHaveText('Back to the match');
      await expect(back).toHaveAttribute('href', new RegExp(`/matches/${ID}$`));
      await expect(page.getByText('Back to editing')).toHaveCount(0);
      expectEverythingStubbed(unstubbed);
    });
  }

  // Teams opens once sign-ups are closed, Score and goals once the teams are
  // saved: until then they stay greyed, aria-disabled and inert.
  for (const [stage, match] of [
    ['not open yet', () => closedMatch({ RegistrationsClosedAt: undefined, RegistrationOpensAt: '2026-09-05T12:00:00+02:00' })],
    ['open', STAGES.open],
  ]) {
    test(`steps, sign-ups ${stage}, teams and score inactive`, async ({ page }) => {
      const { unstubbed } = await openEdit(page, EDIT, { match: match() });
      for (const name of ['2 Teams', '3 Score and goals']) {
        const item = page.getByRole('button', { name });
        await expect(item).toHaveAttribute('aria-disabled', 'true');
        await expect(item).toHaveClass(/is-inactive/);
        await item.click({ force: true });
        await expect(page).toHaveURL(new RegExp(`/matches/${ID}/edit$`));
      }
      await expect(page.locator('.step-i.is-todo')).toHaveCount(0);
      expectEverythingStubbed(unstubbed);
    });
  }

  test('steps, a link to a step not open yet shows the last open one', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=score`, { match: STAGES.open() });
    await expect(page.getByRole('button', { name: '1 Sign-ups' })).toHaveAttribute('aria-current', 'step');
    await expect(page).toHaveURL(new RegExp(`/matches/${ID}/edit$`));
    expectEverythingStubbed(unstubbed);
  });

  test('steps, score stays inactive while the teams are only drafted', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=teams`);
    await splitTeams(page);
    await expect(page.getByRole('button', { name: '3 Score and goals' })).toHaveAttribute('aria-disabled', 'true');
    expectEverythingStubbed(unstubbed);
  });

  test('steps, score not possible before the teams', async ({ page }) => {
    const { unstubbed } = await openEdit(page, EDIT);
    const score = page.getByRole('button', { name: '3 Score and goals' });
    await expect(score).toHaveAttribute('aria-disabled', 'true');
    // Playwright waits for an aria-disabled button to be enabled; force the
    // click to check it really does nothing.
    await score.click({ force: true });
    await expect(page).toHaveURL(new RegExp(`/matches/${ID}/edit$`));
    await page.getByRole('button', { name: '2 Teams' }).click();
    await expect(page).toHaveURL(/step=teams/);
    await expect(page.getByRole('button', { name: '2 Teams' })).toHaveAttribute('aria-current', 'step');
    expectEverythingStubbed(unstubbed);
  });

  test('teams, moving a player with the keyboard', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=teams`);
    const row = page.locator('.compose-col').first().locator('.p-row').first();
    await row.focus();
    await page.keyboard.press('Enter');
    const items = page.locator('.move-menu .move-opt');
    await expect(items).toHaveText(['Les Bleus', 'Les Rouges']);
    await expect(items.first()).toBeFocused();
    await page.keyboard.press('ArrowDown');
    await expect(items.nth(1)).toBeFocused();
    await page.keyboard.press('Enter');
    await expect(page.locator('.size-count')).toHaveText('0 v 1');
    await expect(page.locator('.compose-col').nth(2).locator('.p-row')).toBeFocused();
    await page.keyboard.press('Enter');
    await page.keyboard.press('Escape');
    await expect(page.locator('.move-menu')).toHaveCount(0);
    await expect(page.locator('.compose-col').nth(2).locator('.p-row')).toBeFocused();
    expectEverythingStubbed(unstubbed);
  });

  test('score', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=score`, { match: scoredMatch() });
    await expect(page.getByRole('button', { name: '+ Own goal' }).first()).toBeVisible();
    await expect(page.locator('.og-plain', { hasText: '2 OG' })).toBeVisible();
    await expect(page.locator('input[type="number"]')).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-edit-score.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('score, dark', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=score`, { theme: 'dark', match: scoredMatch() });
    await expect(page.locator('.og-plain', { hasText: '2 OG' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-score-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('score @mobile', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=score`, { match: scoredMatch() });
    await expect(page.locator('.og-plain', { hasText: '2 OG' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-score-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('own goal picker', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=score`, { match: scoredMatch() });
    await page.getByRole('button', { name: '+ Own goal' }).nth(1).click();
    await expect(page.locator('.csc-pop')).toBeVisible();
    await expect(page.locator('.csc-pop .csc-pick', { hasText: 'Giovanni Esposito' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-own-goal.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('own goal picker @mobile', async ({ page }) => {
    const { unstubbed } = await openEdit(page, `${EDIT}?step=score`, { match: scoredMatch() });
    await page.getByRole('button', { name: '+ Own goal' }).nth(1).click();
    await expect(page.locator('.csc-sheet')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-edit-own-goal-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });
});
