// The text of a failed API call, in the active language.
//
// The backend answers an error with its English text (`error`) and, for the
// errors a person can see, a stable `code` (and `params` for a text with a
// number in it; app/internal/handlers/errorcodes.go). The message for a code is
// `apiError.<code>` in the bundles, written in the language shown. A response
// with no code, or with a code this build does not know, keeps its English
// text, and a call that got no answer at all gives the caller's fallback.
import i18n, { t } from '@/i18n';

const KEY = (code) => `apiError.${code}`;

export function apiErrorMessage(error, fallback) {
  const data = error?.response?.data;
  const code = data?.code;
  // Looked up in English, the language every other one falls back to, so a
  // code a language has not translated yet still reads (in English).
  if (typeof code === 'string' && i18n.global.te(KEY(code), 'en')) {
    const params = data.params && typeof data.params === 'object' ? data.params : {};
    return t(KEY(code), params);
  }
  return data?.error || fallback;
}

// Whether the response carries this code, whatever the language.
export function hasApiErrorCode(error, code) {
  return error?.response?.data?.code === code;
}
