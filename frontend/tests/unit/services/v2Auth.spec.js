import { afterEach } from 'vitest';
import { i18n } from '@/i18n';
import { asSentence, authErrorMessage, expiredResetLink, resetErrorMessage } from '@/services/v2Auth';

vi.mock('@/router/index', () => ({ findGroupForMatch: vi.fn() }));

const rejected = (error) => ({ response: { data: { error } } });

describe('v2Auth', () => {
  it.each([
    ['no group matches this invite code', 'No group matches this invite code.'],
    ['  invalid email or password ', 'Invalid email or password.'],
    ['Already a sentence.', 'Already a sentence.'],
    ['Really?', 'Really?'],
    ['', ''],
    [undefined, ''],
  ])('asSentence(%j) is %j', (input, expected) => {
    expect(asSentence(input)).toBe(expected);
  });

  it("uses the backend's reason, else the fallback, as a sentence", () => {
    expect(authErrorMessage(rejected('invite code is required'), 'Fallback.')).toBe('Invite code is required.');
    expect(authErrorMessage(new Error('Network Error'), 'Signup failed. Please try again.')).toBe('Signup failed. Please try again.');
  });

  it('turns the invalid reset token answer into the expired link message', () => {
    expect(resetErrorMessage(rejected('invalid or expired reset token'))).toBe(expiredResetLink());
    expect(expiredResetLink()).toBe('This link has expired. Request a new one.');
    expect(resetErrorMessage(rejected('password is required'))).toBe('Password is required.');
    expect(resetErrorMessage(undefined)).toBe('Could not reset the password. Please try again.');
  });
});

describe('v2Auth errors in the active language', () => {
  const coded = (error, code) => ({ response: { data: { error, code } } });
  afterEach(() => { i18n.global.locale.value = 'en'; });

  it('writes a coded refusal in the language, as a sentence', () => {
    i18n.global.locale.value = 'de';
    expect(authErrorMessage(coded('no group matches this invite code', 'invite_code_not_found'), 'x')).toBe('Zu diesem Einladungscode gibt es keine Gruppe.');
    expect(authErrorMessage(coded('invalid email or password', 'invalid_credentials'), 'x')).toBe('E-Mail-Adresse oder Passwort ist falsch.');
  });

  it('keeps the English sentence in English', () => {
    expect(authErrorMessage(coded('no group matches this invite code', 'invite_code_not_found'), 'x')).toBe('No group matches this invite code.');
  });

  it('recognises the expired reset link by its code, in any language', () => {
    i18n.global.locale.value = 'fr';
    expect(resetErrorMessage(coded('invalid or expired reset token', 'invalid_reset_token'))).toBe(expiredResetLink());
  });
});
