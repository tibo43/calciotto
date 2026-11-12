import {
  INSTALL_STATE, isStandalone, isIos, deriveInstallState, createInstallTracker
} from '@/services/pwaInstall';

const IPHONE_UA = 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1';
const ANDROID_UA = 'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Mobile Safari/537.36';
const MAC_UA = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15';

// A window just real enough for the tracker: event listeners, a user agent,
// and a display-mode media query.
function fakeWindow({ userAgent = ANDROID_UA, standalone = false, maxTouchPoints = 0 } = {}) {
  const target = new EventTarget();
  return {
    addEventListener: target.addEventListener.bind(target),
    dispatchEvent: target.dispatchEvent.bind(target),
    matchMedia: () => ({ matches: standalone }),
    navigator: { userAgent, maxTouchPoints }
  };
}

function fakeInstallPrompt(outcome) {
  const event = new Event('beforeinstallprompt', { cancelable: true });
  event.prompt = jest.fn(() => Promise.resolve());
  event.userChoice = Promise.resolve({ outcome });
  return event;
}

describe('isStandalone', () => {
  it('reads the display-mode media query', () => {
    expect(isStandalone(fakeWindow({ standalone: true }))).toBe(true);
    expect(isStandalone(fakeWindow())).toBe(false);
  });

  it("falls back on iOS's navigator.standalone", () => {
    const win = fakeWindow();
    win.navigator.standalone = true;
    expect(isStandalone(win)).toBe(true);
  });
});

describe('isIos', () => {
  it('recognises an iPhone and not Android', () => {
    expect(isIos({ userAgent: IPHONE_UA })).toBe(true);
    expect(isIos({ userAgent: ANDROID_UA })).toBe(false);
  });

  it('tells an iPad posing as a Mac from a real Mac by touch support', () => {
    expect(isIos({ userAgent: MAC_UA, maxTouchPoints: 5 })).toBe(true);
    expect(isIos({ userAgent: MAC_UA, maxTouchPoints: 0 })).toBe(false);
  });
});

describe('deriveInstallState', () => {
  const base = { standalone: false, installed: false, canPrompt: false, ios: false };

  it('offers nothing once installed, whatever else is true', () => {
    expect(deriveInstallState({ ...base, standalone: true, canPrompt: true, ios: true })).toBe(INSTALL_STATE.INSTALLED);
    expect(deriveInstallState({ ...base, installed: true, canPrompt: true })).toBe(INSTALL_STATE.INSTALLED);
  });

  it('prefers a real install prompt, then the iOS instructions', () => {
    expect(deriveInstallState({ ...base, canPrompt: true })).toBe(INSTALL_STATE.PROMPT);
    expect(deriveInstallState({ ...base, ios: true })).toBe(INSTALL_STATE.IOS);
    expect(deriveInstallState(base)).toBe(INSTALL_STATE.UNAVAILABLE);
  });
});

describe('createInstallTracker', () => {
  it("holds on to Chrome's offer, suppressing its own banner", () => {
    const win = fakeWindow();
    const tracker = createInstallTracker(win);
    const listener = jest.fn();
    tracker.subscribe(listener);
    expect(tracker.state()).toBe(INSTALL_STATE.UNAVAILABLE);

    const event = fakeInstallPrompt('accepted');
    win.dispatchEvent(event);
    expect(event.defaultPrevented).toBe(true);
    expect(tracker.state()).toBe(INSTALL_STATE.PROMPT);
    expect(listener).toHaveBeenCalledWith(INSTALL_STATE.PROMPT);
  });

  it('uses the offer once, then stops offering if dismissed', async () => {
    const win = fakeWindow();
    const tracker = createInstallTracker(win);
    const event = fakeInstallPrompt('dismissed');
    win.dispatchEvent(event);

    await expect(tracker.prompt()).resolves.toBe('dismissed');
    expect(event.prompt).toHaveBeenCalledTimes(1);
    expect(tracker.state()).toBe(INSTALL_STATE.UNAVAILABLE);
    await expect(tracker.prompt()).resolves.toBe('unavailable');
  });

  it('counts as installed after appinstalled, even in the browser tab', () => {
    const win = fakeWindow();
    const tracker = createInstallTracker(win);
    win.dispatchEvent(fakeInstallPrompt('accepted'));
    win.dispatchEvent(new Event('appinstalled'));
    expect(tracker.state()).toBe(INSTALL_STATE.INSTALLED);
  });

  it('offers instructions on iOS, where no event ever comes', () => {
    expect(createInstallTracker(fakeWindow({ userAgent: IPHONE_UA })).state()).toBe(INSTALL_STATE.IOS);
  });

  it('stops notifying after unsubscribe', () => {
    const win = fakeWindow();
    const tracker = createInstallTracker(win);
    const listener = jest.fn();
    const unsubscribe = tracker.subscribe(listener);
    unsubscribe();
    win.dispatchEvent(fakeInstallPrompt('accepted'));
    expect(listener).not.toHaveBeenCalled();
  });
});
