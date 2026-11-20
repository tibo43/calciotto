// Turning push notifications on and off for this device, for Profile.vue's
// "Notifications" card. The backend decides what is sent and when
// (app/internal/services/notifications.go); public/sw.js shows them.
//
// Every function takes the window so the tests can hand in a fake one.

import { getPushPublicKey, savePushSubscription, deletePushSubscription } from '@/services/api';
import { isIos, isStandalone } from '@/services/pwaInstall';

// The hidden-feature key (GET /players/me/features) that shows the card.
// Must match the backend's models.FeaturePushNotifications.
export const PUSH_FEATURE = 'push-notifications';

export const PUSH_STATE = {
  // No service worker or Push API: an old browser, the dev server (the
  // worker is only registered in production builds), or the visual tests.
  UNSUPPORTED: 'unsupported',
  // Not running as the installed app. Notifications are only offered there,
  // on every platform (an iPhone has no choice anyway: Safari tabs have no
  // push at all), so they behave the same everywhere and arrive as the app.
  NEEDS_INSTALL: 'needs-install',
  // The user refused, or blocked Calciotto in the phone's settings. Only
  // they can undo it, there; the page cannot ask again.
  DENIED: 'denied',
  OFF: 'off',
  ON: 'on'
};

// VAPID public keys travel as base64url; the Push API wants the raw bytes.
export function urlBase64ToUint8Array(base64) {
  const padded = (base64 + '='.repeat((4 - (base64.length % 4)) % 4)).replace(/-/g, '+').replace(/_/g, '/');
  const raw = atob(padded);
  return Uint8Array.from(raw, (char) => char.charCodeAt(0));
}

async function registrationOf(win) {
  if (!win.navigator.serviceWorker) return undefined;
  // getRegistration rather than `.ready`, which never settles when no worker
  // is registered (the dev server) and would leave the card loading forever.
  return win.navigator.serviceWorker.getRegistration();
}

export async function getPushState(win = window) {
  if (isIos(win.navigator) && !isStandalone(win)) {
    return PUSH_STATE.NEEDS_INSTALL;
  }
  if (!('PushManager' in win) || !('Notification' in win)) {
    return PUSH_STATE.UNSUPPORTED;
  }
  const registration = await registrationOf(win);
  if (!registration) {
    return PUSH_STATE.UNSUPPORTED;
  }
  if (win.Notification.permission === 'denied') {
    return PUSH_STATE.DENIED;
  }
  // A subscription made in the installed app is shared with the browser on
  // Android and desktop, so a tab can be "on" too: it is reported as such,
  // so the player can still turn it off from there. Only turning on is kept
  // for the installed app.
  if (await registration.pushManager.getSubscription()) {
    return PUSH_STATE.ON;
  }
  return isStandalone(win) ? PUSH_STATE.OFF : PUSH_STATE.NEEDS_INSTALL;
}

// Re-sends this device's subscription when it is on. The backend may have
// dropped it (it expired, or another account subscribed this browser since),
// and the upsert is harmless otherwise. Best effort: the card shows "on"
// either way, since the browser side is.
export async function resyncPushSubscription(win = window) {
  try {
    const registration = await registrationOf(win);
    const subscription = registration && await registration.pushManager.getSubscription();
    if (subscription) {
      await savePushSubscription(subscription.toJSON());
    }
  } catch (error) {
    console.error('Could not re-send the push subscription:', error);
  }
}

// Must run from a click: browsers only show the permission prompt for a user
// gesture (iOS refuses it otherwise).
export async function enablePush(win = window) {
  const permission = await win.Notification.requestPermission();
  if (permission !== 'granted') {
    return permission === 'denied' ? PUSH_STATE.DENIED : PUSH_STATE.OFF;
  }
  const publicKey = await getPushPublicKey();
  if (!publicKey) {
    throw new Error('Notifications are not available yet. Please try again later.');
  }
  const registration = await registrationOf(win);
  const subscription = await registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(publicKey)
  });
  await savePushSubscription(subscription.toJSON());
  return PUSH_STATE.ON;
}

// The backend first, then the browser: if the request fails, the device is
// still subscribed and the card still says so, rather than the backend
// keeping a subscription the device has thrown away.
export async function disablePush(win = window) {
  const registration = await registrationOf(win);
  const subscription = registration && await registration.pushManager.getSubscription();
  if (subscription) {
    await deletePushSubscription(subscription.endpoint);
    await subscription.unsubscribe();
  }
  return PUSH_STATE.OFF;
}

// Drops this browser's subscription without telling the backend: for when
// that isn't possible (no session any more, no network). The push service
// then answers the backend's next send with "gone", and the backend deletes
// its row on its own (NotificationService.send). Never throws.
export async function forgetPushOnThisBrowser(win = window) {
  try {
    const registration = await registrationOf(win);
    const subscription = registration && await registration.pushManager.getSubscription();
    if (subscription) {
      await subscription.unsubscribe();
    }
  } catch (error) {
    console.error('Could not drop the push subscription:', error);
  }
}

// On logout: this device must stop receiving the account's notifications.
// The backend is told when it can be; the browser side is dropped in every
// case, so a failed request (offline) can't leave the device subscribed.
// Never throws, since logging out must not depend on it.
export async function disablePushOnLogout(win = window) {
  try {
    await disablePush(win);
  } catch (error) {
    console.error('Could not turn notifications off on logout:', error);
    await forgetPushOnThisBrowser(win);
  }
}

// --- The app was uninstalled ------------------------------------------------
// On Android the subscription belongs to Chrome, not to the installed app, so
// uninstalling the app doesn't stop the notifications, and no event tells a
// site it was uninstalled. What Chrome can do is answer "is Calciotto
// installed?" (navigator.getInstalledRelatedApps, with the manifest listing
// the app as its own related application). That only runs when the site is
// opened, so this catches a player who uninstalls and comes back in a tab;
// one who never comes back keeps receiving them through Chrome.
//
// The answer is only trusted once it has been seen to work on this device:
// opened as the installed app, the check must say "installed", and that is
// recorded. Without that proof (another browser, a version that doesn't
// support it), an empty answer could just mean "can't tell", and turning
// notifications off on a guess would be worse than leaving them on.
const DETECTION_KEY = 'calciotto-installed-app-detection';

async function relatedAppInstalled(win) {
  if (typeof win.navigator.getInstalledRelatedApps !== 'function') {
    return null;
  }
  try {
    const apps = await win.navigator.getInstalledRelatedApps();
    return Array.isArray(apps) ? apps.length > 0 : null;
  } catch {
    return null;
  }
}

function detectionProven(win) {
  try {
    return win.localStorage.getItem(DETECTION_KEY) === 'works';
  } catch {
    return false;
  }
}

function recordDetectionWorks(win) {
  try {
    win.localStorage.setItem(DETECTION_KEY, 'works');
  } catch {
    // Private mode or blocked storage: the check just stays inactive.
  }
}

// Called once per page load (App.vue). Resolves to true when it turned
// notifications off because the app is no longer installed.
export async function stopPushIfAppUninstalled(win = window) {
  const installed = await relatedAppInstalled(win);
  if (installed === null) {
    return false;
  }
  if (isStandalone(win)) {
    if (installed) {
      recordDetectionWorks(win);
    }
    return false;
  }
  if (installed || !detectionProven(win)) {
    return false;
  }
  const registration = await registrationOf(win);
  const subscription = registration && await registration.pushManager.getSubscription();
  if (!subscription) {
    return false;
  }
  await disablePushOnLogout(win);
  return true;
}
