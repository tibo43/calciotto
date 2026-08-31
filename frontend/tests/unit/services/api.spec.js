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
