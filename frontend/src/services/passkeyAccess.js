const LOGIN_HINT_KEY = 'calciotto-passkey-login';

export function hasPasskeyAccess(features) {
  return Array.isArray(features) && features.includes('passkeys');
}

// Only a display hint for the signed-out page. The API checks the flag
// against the account for each passkey sign-in.
export function shouldOfferPasskeyLogin() {
  return localStorage.getItem(LOGIN_HINT_KEY) === '1';
}

export function rememberPasskeyAccess(enabled) {
  if (enabled) {
    localStorage.setItem(LOGIN_HINT_KEY, '1');
  } else {
    localStorage.removeItem(LOGIN_HINT_KEY);
  }
}
