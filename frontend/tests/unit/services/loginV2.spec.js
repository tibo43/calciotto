import { hasDesignV2LoginMark, rememberDesignV2Login, shouldUseLoginV2 } from '@/services/loginV2';

describe('loginV2', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it('uses the v2 login only on a browser carrying the mark', () => {
    expect(shouldUseLoginV2()).toBe(false);
    rememberDesignV2Login(true);
    expect(localStorage.getItem('calciotto-login-v2')).toBe('1');
    expect(hasDesignV2LoginMark()).toBe(true);
    expect(shouldUseLoginV2()).toBe(true);
    rememberDesignV2Login(false);
    expect(localStorage.getItem('calciotto-login-v2')).toBeNull();
    expect(shouldUseLoginV2()).toBe(false);
  });

  it('ignores any other value', () => {
    localStorage.setItem('calciotto-login-v2', 'true');
    expect(shouldUseLoginV2()).toBe(false);
  });

  it('keeps the v1 login when storage is unavailable', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('denied'); });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('denied'); });
    expect(() => rememberDesignV2Login(true)).not.toThrow();
    expect(shouldUseLoginV2()).toBe(false);
  });
});
