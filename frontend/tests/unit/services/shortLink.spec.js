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

  it('rejects a code with a character outside the base62 alphabet', () => {
    expect(() => decodeMatchId('not-base62!')).toThrow();
  });
});
