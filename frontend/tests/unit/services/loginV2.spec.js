import { shouldUseLoginV2 } from '@/services/loginV2';

describe('loginV2', () => {
  it('gives every browser the v2 login', () => {
    localStorage.clear();
    expect(shouldUseLoginV2()).toBe(true);
  });
});
