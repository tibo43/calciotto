// Which login page a signed-out browser gets. The v2 page (LoginV2.vue)
// needs a design-v2 session to have happened on this browser before: there
// is no session on /login to ask GET /players/me/features with.
//
// The mark is only a display hint, like the passkey one (passkeyAccess.js).
// api.js writes it from each features response (set for a design-v2 player,
// removed for anyone else) and removes it when the account is deleted. It
// survives logout on purpose: the next screen after logout is this one.
const MARK_KEY = 'calciotto-login-v2';

export function hasDesignV2LoginMark() {
  try {
    return localStorage.getItem(MARK_KEY) === '1';
  } catch {
    return false;
  }
}

export function rememberDesignV2Login(enabled) {
  try {
    if (enabled) {
      localStorage.setItem(MARK_KEY, '1');
    } else {
      localStorage.removeItem(MARK_KEY);
    }
  } catch {
    // Private mode or storage full: the browser just keeps the v1 login.
  }
}

// THE switch, read by App.vue (shell and fonts) and, through it, by
// LoginGate.vue. Decided: only browsers carrying the mark get the v2 login.
// To show it to everyone instead, make this `return true;`.
export function shouldUseLoginV2() {
  return hasDesignV2LoginMark();
}
