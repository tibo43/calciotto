import { getMyGroups } from '@/services/api';

jest.mock('@/services/api', () => ({
  getMyGroups: jest.fn()
}));

// Each test needs a fresh copy of the module: activeGroup.js caches
// `myGroupsPromise` at module scope, and that cache is exactly what's under
// test here, so it must not leak between tests.
const loadActiveGroupModule = () => {
  let mod;
  jest.isolateModules(() => {
    // eslint-disable-next-line global-require
    mod = require('@/services/activeGroup');
  });
  return mod;
};

describe('activeGroup', () => {
  const GROUP_A = { id: 'group-a', name: 'A' };
  const GROUP_B = { id: 'group-b', name: 'B' };

  beforeEach(() => {
    localStorage.clear();
    jest.clearAllMocks();
  });

  describe('localStorage wrappers', () => {
    it('getActiveGroupId returns empty string when nothing is stored', () => {
      const { getActiveGroupId } = loadActiveGroupModule();
      expect(getActiveGroupId()).toBe('');
    });

    it('setActiveGroupId/getActiveGroupId round-trip through localStorage', () => {
      const { setActiveGroupId, getActiveGroupId } = loadActiveGroupModule();
      setActiveGroupId('group-a');
      expect(getActiveGroupId()).toBe('group-a');
      expect(localStorage.getItem('calciotto-active-group')).toBe('group-a');
    });

    it('clearActiveGroupId removes the stored value', () => {
      const { setActiveGroupId, clearActiveGroupId, getActiveGroupId } = loadActiveGroupModule();
      setActiveGroupId('group-a');
      clearActiveGroupId();
      expect(getActiveGroupId()).toBe('');
    });
  });

  describe('resolveActiveGroup', () => {
    it('keeps a stored id that matches one of the returned groups, without rewriting it', async () => {
      getMyGroups.mockResolvedValue([GROUP_A, GROUP_B]);
      const { setActiveGroupId, resolveActiveGroup } = loadActiveGroupModule();
      setActiveGroupId('group-b');

      const result = await resolveActiveGroup();

      expect(result).toEqual({ groups: [GROUP_A, GROUP_B], activeGroupId: 'group-b' });
      expect(localStorage.getItem('calciotto-active-group')).toBe('group-b');
    });

    it('falls back to the first group and rewrites storage when the stored id matches nothing', async () => {
      getMyGroups.mockResolvedValue([GROUP_A, GROUP_B]);
      const { setActiveGroupId, resolveActiveGroup } = loadActiveGroupModule();
      setActiveGroupId('stale-id');

      const result = await resolveActiveGroup();

      expect(result).toEqual({ groups: [GROUP_A, GROUP_B], activeGroupId: 'group-a' });
      expect(localStorage.getItem('calciotto-active-group')).toBe('group-a');
    });

    it('clears the stored id and returns an empty active id when there are no groups at all', async () => {
      getMyGroups.mockResolvedValue([]);
      const { setActiveGroupId, resolveActiveGroup } = loadActiveGroupModule();
      setActiveGroupId('stale-id');

      const result = await resolveActiveGroup();

      expect(result).toEqual({ groups: [], activeGroupId: '' });
      expect(localStorage.getItem('calciotto-active-group')).toBeNull();
    });

    it('caches the in-flight/resolved promise: two calls without force only hit getMyGroups once', async () => {
      getMyGroups.mockResolvedValue([GROUP_A]);
      const { resolveActiveGroup } = loadActiveGroupModule();

      await resolveActiveGroup();
      await resolveActiveGroup();

      expect(getMyGroups).toHaveBeenCalledTimes(1);
    });

    it('clearMyGroupsCache() forces the next call to re-fetch', async () => {
      getMyGroups.mockResolvedValue([GROUP_A]);
      const { resolveActiveGroup, clearMyGroupsCache } = loadActiveGroupModule();

      await resolveActiveGroup();
      clearMyGroupsCache();
      await resolveActiveGroup();

      expect(getMyGroups).toHaveBeenCalledTimes(2);
    });

    it('{ force: true } forces a re-fetch without needing clearMyGroupsCache()', async () => {
      getMyGroups.mockResolvedValue([GROUP_A]);
      const { resolveActiveGroup } = loadActiveGroupModule();

      await resolveActiveGroup();
      await resolveActiveGroup({ force: true });

      expect(getMyGroups).toHaveBeenCalledTimes(2);
    });
  });

  describe('resolveActiveGroupId', () => {
    it('returns just the active group id on success', async () => {
      getMyGroups.mockResolvedValue([GROUP_A]);
      const { resolveActiveGroupId } = loadActiveGroupModule();

      await expect(resolveActiveGroupId()).resolves.toBe('group-a');
    });

    it('swallows a getMyGroups failure and resolves to an empty string', async () => {
      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
      getMyGroups.mockRejectedValue(new Error('network down'));
      const { resolveActiveGroupId } = loadActiveGroupModule();

      await expect(resolveActiveGroupId()).resolves.toBe('');

      consoleErrorSpy.mockRestore();
    });

    it('does not cache a failure: a later successful call still returns a group', async () => {
      getMyGroups.mockRejectedValueOnce(new Error('network down'));
      getMyGroups.mockResolvedValueOnce([GROUP_A]);
      jest.spyOn(console, 'error').mockImplementation(() => {});
      const { resolveActiveGroupId } = loadActiveGroupModule();

      await expect(resolveActiveGroupId()).resolves.toBe('');
      await expect(resolveActiveGroupId()).resolves.toBe('group-a');

      expect(getMyGroups).toHaveBeenCalledTimes(2);
    });
  });
});
