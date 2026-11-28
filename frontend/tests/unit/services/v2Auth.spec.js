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
