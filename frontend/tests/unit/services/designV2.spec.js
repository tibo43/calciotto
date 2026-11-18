import {
  DESIGN_V2_FEATURE,
  DESIGN_V2_FONT_CSS,
  loadDesignV2,
  knownDesignV2State,
  resetDesignV2,
  hasDesignV2,
  ensureDesignV2Fonts,
  removeDesignV2Fonts
} from '@/services/designV2';
import { getMyFeatureStatus, getSessionKey } from '@/services/api';

vi.mock('@/services/api', () => ({
  getMyFeatureStatus: vi.fn(),
  getSessionKey: vi.fn()
}));

describe('designV2 session flag', () => {
  beforeEach(() => {
    resetDesignV2();
    localStorage.clear();
    getMyFeatureStatus.mockReset();
    getSessionKey.mockReset();
    getSessionKey.mockReturnValue('token-a');
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('resolves false without asking when there is no token', async () => {
    getSessionKey.mockReturnValue(null);
    await expect(loadDesignV2()).resolves.toBe(false);
    expect(getMyFeatureStatus).not.toHaveBeenCalled();
    expect(knownDesignV2State()).toBeNull();
  });

  it('takes the flag from the features response', async () => {
    getMyFeatureStatus.mockResolvedValue({ features: ['pwa-install', 'design-v2'] });
    expect(knownDesignV2State()).toBeNull();
    await expect(loadDesignV2()).resolves.toBe(true);
    expect(knownDesignV2State()).toEqual({ enabled: true, guideRequired: false });
  });

  it('shares one request between concurrent callers and reuses it for the session', async () => {
    getMyFeatureStatus.mockResolvedValue({ features: [] });
    const [a, b] = await Promise.all([loadDesignV2(), loadDesignV2()]);
    await loadDesignV2();
    expect(a).toBe(false);
    expect(b).toBe(false);
    expect(getMyFeatureStatus).toHaveBeenCalledTimes(1);
    expect(knownDesignV2State()).toEqual({ enabled: false, guideRequired: false });
  });

  it('asks again for another token', async () => {
    getMyFeatureStatus.mockResolvedValueOnce({ features: ['design-v2'] }).mockResolvedValueOnce({ features: [] });
    await expect(loadDesignV2()).resolves.toBe(true);
    getSessionKey.mockReturnValue('token-b');
    expect(knownDesignV2State()).toBeNull();
    await expect(loadDesignV2()).resolves.toBe(false);
    expect(getMyFeatureStatus).toHaveBeenCalledTimes(2);
  });

  it('falls back to false on a failed request and asks again next time', async () => {
    getMyFeatureStatus.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({ features: ['design-v2'] });
    await expect(loadDesignV2()).resolves.toBe(false);
    expect(knownDesignV2State()).toBeNull();
    expect(console.error).toHaveBeenCalled();
    await expect(loadDesignV2()).resolves.toBe(true);
    expect(getMyFeatureStatus).toHaveBeenCalledTimes(2);
  });

  it('forgets the answer on reset', async () => {
    getMyFeatureStatus.mockResolvedValue({ features: ['design-v2'] });
    await loadDesignV2();
    resetDesignV2();
    expect(knownDesignV2State()).toBeNull();
  });

  it('never writes anything to localStorage', async () => {
    getMyFeatureStatus.mockResolvedValue({ features: ['design-v2'] });
    await loadDesignV2();
    expect(localStorage.length).toBe(0);
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
