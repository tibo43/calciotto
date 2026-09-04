import { encodeMatchId, decodeMatchId } from '@/services/shortLink';

describe('encodeMatchId / decodeMatchId', () => {
  it('round-trips a real-looking UUID', () => {
    const uuid = '92bbe7a1-fa67-49dc-9058-85837504d0d4';
    expect(decodeMatchId(encodeMatchId(uuid))).toBe(uuid);
  });

  it('round-trips the all-zero UUID, where every leading bit is zero', () => {
    const uuid = '00000000-0000-0000-0000-000000000000';
    expect(decodeMatchId(encodeMatchId(uuid))).toBe(uuid);
  });

  it('round-trips the all-F UUID, the largest possible value', () => {
    const uuid = 'ffffffff-ffff-ffff-ffff-ffffffffffff';
    expect(decodeMatchId(encodeMatchId(uuid))).toBe(uuid);
  });

  it('is noticeably shorter than the UUID it encodes', () => {
    const uuid = '92bbe7a1-fa67-49dc-9058-85837504d0d4';
    expect(encodeMatchId(uuid).length).toBeLessThan(uuid.length);
  });

  // Every real code is left-padded to the same fixed width (see
  // CODE_LENGTH in shortLink.js), even one encoding a UUID with several
  // leading zero bits — this is exactly what makes a truncated code
  // detectable at all, below.
  it('pads every code to the same fixed width regardless of the UUID', () => {
    const short = encodeMatchId('00000000-0000-0000-0000-000000000000');
    const long = encodeMatchId('ffffffff-ffff-ffff-ffff-ffffffffffff');

    expect(short.length).toBe(long.length);
  });

  it('rejects a code with a character outside the base62 alphabet', () => {
    const code = encodeMatchId('92bbe7a1-fa67-49dc-9058-85837504d0d4');
    const withBadChar = '!' + code.slice(1);

    expect(withBadChar.length).toBe(code.length);
    expect(() => decodeMatchId(withBadChar)).toThrow();
  });

  // A truncated code (e.g. cut off mid-copy-paste) is still made up
  // entirely of valid base62 characters representing a perfectly valid,
  // perfectly canonical *smaller* number — nothing about its content marks
  // it as truncated, so only the fixed-width check below can catch it.
  it('rejects a truncated code instead of silently decoding a different match id', () => {
    const code = encodeMatchId('92bbe7a1-fa67-49dc-9058-85837504d0d4');
    const truncated = code.slice(0, -3);

    expect(() => decodeMatchId(truncated)).toThrow();
  });

  it('rejects a code with stray extra characters appended', () => {
    const code = encodeMatchId('92bbe7a1-fa67-49dc-9058-85837504d0d4');
    const tooLong = code + '00';

    expect(() => decodeMatchId(tooLong)).toThrow();
  });

  // Same fixed width as every other code here, but representing a value
  // past 2^128-1 (62^22 exceeds 2^128) — the length check alone wouldn't
  // catch this, only the separate 128-bit range check does.
  it('rejects a code whose decoded value no longer fits in 128 bits', () => {
    const codeLength = encodeMatchId('ffffffff-ffff-ffff-ffff-ffffffffffff').length;
    const tooLarge = 'z'.repeat(codeLength);

    expect(() => decodeMatchId(tooLarge)).toThrow();
  });
});
