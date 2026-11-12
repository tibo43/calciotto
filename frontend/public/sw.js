// Calciotto's service worker. Deliberately empty of any caching: it exists so
// the app is installable and so it can receive push notifications — not to
// work offline. A worker that caches the app shell is the classic cause of
// "the app never updates", and nothing here needs offline use.
//
// No `fetch` handler either: without one, every request goes straight to the
// network exactly as it would with no worker at all.

// Take over as soon as a new version is installed, instead of waiting for
// every tab of the old one to close: there is no cached state to migrate.
self.addEventListener('install', () => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim());
});

// ---------------------------------------------------------------------------
// Push notifications. The backend (services/notifications.go) sends data, not
// text: the wording lives here so that a kick-off time is written in the
// phone's own time zone, the way the rest of the app shows it
// (src/services/datetime.js, same 'en-US' format).

function formatKickoff(iso) {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleString('en-US', {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
}

function formatTime(iso) {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
}

function formatScore(score) {
  return (score || []).map((line) => `${line.team} ${line.goals}`).join(' – ');
}

// Signed, two decimals — the same rounding a settled prediction row shows
// (PredictionsPanel.formatPoints). A miss is −1.00; a hit is the odds minus
// the one implicit token.
function formatSignedPoints(points) {
  const value = Number(points);
  if (!Number.isFinite(value)) return '0.00';
  return `${value > 0 ? '+' : ''}${value.toFixed(2)}`;
}

// Title and body for one push payload. Unknown types still show something
// rather than nothing, since a push the user can't see is a wasted one.
function buildNotification(data) {
  const kickoff = data.kickoffAt ? formatKickoff(data.kickoffAt) : '';
  const prefix = data.groupName ? `${data.groupName}: ` : '';
  switch (data.type) {
    case 'registration_open':
      return {
        title: 'Sign-ups are open',
        body: `${prefix}match on ${kickoff}. Grab your spot.`,
      };
    case 'team_assigned':
      return {
        title: `You're playing for ${data.teamName}`,
        body: `${prefix}match on ${kickoff}.`,
      };
    case 'predictions_open':
      return {
        title: 'Predictions are open',
        body: `${prefix}match on ${kickoff}. Make your pick.`,
      };
    case 'kickoff_reminder':
      return {
        title: 'Match starting soon',
        body: `Kick-off at ${formatTime(data.kickoffAt)}` + (data.teamName ? `, you're with ${data.teamName}.` : '.'),
      };
    case 'result_recorded': {
      const score = formatScore(data.score);
      return {
        title: score ? `Final score: ${score}` : 'The result is in',
        body: 'Vote for your Man of the Match.',
      };
    }
    case 'predictions_settled': {
      const score = formatScore(data.score);
      const earned = `You earned ${formatSignedPoints(data.points)} points.`;
      return {
        title: 'Predictions settled',
        body: score ? `Final score: ${score}. ${earned}` : earned,
      };
    }
    default:
      return { title: 'Calciotto', body: prefix ? prefix.slice(0, -2) : 'Something new in your group.' };
  }
}

self.addEventListener('push', (event) => {
  let data = {};
  try {
    data = event.data ? event.data.json() : {};
  } catch (error) {
    data = {};
  }
  const { title, body } = buildNotification(data);
  event.waitUntil(self.registration.showNotification(title, {
    body,
    icon: '/icons/icon-192.png',
    // The same tag replaces the previous notification of that kind for that
    // match (e.g. a team change after a first assignment) instead of stacking.
    tag: data.tag,
    data: { url: data.url || '/' },
  }));
});

// Tapping a notification focuses an open Calciotto window and sends it to the
// match, or opens one.
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const url = new URL((event.notification.data && event.notification.data.url) || '/', self.location.origin).href;
  event.waitUntil((async () => {
    const windows = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
    for (const client of windows) {
      if (new URL(client.url).origin === self.location.origin) {
        await client.focus();
        // navigate() only works on a window this worker controls; any
        // other gets a fresh window instead.
        try {
          return await client.navigate(url);
        } catch (error) {
          break;
        }
      }
    }
    return self.clients.openWindow(url);
  })());
});

// Lets the unit tests import the wording without a service worker around.
if (typeof module === 'object' && module.exports) {
  module.exports = { buildNotification };
}
