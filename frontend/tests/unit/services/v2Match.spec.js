import { findMatchInMyGroups, actionErrorMessage } from '@/services/v2Match';
import { resolveActiveGroup } from '@/services/activeGroup';
import { findGroupForMatch } from '@/router/index';

vi.mock('@/services/activeGroup', () => ({ resolveActiveGroup: vi.fn() }));
vi.mock('@/router/index', () => ({ findGroupForMatch: vi.fn() }));

const GROUPS = [
  { id: 'first', role: 'member' },
  { id: 'active', role: 'admin' },
  { id: 'third', role: 'member' }
];

describe('findMatchInMyGroups', () => {
  beforeEach(() => {
    resolveActiveGroup.mockResolvedValue({ groups: GROUPS, activeGroupId: 'active' });
    findGroupForMatch.mockReset();
  });

  it('searches every group, the active one first', async () => {
    findGroupForMatch.mockResolvedValue({ group: GROUPS[2], details: { ID: 'm' } });
    const found = await findMatchInMyGroups('m');
    expect(findGroupForMatch).toHaveBeenCalledWith('m', [GROUPS[1], GROUPS[0], GROUPS[2]]);
    expect(found).toEqual({ group: GROUPS[2], details: { ID: 'm' }, groups: GROUPS });
  });

  it('fails when the match is in none of them', async () => {
    findGroupForMatch.mockResolvedValue(null);
    await expect(findMatchInMyGroups('m')).rejects.toThrow('not in any of your groups');
  });
});

describe('actionErrorMessage', () => {
  it("prefers the backend's reason, else the fallback", () => {
    expect(actionErrorMessage({ response: { data: { error: 'registrations are closed' } } }, 'x')).toBe('registrations are closed');
    expect(actionErrorMessage(new Error('network'), 'Could not save.')).toBe('Could not save.');
  });
});
