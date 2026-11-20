import {
  PUSH_STATE, getPushState, enablePush, disablePush, urlBase64ToUint8Array,
  disablePushOnLogout, forgetPushOnThisBrowser, stopPushIfAppUninstalled
} from '@/services/pushNotifications';
import { getPushPublicKey, savePushSubscription, deletePushSubscription } from '@/services/api';

vi.mock('@/services/api', () => ({
  getPushPublicKey: vi.fn(),
  savePushSubscription: vi.fn(),
  deletePushSubscription: vi.fn()
}));

const ANDROID_UA = 'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Mobile Safari/537.36';
const IPHONE_UA = 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1';

// A window with a registered worker whose push subscription the test controls.
function fakeWindow({
  userAgent = ANDROID_UA, standalone = false, permission = 'default', subscription = null, registered = true, push = true,
  relatedApps = undefined, storage = {}
} = {}) {
  const pushManager = {
    getSubscription: vi.fn(() => Promise.resolve(subscription)),
    subscribe: vi.fn(() => Promise.resolve({
      endpoint: 'https://push.test/new',
      toJSON: () => ({ endpoint: 'https://push.test/new', keys: { p256dh: 'k', auth: 'a' } })
    }))
  };
  const win = {
    matchMedia: () => ({ matches: standalone }),
    navigator: {
      userAgent,
      maxTouchPoints: 0,
      serviceWorker: { getRegistration: () => Promise.resolve(registered ? { pushManager } : undefined) }
    },
    Notification: { permission, requestPermission: vi.fn(() => Promise.resolve(permission)) },
    localStorage: {
      getItem: (key) => (key in storage ? storage[key] : null),
      setItem: (key, value) => { storage[key] = value; }
    }
  };
  // undefined: the browser has no getInstalledRelatedApps at all.
  if (relatedApps !== undefined) {
    win.navigator.getInstalledRelatedApps = vi.fn(() => Promise.resolve(relatedApps));
  }
  if (push) win.PushManager = function PushManager() {};
  return { win, pushManager };
}

beforeEach(() => vi.clearAllMocks());

describe('getPushState', () => {
  it('asks iPhone users in a Safari tab to install the app first', async () => {
    expect(await getPushState(fakeWindow({ userAgent: IPHONE_UA, push: false }).win)).toBe(PUSH_STATE.NEEDS_INSTALL);
  });

  it('works on an installed iPhone app', async () => {
    expect(await getPushState(fakeWindow({ userAgent: IPHONE_UA, standalone: true }).win)).toBe(PUSH_STATE.OFF);
  });

  it('is unsupported without the Push API or without a registered worker', async () => {
    expect(await getPushState(fakeWindow({ push: false }).win)).toBe(PUSH_STATE.UNSUPPORTED);
    expect(await getPushState(fakeWindow({ registered: false }).win)).toBe(PUSH_STATE.UNSUPPORTED);
  });

  it('reports a blocked permission, then on or off from the subscription', async () => {
    expect(await getPushState(fakeWindow({ permission: 'denied' }).win)).toBe(PUSH_STATE.DENIED);
    expect(await getPushState(fakeWindow({ standalone: true, subscription: { endpoint: 'x' } }).win)).toBe(PUSH_STATE.ON);
    expect(await getPushState(fakeWindow({ standalone: true }).win)).toBe(PUSH_STATE.OFF);
  });

  it('only offers turning them on in the installed app, on Android and desktop too', async () => {
    expect(await getPushState(fakeWindow().win)).toBe(PUSH_STATE.NEEDS_INSTALL);
  });

  it('still shows them on in a tab when the installed app subscribed, so they can be turned off', async () => {
    expect(await getPushState(fakeWindow({ subscription: { endpoint: 'x' } }).win)).toBe(PUSH_STATE.ON);
  });
});

describe('enablePush', () => {
  it('subscribes with the server key and sends the subscription to the backend', async () => {
    getPushPublicKey.mockResolvedValue('AQID');
    const { win, pushManager } = fakeWindow({ permission: 'granted' });
    await expect(enablePush(win)).resolves.toBe(PUSH_STATE.ON);
    expect(pushManager.subscribe).toHaveBeenCalledWith({ userVisibleOnly: true, applicationServerKey: new Uint8Array([1, 2, 3]) });
    expect(savePushSubscription).toHaveBeenCalledWith({ endpoint: 'https://push.test/new', keys: { p256dh: 'k', auth: 'a' } });
  });

  it('stops at a refused permission without subscribing', async () => {
    const { win, pushManager } = fakeWindow({ permission: 'denied' });
    await expect(enablePush(win)).resolves.toBe(PUSH_STATE.DENIED);
    expect(pushManager.subscribe).not.toHaveBeenCalled();
  });

  it('fails clearly when the backend has no key configured', async () => {
    getPushPublicKey.mockResolvedValue('');
    await expect(enablePush(fakeWindow({ permission: 'granted' }).win)).rejects.toThrow('not available');
  });
});

