// No existing precedent in this codebase for mounting/exercising the router
// itself in a test, so this is a plain Jest unit test of the guard function
// canEditMatch exports in isolation, the same "pure logic gets tested
// directly" rule the rest of tests/unit/ already follows for
// matchRegistration.js/motmVoting.js.

import { canEditMatch, findGroupForMatch } from '@/router/index';
import { getMatchDetailsByID } from '@/services/api';
import { loadMyGroups } from '@/services/activeGroup';

jest.mock('@/services/api', () => ({
  getMatchDetailsByID: jest.fn()
}));

jest.mock('@/services/activeGroup', () => ({
  loadMyGroups: jest.fn()
}));

const notFound = () => Promise.reject({ response: { status: 404 } });
const serverError = () => Promise.reject({ response: { status: 500 } });

beforeEach(() => {
  getMatchDetailsByID.mockReset();
  loadMyGroups.mockReset();
});

describe('canEditMatch', () => {
  it('allows an admin of the match\'s only group', async () => {
    loadMyGroups.mockResolvedValue([{ id: 'group-a', role: 'admin' }]);
    getMatchDetailsByID.mockResolvedValue({ ID: 'match-1', GroupID: 'group-a' });

    await expect(canEditMatch('match-1')).resolves.toBe(true);
    expect(getMatchDetailsByID).toHaveBeenCalledWith('match-1', 'group-a');
  });

  it('refuses a plain member of the match\'s only group', async () => {
    loadMyGroups.mockResolvedValue([{ id: 'group-a', role: 'member' }]);
    getMatchDetailsByID.mockResolvedValue({ ID: 'match-1', GroupID: 'group-a' });

    await expect(canEditMatch('match-1')).resolves.toBe(false);
  });

  it('tries every group in turn until the match is found, then checks that group\'s role', async () => {
    loadMyGroups.mockResolvedValue([
      { id: 'group-a', role: 'admin' },
      { id: 'group-b', role: 'admin' }
    ]);
    // The match isn't in group-a (404); it's in group-b.
    getMatchDetailsByID
      .mockImplementationOnce(() => notFound())
      .mockImplementationOnce(() => Promise.resolve({ ID: 'match-1', GroupID: 'group-b' }));

    await expect(canEditMatch('match-1')).resolves.toBe(true);
    expect(getMatchDetailsByID).toHaveBeenNthCalledWith(1, 'match-1', 'group-a');
    expect(getMatchDetailsByID).toHaveBeenNthCalledWith(2, 'match-1', 'group-b');
  });

  it('uses the role of the group the match actually belongs to, not the first group tried', async () => {
    loadMyGroups.mockResolvedValue([
      { id: 'group-a', role: 'admin' },
      { id: 'group-b', role: 'member' }
    ]);
    getMatchDetailsByID
      .mockImplementationOnce(() => notFound())
      .mockImplementationOnce(() => Promise.resolve({ ID: 'match-1', GroupID: 'group-b' }));

    // The match is in group-b, where the caller is only a member.
    await expect(canEditMatch('match-1')).resolves.toBe(false);
  });

  it('refuses when the match belongs to no group the caller is a member of', async () => {
    loadMyGroups.mockResolvedValue([
      { id: 'group-a', role: 'admin' },
      { id: 'group-b', role: 'admin' }
    ]);
    getMatchDetailsByID.mockImplementation(() => notFound());

    await expect(canEditMatch('match-1')).resolves.toBe(false);
    expect(getMatchDetailsByID).toHaveBeenCalledTimes(2);
  });

  it('refuses when the caller belongs to no group at all', async () => {
    loadMyGroups.mockResolvedValue([]);

    await expect(canEditMatch('match-1')).resolves.toBe(false);
    expect(getMatchDetailsByID).not.toHaveBeenCalled();
  });

  it('propagates a non-404 failure rather than treating it as "not this group"', async () => {
    loadMyGroups.mockResolvedValue([{ id: 'group-a', role: 'admin' }]);
    getMatchDetailsByID.mockImplementation(() => serverError());

    await expect(canEditMatch('match-1')).rejects.toEqual({ response: { status: 500 } });
  });
});

// The shared "which of my groups owns this match" search canEditMatch above
// is built on — also reused by MatchesAndStandings.vue to resolve a
// `?match=<code>` deep link (a shared /m/:code tinylink) into the group to
// switch into and the match's own details, without a second copy of the
// same try-each-group loop.
describe('findGroupForMatch', () => {
  it('returns the group and match details once found', async () => {
    getMatchDetailsByID.mockResolvedValue({ ID: 'match-1', Date: '2026-09-06' });

    const result = await findGroupForMatch('match-1', [{ id: 'group-a', role: 'member' }]);

    expect(result).toEqual({
      group: { id: 'group-a', role: 'member' },
      details: { ID: 'match-1', Date: '2026-09-06' }
    });
  });

  it('tries every group in turn until the match is found', async () => {
    getMatchDetailsByID
      .mockImplementationOnce(() => notFound())
      .mockImplementationOnce(() => Promise.resolve({ ID: 'match-1', Date: '2026-09-06' }));

    const result = await findGroupForMatch('match-1', [
      { id: 'group-a', role: 'admin' },
      { id: 'group-b', role: 'member' }
    ]);

    expect(result.group.id).toBe('group-b');
    expect(getMatchDetailsByID).toHaveBeenNthCalledWith(1, 'match-1', 'group-a');
    expect(getMatchDetailsByID).toHaveBeenNthCalledWith(2, 'match-1', 'group-b');
  });

  it('returns null when the match belongs to none of the given groups', async () => {
    getMatchDetailsByID.mockImplementation(() => notFound());

    const result = await findGroupForMatch('match-1', [{ id: 'group-a', role: 'admin' }]);

    expect(result).toBeNull();
  });

  it('returns null when given no groups at all', async () => {
    const result = await findGroupForMatch('match-1', []);

    expect(result).toBeNull();
    expect(getMatchDetailsByID).not.toHaveBeenCalled();
  });

  it('propagates a non-404 failure rather than treating it as "not this group"', async () => {
    getMatchDetailsByID.mockImplementation(() => serverError());

    await expect(findGroupForMatch('match-1', [{ id: 'group-a', role: 'admin' }])).rejects.toEqual({
      response: { status: 500 }
    });
  });
});
