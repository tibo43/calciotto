import {
  DESIGN_V2_FEATURE,
  DESIGN_V2_FONT_CSS,
  loadDesignV2,
  hasDesignV2,
  ensureDesignV2Fonts,
  removeDesignV2Fonts
} from '@/services/designV2';
import { loadFeatureStatus } from '@/services/featureStatus';

vi.mock('@/services/featureStatus', () => ({
  loadFeatureStatus: vi.fn()
}));

it('loads the design flag from the session feature status', async () => {
  loadFeatureStatus.mockResolvedValue({ features: ['pwa-install', 'design-v2'] });
  await expect(loadDesignV2()).resolves.toBe(true);
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
