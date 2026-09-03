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

export function encodeMatchId(uuid) {
  const hex = uuid.replace(/-/g, '');
  return bigIntToBase62(BigInt(`0x${hex}`));
}

// The inverse of encodeMatchId. Re-pads to the UUID's fixed 32 hex digits
// before re-inserting the hyphens: base62 strips leading zero bits (a UUID
// starting 0x00...) the same way parseInt would, so the length of `code`
// alone can't be trusted to reconstruct the original grouping.
export function decodeMatchId(code) {
  const hex = base62ToBigInt(code).toString(16).padStart(32, '0');
  return [
    hex.slice(0, 8),
    hex.slice(8, 12),
    hex.slice(12, 16),
    hex.slice(16, 20),
    hex.slice(20, 32)
  ].join('-');
}
