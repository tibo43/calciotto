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
  // iPhone in a Safari tab: push only exists once the app is installed.
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
  return (await registration.pushManager.getSubscription()) ? PUSH_STATE.ON : PUSH_STATE.OFF;
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

// On logout: this device must stop receiving the account's notifications.
// Never throws, since logging out must not depend on it.
export async function disablePushOnLogout(win = window) {
  try {
    await disablePush(win);
  } catch (error) {
    console.error('Could not turn notifications off on logout:', error);
  }
}
