// The design-v2 screens photographed in every language (i18n.spec.js), with
// the fixtures each needs. One list, so the German baselines, the overflow
// check in all languages and any review capture walk the same pages.

const { expect } = require('@playwright/test');
const data = require('./data');

const V2 = { features: ['design-v2'] };

const editRegistrations = data.registrations.slice(0, 13).map((entry, index) => ({
  ...entry,
  Position: index + 1,
  IsWaiting: index >= 11,
}));
const editMatch = {
  ...data.scheduledClosedMatch,
  MaxPlayers: 11,
  Teams: data.scheduledClosedMatch.Teams.map((team) => ({ ...team, Players: [], Score: 0 })),
};

const stats = JSON.parse(JSON.stringify(data.playerStats));
stats.Seasons = ['2025-2026', '2026-2027'];
stats.Overall.PredictionPoints = 4.2;
stats.PerGroup[0].PredictionPoints = 5.1;
stats.PerGroup[1].PredictionPoints = -0.9;
stats.Email = 'thibaut@example.com';

const PASSKEYS = [
  { id: 'pk-1', name: 'iPhone', created_at: '2026-09-02T12:00:00Z', last_used_at: '2026-09-03T12:00:00Z' },
  { id: 'pk-2', name: 'MacBook Air', created_at: '2026-08-28T12:00:00Z', last_used_at: null },
];

const PREDICTIONS = {
  features: ['design-v2'],
  matches: data.predictionMatches,
  predictionSummaries: data.predictionSummaries,
  predictionStandings: data.predictionStandings,
};

const PROFILE = { features: ['design-v2', 'push-notifications'], playerStats: stats };

// name: file-name stem; ready: a locator that proves the page is drawn.
const SCREENS = [
  { name: 'home', path: '/', options: V2, ready: '.hero, .home-hero, main' },
  { name: 'match-open', path: `/matches/${data.scheduledOpenMatch.ID}`, options: V2 },
  { name: 'match-played', path: `/matches/${data.playedMatch.ID}`, options: V2 },
  { name: 'match-edit', path: `/matches/${data.scheduledClosedMatch.ID}/edit`, options: { ...V2, registrations: editRegistrations, matchDetails: { [editMatch.ID]: editMatch } } },
  { name: 'match-new', path: '/matches/new', options: V2 },
  { name: 'rankings-points', path: '/rankings', options: { features: ['design-v2', 'agoranking'] } },
  { name: 'rankings-scorers', path: '/rankings?tab=scorers', options: { features: ['design-v2', 'agoranking'] } },
  { name: 'rankings-mvp', path: '/rankings?tab=motm', options: { features: ['design-v2', 'agoranking'] } },
  { name: 'rankings-agoranking', path: '/rankings?tab=agoranking', options: { features: ['design-v2', 'agoranking'] } },
  { name: 'predictions-matches', path: '/predictions', options: PREDICTIONS },
  { name: 'predictions-leaderboard', path: '/predictions?tab=leaderboard', options: PREDICTIONS },
  { name: 'profile', path: '/profile', options: PROFILE },
  { name: 'profile-group', path: `/profile/groups/${data.GROUP_ID}`, options: PROFILE },
  { name: 'profile-notifications', path: '/profile/notifications', options: PROFILE },
  { name: 'profile-settings', path: `/profile/groups/${data.GROUP_ID}/settings`, options: PROFILE },
  { name: 'account', path: '/profile/account', options: { ...PROFILE, features: [...PROFILE.features, 'passkeys'], passkeys: PASSKEYS } },
];

// The public pages, signed out.
const PUBLIC_SCREENS = [
  { name: 'login', path: '/login' },
  { name: 'signup', path: '/signup' },
  { name: 'forgot-password', path: '/forgot-password' },
  { name: 'reset-password', path: '/reset-password?token=abc' },
];

// What overflows the page sideways, for a failure message that says where.
async function horizontalOverflow(page) {
  return page.evaluate(() => {
    const root = document.documentElement;
    const width = root.clientWidth;
    const offenders = [];
    if (root.scrollWidth > width) {
      document.querySelectorAll('body *').forEach((el) => {
        const box = el.getBoundingClientRect();
        if (box.width && box.right > width + 0.5) {
          const label = `${el.tagName.toLowerCase()}${el.className && typeof el.className === 'string' ? `.${el.className.trim().split(/\s+/).join('.')}` : ''}`;
          offenders.push(`${label} (right ${Math.round(box.right)} > ${width}) "${(el.textContent || '').trim().slice(0, 40)}"`);
        }
      });
    }
    return { scrollWidth: root.scrollWidth, clientWidth: width, offenders: offenders.slice(0, 8) };
  });
}

// Makes the viewport exactly as tall as the page, so a screenshot of it shows
// the page once, with the fixed bars (tab bar, Delete/Save bar) at the very
// bottom where a phone puts them. A fullPage screenshot instead stitches the
// page together and leaves those bars in the middle of the picture, hiding
// content. Measured from a short viewport, so a min-height of 100vh can't
// keep the page tall; repeated until the height stops changing.
async function fitViewportToPage(page) {
  const width = page.viewportSize().width;
  let height = 300;
  for (let pass = 0; pass < 6; pass += 1) {
    await page.setViewportSize({ width, height });
    await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    const measured = await page.evaluate(() => Math.ceil(document.documentElement.scrollHeight));
    if (measured === height) return height;
    height = measured;
  }
  return height;
}

async function expectNoOverflow(page, label) {
  const result = await horizontalOverflow(page);
  expect(result.scrollWidth, `${label}: horizontal overflow\n  ${result.offenders.join('\n  ')}`).toBeLessThanOrEqual(result.clientWidth);
}

module.exports = { SCREENS, PUBLIC_SCREENS, horizontalOverflow, expectNoOverflow, fitViewportToPage };
