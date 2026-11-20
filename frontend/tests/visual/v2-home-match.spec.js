// Home and match detail with the design-v2 feature on. The flag stays off in
// every other spec, so those baselines keep photographing today's screens.

const { test, expect } = require('@playwright/test');
const data = require('./fixtures/data');
const { gotoApp, expectEverythingStubbed, expectNoHorizontalOverflow } = require('./fixtures/app');

const V2 = { features: ['design-v2'] };
const OPEN = data.scheduledOpenMatch.ID;
const PLAYED = data.playedMatch.ID;

test.describe('design v2 home and match', () => {
  test('home, signed up', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', V2);
    await expect(page.getByText("You're signed up")).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('home, signed up, dark', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', { ...V2, theme: 'dark' });
    await expect(page.locator('.design-v2.dark-mode')).toBeVisible();
    await expect(page.getByText("You're signed up")).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('home, signed up @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', V2);
    await expect(page.getByText("You're signed up")).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('home, signed up, dark @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', { ...V2, theme: 'dark' });
    await expect(page.getByText("You're signed up")).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home-dark-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('home, withdraw confirmation', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', V2);
    await page.getByRole('button', { name: 'Withdraw', exact: true }).click();
    await expect(page.getByRole('dialog').getByText('Withdraw from this match?')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home-withdraw.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('upcoming match, waiting list', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${OPEN}`, V2);
    await expect(page.getByRole('heading', { name: 'Waiting list' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Manage match' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-upcoming.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('upcoming match, waiting list, dark', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${OPEN}`, { ...V2, theme: 'dark' });
    await expect(page.getByRole('heading', { name: 'Waiting list' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-upcoming-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('upcoming match, waiting list @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${OPEN}`, V2);
    await expect(page.getByRole('heading', { name: 'Waiting list' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-upcoming-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('played match, vote closed', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${PLAYED}`, V2);
    await expect(page.getByText('Star of the Match vote closed', { exact: true })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-played.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  // Group admins only: each player's preferred positions after the name.
  const matchPositions = [
    [1, ['FWD']], [2, ['GK']], [3, ['DEF', 'MID']], [4, ['GK', 'DEF']],
    [5, ['MID', 'FWD']], [6, ['FWD']], [7, ['DEF']],
  ].map(([n, positions]) => ({ PlayerID: `bbbbbbbb-0000-4000-8000-0000000000${String(n).padStart(2, '0')}`, PreferredPositions: positions }));

  test('played match, positions for an admin', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${PLAYED}`, { ...V2, matchPositions });
    await expect(page.locator('.player', { hasText: 'Thibaut Fabre' }).locator('.pos-tag')).toHaveText('DEF · MID');
    await expect(page.locator('.player', { hasText: 'Alessandro Moretti' }).locator('.pos-tag')).toHaveCount(0);
    await expect(page).toHaveScreenshot('v2-match-played-positions.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('played match, positions for an admin @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${PLAYED}`, { ...V2, matchPositions, theme: 'dark' });
    await expect(page.locator('.pos-tag')).toHaveCount(7);
    await expect(page).toHaveScreenshot('v2-match-played-positions-mobile.png', { fullPage: true });
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('played match, no positions for a member', async ({ page }) => {
    const asked = [];
    page.on('request', (request) => { if (request.url().endsWith('/positions')) asked.push(request.url()); });
    const groups = data.groups.map((group) => ({ ...group, role: 'member' }));
    const { unstubbed } = await gotoApp(page, `/matches/${PLAYED}`, { ...V2, groups, matchPositions });
    await expect(page.getByText('Star of the Match vote closed', { exact: true })).toBeVisible();
    await expect(page.locator('.pos-tag')).toHaveCount(0);
    expect(asked).toEqual([]);
    expectEverythingStubbed(unstubbed);
  });

  test('played match, vote closed, dark', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${PLAYED}`, { ...V2, theme: 'dark' });
    await expect(page.getByText('Star of the Match vote closed', { exact: true })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-played-dark.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });
});

// The same match once the admin has set the teams, with the caller on one of
// them: their side instead of the places, both line-ups, and the prediction
// bar pinned above the tab bar.
const ME = '33333333-3333-4333-8333-333333333333';
const member = (n, name) => ({ ID: n === 'me' ? ME : `bbbbbbbb-0000-4000-8000-0000000000${n}`, Name: name, GoalNumber: 0 });
const teamsSet = {
  ...data.scheduledOpenMatch,
  RegistrationsClosedAt: '2026-09-04T09:00:00+02:00',
  Teams: [
    { ...data.playedMatch.Teams[0], Score: 0, Players: [member('01', 'Marco Rossi'), member('me', 'Thibaut Fabre'), member('02', 'Luca Bianchi'), member('04', 'Giovanni Esposito')] },
    { ...data.playedMatch.Teams[1], Score: 0, Players: [member('05', 'Andrea Conti'), member('06', 'Matteo Ricci'), member('07', 'Federico Greco'), member('08', 'Alessandro Moretti')] },
  ],
};
const TEAMS_SET = {
  ...V2,
  matches: data.matches.map((match) => (match.ID === teamsSet.ID ? teamsSet : match)),
  predictionSummaries: { [teamsSet.ID]: { Entries: [], Count: 5, Revealed: false, Odds: { TeamAOdds: 1.85, DrawOdds: 3.4, TeamBOdds: 2.1 } } },
};

test.describe('design v2 once the teams are set', () => {
  test('upcoming match, teams set, prediction bar @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${OPEN}`, TEAMS_SET);
    await expect(page.getByText('Your prediction')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-teams-set-mobile.png');
    await expectNoHorizontalOverflow(page);
    expectEverythingStubbed(unstubbed);
  });

  test('upcoming match, teams set, prediction bar', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${OPEN}`, TEAMS_SET);
    await expect(page.getByText('Your prediction')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-match-teams-set.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  test('home, teams set @mobile', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', TEAMS_SET);
    await expect(page.getByText('You play for')).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home-teams-set-mobile.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });

  // Every place taken and the caller not signed up: the waiting list.
  test('home, list full @mobile', async ({ page }) => {
    const notMe = data.registrations.map((entry) => (entry.PlayerID === ME ? { ...entry, PlayerID: 'eeeeeeee-0000-4000-8000-000000000001' } : entry));
    const { unstubbed } = await gotoApp(page, '/', { ...V2, registrations: notMe });
    await expect(page.getByRole('button', { name: 'Join the waiting list' })).toBeVisible();
    await expect(page).toHaveScreenshot('v2-home-full-mobile.png', { fullPage: true });
    expectEverythingStubbed(unstubbed);
  });
});

// A match is over as soon as its score is recorded, here on its own day
// (kick-off 08:00, the clock at 10:00). It keeps the home header, as "Last
// match" with its score, until the next match is created; then it heads
// Latest matches. Its page shows the result with nothing about sign-ups.
const recorded = {
  ...data.playedMatch, ID: 'cccccccc-0000-4000-8000-0000000000f1', Date: '2026-09-04', ScheduledAt: '2026-09-04T08:00:00+02:00',
  RegistrationOpensAt: '2026-09-01T18:00:00+02:00', RegistrationsClosedAt: '2026-09-04T07:00:00+02:00',
  ResultRecordedAt: '2026-09-04T09:30:00+02:00', MaxPlayers: 8,
};
const NAV = '.nav-text, .nav-ico';

test.describe('design v2 once the score is recorded', () => {
  test('home, the last match keeps the header, as an admin', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', { ...V2, matches: [...data.matches.slice(0, 2), recorded] });
    const hero = page.locator('.hero-last');
    await expect(hero.locator('.label')).toHaveText('Last match');
    await expect(hero.locator('.hero-date')).toHaveText('Fri 4 Sep');
    await expect(hero.locator('.board-score')).toHaveText('4–6');
    await expect(hero.getByRole('link', { name: 'Manage match' })).toHaveAttribute('href', `/matches/${recorded.ID}/edit`);
    await expect(hero).not.toContainText(/Sign up|Withdraw|Sign-ups|places/);
    // Never listed twice: Latest matches starts with the one before.
    await expect(page.locator(`.list .row[href="/matches/${recorded.ID}"]`)).toHaveCount(0);
    await expect(page.locator('.list .row').first()).toHaveAttribute('href', `/matches/${data.matches[0].ID}`);
    // Its vote is open: the MOTM line points at it.
    await expect(page.locator('.motm-row')).toHaveAttribute('href', `/matches/${recorded.ID}`);
    await expect(page.getByRole('link', { name: 'New match' })).toBeVisible();
    expectEverythingStubbed(unstubbed);
  });

  test('home, the next scheduled match after it', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', { ...V2, matches: [...data.matches, recorded] });
    await expect(page.locator('.hero .label').first()).toHaveText('Next match');
    await expect(page.locator('.hero .hero-date')).toHaveText('Sun 6 Sep');
    await expect(page.locator('.hero-last')).toHaveCount(0);
    await expect(page.locator('.list .row').first()).toHaveAttribute('href', `/matches/${recorded.ID}`);
    // The vote of a match no longer in the header has no line under it...
    await expect(page.locator('.motm-row')).toHaveCount(0);
    // ...but its row says so, in grey in place of its kick-off time, and
    // leads to it.
    const row = page.locator(`.list .row[href="/matches/${recorded.ID}"]`);
    await expect(row.locator('.row-sub')).toHaveText('Star of the Match vote open');
    await expect(row).not.toContainText('08:00');
    const vote = row.locator('.row-sub-vote');
    const [colour, muted] = await vote.evaluate((node) => {
      const probe = document.createElement('span');
      probe.style.color = 'var(--v2-text-muted)';
      node.appendChild(probe);
      const result = [getComputedStyle(node).color, getComputedStyle(probe).color];
      probe.remove();
      return result;
    });
    expect(colour).toBe(muted);
    await expect(page.locator('.list .row-sub-vote')).toHaveCount(1);
    await vote.click();
    await expect(page).toHaveURL(new RegExp(`/matches/${recorded.ID}$`));
    expectEverythingStubbed(unstubbed);
  });

  test('home, the match button alone on the right, no sign-up status line', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', V2);
    await expect(page.locator('.hero-status .hero-meta')).toHaveCount(0);
    await expect(page.locator('.hero-status .hero-status-btn')).toHaveText('View match');
    await expect(page.getByRole('link', { name: 'Manage match' })).toBeVisible();
    const group = await page.locator('.hero-status').boundingBox();
    const hero = await page.locator('.hero').boundingBox();
    expect(Math.abs(group.x + group.width - (hero.x + hero.width))).toBeLessThanOrEqual(1);
    expectEverythingStubbed(unstubbed);
  });

  test('match page, the result and no sign-ups', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${recorded.ID}`, { ...V2, matches: [...data.matches, recorded] });
    await expect(page.locator('.board .board-score')).toHaveText('4–6');
    await expect(page.locator('.motm-line')).toBeVisible();
    await expect(page.locator('main.page')).not.toContainText(/Sign up|Withdraw|Sign-ups|places left/);
    expectEverythingStubbed(unstubbed);
  });
});

// Only a tab's own page highlights its entry in the menu.
test.describe('design v2 menu', () => {
  for (const [name, path] of [['match page', `/matches/${PLAYED}`], ['edit page', `/matches/${data.scheduledClosedMatch.ID}/edit`], ['new match', '/matches/new']]) {
    test(`${name}, no tab highlighted`, async ({ page }) => {
      const closed = data.scheduledClosedMatch;
      const { unstubbed } = await gotoApp(page, path, { ...V2, matchDetails: { [closed.ID]: closed } });
      await expect(page).toHaveURL(new RegExp(`${path}$`));
      await expect(page.locator('main.page')).toBeVisible();
      await expect(page.locator(`${NAV}`).first()).toBeAttached();
      await expect(page.locator(`:is(${NAV}).is-on`)).toHaveCount(0);
      await expect(page.locator(`:is(${NAV})[aria-current="page"]`)).toHaveCount(0);
      expectEverythingStubbed(unstubbed);
    });
  }

  test('home, Matches highlighted', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', V2);
    await expect(page.locator('.hero')).toBeVisible();
    const on = page.locator(`:is(${NAV}).is-on`);
    await expect(on.first()).toHaveAttribute('aria-current', 'page');
    for (const label of await on.evaluateAll((items) => items.map((item) => item.getAttribute('aria-label') || item.textContent.trim()))) {
      expect(label).toBe('Matches');
    }
    expectEverythingStubbed(unstubbed);
  });
});

// The MOTM counts: muted for everyone while the vote is open (the one ahead
// too), the accent only on the winner once it is closed.
test.describe('design v2 MOTM counts', () => {
  const colours = (page) => page.locator('.player').evaluateAll((rows) => {
    const probe = document.createElement('span');
    probe.style.color = 'var(--v2-text-muted)';
    rows[0].appendChild(probe);
    const muted = getComputedStyle(probe).color;
    probe.remove();
    return rows.map((row) => ({ name: row.querySelector('.nm').textContent.trim(), votes: getComputedStyle(row.querySelector('.votes')).color, star: getComputedStyle(row.querySelector('.votes .star')).color, muted }));
  });

  for (const theme of ['light', 'dark']) {
    test(`vote open, every count muted, ${theme}`, async ({ page }) => {
      const { unstubbed } = await gotoApp(page, `/matches/${recorded.ID}`, { ...V2, theme, matches: [...data.matches, recorded] });
      await expect(page.locator('.motm-line')).toContainText('Star of the Match vote open');
      for (const row of await colours(page)) {
        expect(row.votes).toBe(row.muted);
        expect(row.star).toBe(row.muted);
      }
      expectEverythingStubbed(unstubbed);
    });
  }

  test('vote closed, the winner in the accent', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${PLAYED}`, V2);
    await expect(page.getByText('Star of the Match vote closed', { exact: true })).toBeVisible();
    const rows = await colours(page);
    for (const row of rows) {
      if (row.name === 'Marco Rossi') expect(row.votes).not.toBe(row.muted);
      else expect(row.votes).toBe(row.muted);
    }
    expectEverythingStubbed(unstubbed);
  });
});

test.describe('design v2 home spacing', () => {
  // The MOTM line belongs to the header: 16px under the score (under the
  // match button on a phone), 16px to the rule, and the header 16px over
  // Latest matches.
  for (const tag of ['', ' @mobile']) test(`the MOTM line in the header, 16px above and below${tag}`, async ({ page }) => {
    const { unstubbed } = await gotoApp(page, '/', { ...V2, matches: [...data.matches.slice(0, 2), recorded] });
    const hero = page.locator('.hero-last');
    const line = hero.locator('.hero-motm');
    await expect(line).toHaveAttribute('href', `/matches/${recorded.ID}`);
    const box = await line.boundingBox();
    const heroBox = await hero.boundingBox();
    const above = await Promise.all([hero.locator('.board-score'), hero.locator('.hero-status-btn')].map((item) => item.boundingBox()));
    const lowest = Math.max(...above.map((item) => item.y + item.height));
    expect(Math.round(box.y - lowest)).toBe(16);
    expect(Math.round(heroBox.y + heroBox.height - 1 - (box.y + box.height))).toBe(16);
    const heading = await page.locator('.sec').filter({ hasText: 'Latest matches' }).boundingBox();
    expect(Math.round(heading.y - (heroBox.y + heroBox.height))).toBe(16);
    expectEverythingStubbed(unstubbed);
  });

  test('web, the match button on the date line, centred on it', async ({ page }) => {
    test.skip(page.viewportSize().width < 900, 'web only');
    const { unstubbed } = await gotoApp(page, '/', V2);
    const date = await page.locator('.hero-date').boundingBox();
    const group = await page.locator('.hero-status').boundingBox();
    const hero = await page.locator('.hero').boundingBox();
    expect(Math.abs((group.y + group.height / 2) - (date.y + date.height / 2))).toBeLessThanOrEqual(1);
    expect(Math.abs(group.x + group.width - (hero.x + hero.width))).toBeLessThanOrEqual(1);
    expectEverythingStubbed(unstubbed);
  });
});

// Latest matches rows: the teams with « vs » in the middle, the score only
// on the right; the name still reads the score with the day. Web: « A vs B »
// on one line when it fits. Phones: one team per line, « A vs » then « B ».
// Never cut, never under the score, whatever the length of the names.
const rowsOf = (page) => page.locator('.list .row').evaluateAll((rows) => rows.map((row) => {
  const rect = (el) => { const r = el.getBoundingClientRect(); return { l: r.left, r: r.right, t: r.top, b: r.bottom }; };
  const textRect = (el) => {
    const range = document.createRange();
    range.selectNodeContents(el);
    const all = [...range.getClientRects()];
    return { l: Math.min(...all.map((x) => x.left)), r: Math.max(...all.map((x) => x.right)), t: Math.min(...all.map((x) => x.top)), b: Math.max(...all.map((x) => x.bottom)) };
  };
  const hits = (a, b) => a.l < b.r && a.r > b.l && a.t < b.b && a.b > b.t;
  const title = row.querySelector('.row-title');
  const score = rect(row.querySelector('.score'));
  const sides = [...row.querySelectorAll('.row-side')].map(rect);
  return {
    title: title.textContent.replace(/\s+/g, ' ').trim(), label: row.getAttribute('aria-label'), score: row.querySelector('.score').innerText.trim(),
    cut: [...row.querySelectorAll('.row-side, .row-title')].some((el) => el.scrollWidth > el.clientWidth + 0.5 || getComputedStyle(el).textOverflow === 'ellipsis'),
    underScore: [...row.querySelectorAll('.row-team')].some((el) => hits(textRect(el), score)),
    sameLine: sides.length === 2 && Math.abs(sides[0].t - sides[1].t) < 1,
    scoreCentred: Math.abs((score.t + score.b) / 2 - (rect(row).t + rect(row).b) / 2) <= 2,
  };
}));
test.describe('design v2 Latest matches rows', () => {
  for (const [tag, width] of [['', 1280], [' @mobile', 390], [' @mobile', 360]]) {
    test(`the score once, on the right, ${width}px${tag}`, async ({ page }) => {
      await page.setViewportSize({ width, height: 844 });
      const { unstubbed } = await gotoApp(page, '/', { ...V2, matches: [...data.matches, recorded] });
      const rows = await rowsOf(page);
      expect(rows[0]).toMatchObject({ title: 'Les Bleus vs Les Rouges', label: 'Les Bleus 4–6 Les Rouges, 4 Sep', score: '4–6', cut: false, underScore: false });
      for (const row of rows) {
        expect(row.title).not.toMatch(/\d/);
        expect(row.cut).toBe(false);
        // One line on the web, one team per line on phones.
        expect(row.sameLine).toBe(width >= 600);
      }
      expectEverythingStubbed(unstubbed);
    });
  }

  test('match page, the score only in the score block', async ({ page }) => {
    const { unstubbed } = await gotoApp(page, `/matches/${recorded.ID}`, { ...V2, matches: [...data.matches, recorded] });
    await expect(page.locator('.board-score')).toHaveText('4–6');
    expect((await page.locator('main.page').innerText()).match(/\d+\s*–\s*\d+/g)).toEqual(['4–6']);
    expectEverythingStubbed(unstubbed);
  });
});

// Real team names from a club (37 and 19 characters) and two ~20-character
// ones: the rows wrap and grow, the header names stay in their column, on at
// most three lines, one size smaller beyond two; nothing is cut, nothing
// overlaps the score, nothing overflows the page.
const LONG_NAMES = [['Association Sportive de Saint Etienne', 'Paris Saint-Germain'], ['Real Sociedad Fútbol', 'Borussia Mönchengladb']];
const named = (match, [a, b]) => ({ ...match, Teams: match.Teams.map((team, i) => ({ ...team, Name: i === 0 ? a : b })) });
test.describe('design v2 long team names', () => {
  for (const [tag, width] of [['', 1280], [' @mobile', 390], [' @mobile', 360]]) {
    for (const names of LONG_NAMES) {
      test(`Latest matches rows and header, ${names[0]}, ${width}px${tag}`, async ({ page }) => {
        await page.setViewportSize({ width, height: 844 });
        const matches = [...data.matches.slice(0, 2), recorded].map((match) => named(match, names));
        const { unstubbed } = await gotoApp(page, '/', { ...V2, matches });
        await page.evaluate(() => document.fonts.ready);
        for (const row of await rowsOf(page)) {
          expect(row).toMatchObject({ cut: false, underScore: false, scoreCentred: true });
          if (width < 600) expect(row.sameLine).toBe(false);
        }
        const header = await page.locator('.hero-last .board-teams').evaluate((board) => {
          const rect = (el) => { const r = el.getBoundingClientRect(); return { l: r.left, r: r.right, t: r.top, b: r.bottom }; };
          const score = rect(board.querySelector('.board-score'));
          return [...board.querySelectorAll('.board-team .name')].map((name) => {
            const range = document.createRange();
            range.selectNodeContents(name);
            const all = [...range.getClientRects()];
            const text = { l: Math.min(...all.map((x) => x.left)), r: Math.max(...all.map((x) => x.right)), t: Math.min(...all.map((x) => x.top)), b: Math.max(...all.map((x) => x.bottom)) };
            const col = rect(name.closest('.board-team'));
            return {
              lines: Math.round(name.getBoundingClientRect().height / parseFloat(getComputedStyle(name).lineHeight)),
              inColumn: text.l >= col.l - 1 && text.r <= col.r + 1,
              overScore: text.l < score.r && text.r > score.l && text.t < score.b && text.b > score.t,
              colWidth: Math.round(col.r - col.l),
              font: getComputedStyle(name).fontSize,
              bottom: text.b,
              colLeft: col.l,
              colRight: col.r,
            };
          });
        });
        expect(header[0].colWidth).toBe(header[1].colWidth);
        // One size for both names.
        expect(header[0].font).toBe(header[1].font);
        if (width < 700) {
          // Phones: both names sit on the score, 16px or more apart.
          expect(Math.abs(header[0].bottom - header[1].bottom)).toBeLessThanOrEqual(1);
          expect(header[1].colLeft - header[0].colRight).toBeGreaterThanOrEqual(16);
        }
        for (const name of header) {
          expect(name).toMatchObject({ inColumn: true, overScore: false });
          expect(name.lines).toBeLessThanOrEqual(3);
        }
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
        expectEverythingStubbed(unstubbed);
      });
    }
  }
});
