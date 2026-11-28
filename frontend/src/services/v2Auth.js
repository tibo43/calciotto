import { t } from '@/i18n';
import { hasApiErrorCode } from './apiError';
import { actionErrorMessage } from './v2Match';

// The backend's auth messages are lower-case fragments ("no group matches
// this invite code"). The v2 pages show them as sentences: a capital first
// letter and a closing full stop.
export function asSentence(text) {
  const trimmed = typeof text === 'string' ? text.trim() : '';
  if (!trimmed) return '';
  const sentence = trimmed.charAt(0).toUpperCase() + trimmed.slice(1);
  return /[.!?…]$/.test(sentence) ? sentence : `${sentence}.`;
}

// The toast text after a failed sign up, reset request or reset: the
// backend's own reason when it gives one, the page's fallback otherwise.
export function authErrorMessage(error, fallback) {
  return asSentence(actionErrorMessage(error, fallback));
}

// What ResetPasswordV2 says for a link without a token, and for the
// backend's one answer to an unknown, expired or used token
// (ErrInvalidResetToken).
export const expiredResetLink = () => t('auth.reset.expired');
const INVALID_RESET_TOKEN = 'invalid or expired reset token';

export function resetErrorMessage(error) {
  const reason = error?.response?.data?.error;
  if (hasApiErrorCode(error, 'invalid_reset_token')
    || (typeof reason === 'string' && reason.trim().toLowerCase() === INVALID_RESET_TOKEN)) {
    return expiredResetLink();
  }
  return authErrorMessage(error, t('auth.reset.failed'));
}
