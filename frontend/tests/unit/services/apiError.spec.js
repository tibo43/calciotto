import { afterEach, describe, expect, it } from 'vitest';
import { apiErrorMessage, hasApiErrorCode } from '@/services/apiError';
import { i18n } from '@/i18n';
import { messages } from '@/locales';

const answer = (data) => ({ response: { data } });

afterEach(() => {
  i18n.global.locale.value = 'en';
  i18n.global.setLocaleMessage('es', messages.es);
});

describe('apiErrorMessage', () => {
  it('writes a coded error in the active language', () => {
    const error = answer({ error: 'registrations for this match are closed', code: 'registrations_closed' });
    expect(apiErrorMessage(error, 'Fallback.')).toBe('Registrations for this match are closed.');
    i18n.global.locale.value = 'de';
    expect(apiErrorMessage(error, 'Fallback.')).toBe('Die Anmeldung für dieses Spiel ist geschlossen.');
    i18n.global.locale.value = 'fr';
    expect(apiErrorMessage(error, 'Fallback.')).toBe('Les inscriptions à ce match sont fermées.');
  });

  it('fills the parameters the answer carries', () => {
    const error = answer({ error: 'password must be at least 8 characters long', code: 'password_too_short', params: { min: 8 } });
    expect(apiErrorMessage(error, '')).toBe('Password must be at least 8 characters long.');
    i18n.global.locale.value = 'it';
    expect(apiErrorMessage(error, '')).toBe('La password deve contenere almeno 8 caratteri.');
  });

  it('keeps the English text of an error with no code', () => {
    i18n.global.locale.value = 'de';
    expect(apiErrorMessage(answer({ error: 'invalid position "X"' }), 'Fallback.')).toBe('invalid position "X"');
  });

  it('keeps the English text of a code this build does not know', () => {
    i18n.global.locale.value = 'de';
    expect(apiErrorMessage(answer({ error: 'something new', code: 'brand_new_code' }), 'Fallback.')).toBe('something new');
  });

  it('reads in English a code a language has not translated yet', () => {
    // Message lookup falls back to English, the language every bundle falls back to.
    i18n.global.locale.value = 'es';
    i18n.global.setLocaleMessage('es', {});
    expect(apiErrorMessage(answer({ error: 'x', code: 'voting_closed' }), '')).toBe('Voting for MVP is closed for this match.');
  });

  it("gives the caller's fallback when there is no answer", () => {
    expect(apiErrorMessage(new Error('Network Error'), 'Could not save.')).toBe('Could not save.');
    expect(apiErrorMessage(undefined, 'Could not save.')).toBe('Could not save.');
    expect(apiErrorMessage(answer({}), 'Could not save.')).toBe('Could not save.');
    expect(apiErrorMessage(answer(''), 'Could not save.')).toBe('Could not save.');
  });

  it('ignores params that are not an object', () => {
    expect(apiErrorMessage(answer({ code: 'match_not_found', params: 'x' }), '')).toBe('Match not found.');
  });
});

describe('hasApiErrorCode', () => {
  it('tells whether the answer carries that code', () => {
    expect(hasApiErrorCode(answer({ code: 'invalid_reset_token' }), 'invalid_reset_token')).toBe(true);
    expect(hasApiErrorCode(answer({ code: 'other' }), 'invalid_reset_token')).toBe(false);
    expect(hasApiErrorCode(undefined, 'invalid_reset_token')).toBe(false);
  });
});
