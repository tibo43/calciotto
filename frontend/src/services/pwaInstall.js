// What Profile.vue's "Install the app" card should offer, and the one action it
// can take. Kept out of the component for the same reason as
// matchRegistration.js: the platform rules are the part worth testing, and
// they don't need a mounted component.
//
// The two platforms install differently, which is why there are two offering
// states rather than one:
// - Chrome (Android, desktop) fires `beforeinstallprompt` once the page is
//   installable. Holding on to that event is what lets our own button open
//   Chrome's install dialog later.
// - iOS has no such event and no way to trigger an install from the page: the
//   user has to go through Safari's Share menu themselves, so all the card can
//   do there is say how.

export const INSTALL_STATE = {
  INSTALLED: 'installed',
  PROMPT: 'prompt',
  IOS: 'ios',
  UNAVAILABLE: 'unavailable'
};

// True when the page is running as the installed app rather than in a browser
// tab. `navigator.standalone` is iOS's older, non-standard flag for the same
// thing, kept as a fallback for versions where the media query isn't reliable.
export function isStandalone(win) {
  const media = typeof win.matchMedia === 'function' && win.matchMedia('(display-mode: standalone)').matches;
  return media || win.navigator.standalone === true;
}

// iPhone, iPod, and iPad — including iPadOS 13+, which reports itself as a
// Mac: a Mac with a touch screen is an iPad. Any browser on iOS counts, since
// from iOS 16.4 Chrome and Firefox there can add to the home screen too.
export function isIos(nav) {
  const ua = nav.userAgent || '';
  return /iPad|iPhone|iPod/.test(ua) || (/Macintosh/.test(ua) && nav.maxTouchPoints > 1);
}

export function deriveInstallState({ standalone, installed, canPrompt, ios }) {
  if (standalone || installed) {
    return INSTALL_STATE.INSTALLED;
  }
  if (canPrompt) {
    return INSTALL_STATE.PROMPT;
  }
  if (ios) {
    return INSTALL_STATE.IOS;
  }
  return INSTALL_STATE.UNAVAILABLE;
}

export function createInstallTracker(win) {
  let deferredPrompt = null;
  let installed = false;
  const listeners = new Set();

  const state = () => deriveInstallState({
    standalone: isStandalone(win),
    installed,
    canPrompt: deferredPrompt !== null,
    ios: isIos(win.navigator)
  });
  const notify = () => listeners.forEach((listener) => listener(state()));

  win.addEventListener('beforeinstallprompt', (event) => {
    // Stops Chrome from showing its own install banner on top of ours; the
    // install entry in Chrome's menu stays available regardless.
    event.preventDefault();
    deferredPrompt = event;
    notify();
  });
  win.addEventListener('appinstalled', () => {
    deferredPrompt = null;
    installed = true;
    notify();
  });

  return {
    state,
    subscribe(listener) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    // Opens Chrome's install dialog. The saved event can only be used once,
    // accepted or not, so the card hides afterwards until Chrome offers again.
    async prompt() {
      if (!deferredPrompt) {
        return 'unavailable';
      }
      const event = deferredPrompt;
      deferredPrompt = null;
      await event.prompt();
      const { outcome } = await event.userChoice;
      notify();
      return outcome;
    }
  };
}

// One tracker for the whole app, created from main.js before the app mounts:
// Chrome fires `beforeinstallprompt` shortly after load, typically long before
// anyone opens the Profile page, and an event nobody was listening for is lost.
let tracker = null;

export function getInstallTracker(win = window) {
  if (!tracker) {
    tracker = createInstallTracker(win);
  }
  return tracker;
}
