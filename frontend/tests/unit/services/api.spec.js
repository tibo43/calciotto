// api.js builds its Axios instance at import time, so the mock has to hand
// back one shared instance whose methods the assertions can inspect.
const mockInstance = {
  post: jest.fn(),
  get: jest.fn(),
  put: jest.fn(),
  patch: jest.fn(),
  delete: jest.fn(),
  interceptors: {
    request: { use: jest.fn() },
    response: { use: jest.fn() }
  }
};

jest.mock('axios', () => ({
  create: () => mockInstance
}));

const api = require('@/services/api');

describe('createMatch', () => {
  beforeEach(() => {
    mockInstance.post.mockReset();
    mockInstance.post.mockResolvedValue({ status: 200, data: 'match-uuid' });
  });

  it('sends only the match plus group_id when no scheduling is given', async () => {
    await api.createMatch({ Date: '2026-03-15' }, 'group-uuid');

    // Byte-identical to the pre-scheduling payload: no scheduling key is
    // added, not even as null/undefined, so an ordinary match reaches the
    // backend exactly as it always has.
    expect(mockInstance.post).toHaveBeenCalledWith('/matches', {
      Date: '2026-03-15',
      group_id: 'group-uuid'
    });
  });

  it('leaves the payload untouched when neither group nor scheduling is given', async () => {
    await api.createMatch({ Date: '2026-03-15' });

    expect(mockInstance.post).toHaveBeenCalledWith('/matches', { Date: '2026-03-15' });
  });

  it('sends the three scheduling fields under their snake_case keys', async () => {
    await api.createMatch({ Date: '2026-03-15' }, 'group-uuid', {
      scheduledAt: '2026-09-06T20:30:00+02:00',
      registrationOpensAt: '2026-09-01T12:00:00+02:00',
      maxPlayers: 16
    });

    expect(mockInstance.post).toHaveBeenCalledWith('/matches', {
      Date: '2026-03-15',
      group_id: 'group-uuid',
      scheduled_at: '2026-09-06T20:30:00+02:00',
      registration_opens_at: '2026-09-01T12:00:00+02:00',
      max_players: 16
    });
  });

  it('returns the bare uuid the backend answers with', async () => {
    await expect(api.createMatch({ Date: '2026-03-15' }, 'group-uuid')).resolves.toBe('match-uuid');
  });
});

describe('the match API surface', () => {
  // PATCH /matches/:id does not exist on the backend, and updateMatchPartial
  // bypassed the group scoping every other match call goes through — it must
  // stay gone rather than quietly come back.
  it('exposes no updateMatchPartial helper', () => {
    expect(api.updateMatchPartial).toBeUndefined();
  });
});

// None of the five sign-up calls takes a group_id: the backend resolves the
// group from the match named in the path, which is what stops a caller naming a
// match in one group while presenting a group they happen to belong to. A
// query string creeping in here would be that hole reopening.
describe('the match sign-up calls', () => {
  beforeEach(() => {
    mockInstance.get.mockReset();
    mockInstance.post.mockReset();
    mockInstance.delete.mockReset();
    mockInstance.get.mockResolvedValue({ status: 200, data: [] });
    mockInstance.post.mockResolvedValue({ status: 200, data: {} });
    mockInstance.delete.mockResolvedValue({ status: 200, data: { unregistered: true } });
  });

  it('signs the caller up with no body and no group_id', async () => {
    mockInstance.post.mockResolvedValue({
      status: 200,
      data: { PlayerID: 'p1', Name: 'me', Position: 17, IsWaiting: true }
    });

    const entry = await api.registerForMatch('match-uuid');

    expect(mockInstance.post).toHaveBeenCalledWith('/matches/match-uuid/registrations');
    // The entry is the only place "you are #17, on the bench" exists.
    expect(entry).toEqual({ PlayerID: 'p1', Name: 'me', Position: 17, IsWaiting: true });
  });

  it('withdraws the caller with a bare DELETE', async () => {
    await expect(api.unregisterFromMatch('match-uuid')).resolves.toEqual({ unregistered: true });

    expect(mockInstance.delete).toHaveBeenCalledWith('/matches/match-uuid/registrations');
  });

  it('fetches the ordered list', async () => {
    await api.getMatchRegistrations('match-uuid');

    expect(mockInstance.get).toHaveBeenCalledWith('/matches/match-uuid/registrations');
  });

  it('closes and reopens sign-ups on their own sub-routes', async () => {
    await api.closeMatchRegistrations('match-uuid');
    expect(mockInstance.post).toHaveBeenCalledWith('/matches/match-uuid/registrations/close');

    await api.reopenMatchRegistrations('match-uuid');
    expect(mockInstance.post).toHaveBeenCalledWith('/matches/match-uuid/registrations/reopen');
  });

  it('rethrows so a 409 reaches the caller that has to explain it', async () => {
    const conflict = { response: { status: 409, data: { error: 'already registered' } } };
    mockInstance.post.mockRejectedValue(conflict);

    await expect(api.registerForMatch('match-uuid')).rejects.toBe(conflict);
  });
});
