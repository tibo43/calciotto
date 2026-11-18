import { browserSupportsWebAuthn, startAuthentication, startRegistration } from '@simplewebauthn/browser';
import { beginPasskeyLogin, finishPasskeyLogin, beginPasskeyRegistration, finishPasskeyRegistration } from './api';

export function supportsPasskeys() {
  return window.isSecureContext && browserSupportsWebAuthn();
}

export async function signInWithPasskey() {
  const { challenge_id, publicKey } = await beginPasskeyLogin();
  const credential = await startAuthentication({ optionsJSON: publicKey });
  return finishPasskeyLogin(challenge_id, credential);
}

export async function addPasskey(name, password) {
  const { challenge_id, publicKey } = await beginPasskeyRegistration(name, password);
  const credential = await startRegistration({ optionsJSON: publicKey });
  return finishPasskeyRegistration(challenge_id, credential);
}
