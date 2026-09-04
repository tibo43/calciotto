// A self-hosted "tinylink" for sharing a match, e.g. on WhatsApp — a UUID
// (`92bbe7a1-fa67-49dc-9058-85837504d0d4`) reads badly in a chat next to
// "Join Calciotto". Rather than reaching for a public shortener (most block
// browser JS from reading their response at all — no CORS header — and it
// would mean an external dependency, and its uptime, for something this
// simple), the match's own id is losslessly re-encoded as base62: the same
// 128 bits, just packed into ~22 characters instead of 36 with no hyphens.
// No lookup table, no backend route, no storage — decodeMatchId is the exact
// inverse of encodeMatchId, so the "short link" is the id itself, just
// spelled differently. See router/index.js's `/m/:code` route, which decodes
// and redirects to the real /matches/:id/edit.
const BASE62_ALPHABET = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';

function bigIntToBase62(value) {
  if (value === 0n) return '0';
  let digits = '';
  let n = value;
  while (n > 0n) {
    digits = BASE62_ALPHABET[Number(n % 62n)] + digits;
    n /= 62n;
  }
  return digits;
}

function base62ToBigInt(code) {
  let value = 0n;
  for (const char of code) {
    const digit = BASE62_ALPHABET.indexOf(char);
    if (digit === -1) {
      throw new Error(`invalid character in short link code: ${char}`);
    }
    value = value * 62n + BigInt(digit);
  }
  return value;
}

// The width every code is padded to: the length of the largest possible
// 128-bit value (2^128 - 1) written in base62, computed rather than
// hardcoded so it stays correct if BASE62_ALPHABET ever changed size. A
// *shorter* value — most real match ids, which almost never have enough
// leading zero bits to need all 22 characters — still gets left-padded with
// '0' (itself a valid base62 digit, distinct from an *absent* character) up
// to this width. Fixing the width is what makes a truncated-in-transit code
// detectable at all: a base62 string with a few trailing characters cut off
// is still a perfectly valid, perfectly canonical encoding of some *smaller*
// value — nothing about its content marks it as truncated, only its length
// does.
const CODE_LENGTH = bigIntToBase62((1n << 128n) - 1n).length;

export function encodeMatchId(uuid) {
  const hex = uuid.replace(/-/g, '');
  return bigIntToBase62(BigInt(`0x${hex}`)).padStart(CODE_LENGTH, '0');
}

// The inverse of encodeMatchId. Re-pads to the UUID's fixed 32 hex digits
// before re-inserting the hyphens: base62 strips leading zero bits (a UUID
// starting 0x00...) the same way parseInt would, so the *decoded value's*
// own hex length can't be trusted to reconstruct the original grouping —
// that's a separate concern from CODE_LENGTH above, which guards the input
// string.
//
// Two checks reject a corrupted code instead of silently decoding a
// different, wrong-but-plausible match id:
// - The code must be exactly CODE_LENGTH characters. A truncated code is
//   shorter; a code with stray extra characters appended is longer; both
//   are still made entirely of valid base62 digits, so only the fixed
//   width catches them.
// - The decoded value must still fit in 128 bits after that — a
//   CODE_LENGTH-character code can represent a value past 2^128-1 (62^22
//   exceeds 2^128), which the fixed-width hex slicing below would
//   otherwise silently truncate from the front instead of rejecting.
export function decodeMatchId(code) {
  if (code.length !== CODE_LENGTH) {
    throw new Error(`short link code has the wrong length: ${code}`);
  }
  const hex = base62ToBigInt(code).toString(16).padStart(32, '0');
  if (hex.length > 32) {
    throw new Error(`short link code out of range: ${code}`);
  }
  return [
    hex.slice(0, 8),
    hex.slice(8, 12),
    hex.slice(12, 16),
    hex.slice(16, 20),
    hex.slice(20, 32)
  ].join('-');
}
