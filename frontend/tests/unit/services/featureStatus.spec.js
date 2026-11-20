import { getMyFeatureStatus, getSessionKey } from '@/services/api';
import { knownFeatureStatus, loadFeatureStatus, rememberGuideCompletion, resetFeatureStatus } from '@/services/featureStatus';

vi.mock('@/services/api', () => ({
  getMyFeatureStatus: vi.fn(),
  getSessionKey: vi.fn()
}));

describe('session feature status', () => {
  beforeEach(() => {
    resetFeatureStatus();
    localStorage.clear();
    getMyFeatureStatus.mockReset();
    getSessionKey.mockReset();
    getSessionKey.mockReturnValue('session-a');
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns empty features without asking when there is no session', async () => {
    getSessionKey.mockReturnValue(null);
    await expect(loadFeatureStatus()).resolves.toEqual({ features: [] });
    expect(getMyFeatureStatus).not.toHaveBeenCalled();
    expect(knownFeatureStatus()).toBeNull();
  });

  it('shares one request between concurrent callers and reuses it for the session', async () => {
    const status = { features: ['pwa-install'], guideCompleted: false };
    getMyFeatureStatus.mockResolvedValue(status);
    const [first, second] = await Promise.all([loadFeatureStatus(), loadFeatureStatus()]);
    await loadFeatureStatus();
    expect(first).toEqual(status);
    expect(second).toEqual(status);
    expect(getMyFeatureStatus).toHaveBeenCalledTimes(1);
    expect(knownFeatureStatus()).toEqual(status);
  });

  it('keeps guide completion scoped to the session that loaded it', async () => {
    getMyFeatureStatus
      .mockResolvedValueOnce({ features: ['pwa-install'], guideCompleted: false })
      .mockResolvedValueOnce({ features: [], guideCompleted: false });
    await loadFeatureStatus();
    rememberGuideCompletion();
    await expect(loadFeatureStatus()).resolves.toEqual({ features: ['pwa-install'], guideCompleted: true });

    getSessionKey.mockReturnValue('session-b');
    expect(knownFeatureStatus()).toBeNull();
    await expect(loadFeatureStatus()).resolves.toEqual({ features: [], guideCompleted: false });
    expect(getMyFeatureStatus).toHaveBeenCalledTimes(2);
  });

  it('falls back to empty features on a failed request and retries next time', async () => {
    const status = { features: ['pwa-install'], guideCompleted: false };
    getMyFeatureStatus.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce(status);
    await expect(loadFeatureStatus()).resolves.toEqual({ features: [] });
    expect(knownFeatureStatus()).toBeNull();
    expect(console.error).toHaveBeenCalled();
    await expect(loadFeatureStatus()).resolves.toEqual(status);
    expect(getMyFeatureStatus).toHaveBeenCalledTimes(2);
  });

  it('forgets the answer on reset', async () => {
    getMyFeatureStatus.mockResolvedValue({ features: ['pwa-install'] });
    await loadFeatureStatus();
    resetFeatureStatus();
    expect(knownFeatureStatus()).toBeNull();
  });

  it('keeps the status in memory', async () => {
    getMyFeatureStatus.mockResolvedValue({ features: ['pwa-install'], guideCompleted: false });
    await loadFeatureStatus();
    rememberGuideCompletion();
    expect(localStorage.length).toBe(0);
  });
});
