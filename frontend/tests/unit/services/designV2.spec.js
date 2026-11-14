import {
  DESIGN_V2_FEATURE,
  DESIGN_V2_FONT_CSS,
  readDesignV2Cached,
  writeDesignV2Cached,
  clearDesignV2Cached,
  hasDesignV2,
  ensureDesignV2Fonts,
  removeDesignV2Fonts
} from '@/services/designV2';

const STORAGE_KEY = 'calciotto-design-v2';

describe('designV2 cache', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('reads false when nothing is stored', () => {
    expect(readDesignV2Cached()).toBe(false);
  });

  it('round-trips an enabled flag through localStorage', () => {
    writeDesignV2Cached(true);
    expect(localStorage.getItem(STORAGE_KEY)).toBe('1');
    expect(readDesignV2Cached()).toBe(true);
  });

  it('clears the flag when the feature is off, rather than storing a false value', () => {
    writeDesignV2Cached(true);
    writeDesignV2Cached(false);
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull();
    expect(readDesignV2Cached()).toBe(false);
  });

  it('clearDesignV2Cached removes a stored flag', () => {
    writeDesignV2Cached(true);
    clearDesignV2Cached();
    expect(readDesignV2Cached()).toBe(false);
  });

  it('treats any value other than the on-flag as off', () => {
    localStorage.setItem(STORAGE_KEY, 'true');
    expect(readDesignV2Cached()).toBe(false);
  });

  it('returns false when reading storage throws', () => {
    jest.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('SecurityError');
    });
    expect(readDesignV2Cached()).toBe(false);
    jest.restoreAllMocks();
  });

  it('swallows a storage failure when writing', () => {
    jest.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('QuotaExceededError');
    });
    expect(() => writeDesignV2Cached(true)).not.toThrow();
    jest.restoreAllMocks();
  });

  it('swallows a storage failure when clearing', () => {
    jest.spyOn(Storage.prototype, 'removeItem').mockImplementation(() => {
      throw new Error('SecurityError');
    });
    expect(() => clearDesignV2Cached()).not.toThrow();
    jest.restoreAllMocks();
  });
});

describe('hasDesignV2', () => {
  it('is true only when the list contains the design-v2 key', () => {
    expect(DESIGN_V2_FEATURE).toBe('design-v2');
    expect(hasDesignV2(['design-v2'])).toBe(true);
    expect(hasDesignV2(['pwa-install', 'design-v2'])).toBe(true);
    expect(hasDesignV2(['pwa-install'])).toBe(false);
    expect(hasDesignV2([])).toBe(false);
  });

  it('is false for a missing or non-list payload', () => {
    expect(hasDesignV2(undefined)).toBe(false);
    expect(hasDesignV2(null)).toBe(false);
    expect(hasDesignV2({ features: ['design-v2'] })).toBe(false);
  });
});

describe('design v2 font stylesheet', () => {
  afterEach(() => {
    removeDesignV2Fonts();
  });

  it('self-hosts the three chosen families and does not call Google Fonts', () => {
    expect(DESIGN_V2_FONT_CSS).toContain('Big Shoulders Display');
    expect(DESIGN_V2_FONT_CSS).toContain('Space Mono');
    expect(DESIGN_V2_FONT_CSS).toContain('DM Sans');
    expect(DESIGN_V2_FONT_CSS).toContain('/fonts/v2/');
    expect(DESIGN_V2_FONT_CSS).not.toContain('fonts.googleapis.com');
  });

  it('injects the font faces once', () => {
    ensureDesignV2Fonts();
    ensureDesignV2Fonts();
    const styles = document.querySelectorAll('#design-v2-fonts');
    expect(styles).toHaveLength(1);
    expect(styles[0].textContent).toBe(DESIGN_V2_FONT_CSS);
  });

  it('removes the stylesheet', () => {
    ensureDesignV2Fonts();
    removeDesignV2Fonts();
    expect(document.getElementById('design-v2-fonts')).toBeNull();
  });
});
