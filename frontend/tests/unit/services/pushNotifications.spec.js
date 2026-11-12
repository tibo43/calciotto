import {
  PUSH_STATE, getPushState, enablePush, disablePush, urlBase64ToUint8Array
} from '@/services/pushNotifications';
import { getPushPublicKey, savePushSubscription, deletePushSubscription } from '@/services/api';

jest.mock('@/services/api', () => ({
  getPushPublicKey: jest.fn(),
  savePushSubscription: jest.fn(),
  deletePushSubscription: jest.fn()
}));

const ANDROID_UA = 'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Mobile Safari/537.36';
const IPHONE_UA = 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1';

// A window with a registered worker whose push subscription the test controls.
function fakeWindow({ userAgent = ANDROID_UA, standalone = false, permission = 'default', subscription = null, registered = true, push = true } = {}) {
  const pushManager = {
    getSubscription: jest.fn(() => Promise.resolve(subscription)),
    subscribe: jest.fn(() => Promise.resolve({
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
    Notification: { permission, requestPermission: jest.fn(() => Promise.resolve(permission)) }
  };
  if (push) win.PushManager = function PushManager() {};
  return { win, pushManager };
}

beforeEach(() => jest.clearAllMocks());

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
    expect(await getPushState(fakeWindow({ subscription: { endpoint: 'x' } }).win)).toBe(PUSH_STATE.ON);
    expect(await getPushState(fakeWindow().win)).toBe(PUSH_STATE.OFF);
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
    const subscription = { endpoint: 'https://push.test/old', unsubscribe: jest.fn(() => { order.push('browser'); return Promise.resolve(true); }) };
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