describe('disablePush', () => {
  it('tells the backend first, then drops the browser subscription', async () => {
    const order = [];
    deletePushSubscription.mockImplementation(() => { order.push('backend'); return Promise.resolve(); });
    const subscription = { endpoint: 'https://push.test/old', unsubscribe: vi.fn(() => { order.push('browser'); return Promise.resolve(true); }) };
    await expect(disablePush(fakeWindow({ subscription }).win)).resolves.toBe(PUSH_STATE.OFF);
    expect(deletePushSubscription).toHaveBeenCalledWith('https://push.test/old');
    expect(order).toEqual(['backend', 'browser']);
  });
});

describe('urlBase64ToUint8Array', () => {
  it('decodes base64url without padding', () => {
    expect(urlBase64ToUint8Array('-_8')).toEqual(new Uint8Array([251, 255]));
  });
});

describe('logging out and lost sessions', () => {
  it('drops the browser subscription even when the backend cannot be told', async () => {
    deletePushSubscription.mockRejectedValue(new Error('offline'));
    const subscription = { endpoint: 'https://push.test/old', unsubscribe: vi.fn(() => Promise.resolve(true)) };
    await disablePushOnLogout(fakeWindow({ subscription }).win);
    expect(subscription.unsubscribe).toHaveBeenCalledTimes(1);
  });

  it('forgets the browser subscription without calling the backend', async () => {
    const subscription = { endpoint: 'https://push.test/old', unsubscribe: vi.fn(() => Promise.resolve(true)) };
    await forgetPushOnThisBrowser(fakeWindow({ subscription }).win);
    expect(subscription.unsubscribe).toHaveBeenCalledTimes(1);
    expect(deletePushSubscription).not.toHaveBeenCalled();
  });
});

describe('stopPushIfAppUninstalled', () => {
  const subscribed = () => ({ endpoint: 'https://push.test/old', unsubscribe: vi.fn(() => Promise.resolve(true)) });

  it('learns, in the installed app, that the check works on this device', async () => {
    const storage = {};
    const subscription = subscribed();
    await expect(stopPushIfAppUninstalled(fakeWindow({ standalone: true, relatedApps: [{ platform: 'webapp' }], storage, subscription }).win)).resolves.toBe(false);
    expect(storage['calciotto-installed-app-detection']).toBe('works');
    expect(subscription.unsubscribe).not.toHaveBeenCalled();
  });

  it('turns notifications off in a tab once the app is gone, when the check is proven', async () => {
    deletePushSubscription.mockResolvedValue({});
    const subscription = subscribed();
    const storage = { 'calciotto-installed-app-detection': 'works' };
    await expect(stopPushIfAppUninstalled(fakeWindow({ relatedApps: [], storage, subscription }).win)).resolves.toBe(true);
    expect(deletePushSubscription).toHaveBeenCalledWith('https://push.test/old');
    expect(subscription.unsubscribe).toHaveBeenCalledTimes(1);
  });

  it('leaves them alone while the app is still installed', async () => {
    const subscription = subscribed();
    const storage = { 'calciotto-installed-app-detection': 'works' };
    await expect(stopPushIfAppUninstalled(fakeWindow({ relatedApps: [{ platform: 'webapp' }], storage, subscription }).win)).resolves.toBe(false);
    expect(subscription.unsubscribe).not.toHaveBeenCalled();
  });

  it('never acts on a guess: no proof on this device, or no API at all', async () => {
    const subscription = subscribed();
    await expect(stopPushIfAppUninstalled(fakeWindow({ relatedApps: [], subscription }).win)).resolves.toBe(false);
    await expect(stopPushIfAppUninstalled(fakeWindow({ subscription, storage: { 'calciotto-installed-app-detection': 'works' } }).win)).resolves.toBe(false);
    expect(subscription.unsubscribe).not.toHaveBeenCalled();
  });
});
