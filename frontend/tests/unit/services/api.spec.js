import * as api from '@/services/api';
import { shouldOfferPasskeyLogin } from '@/services/passkeyAccess';

// api.js installs the interceptors at import time, so keep one shared mock for
// each Axios instance and inspect the registered callbacks below.
const { mockInstance, mockSessionAPI, forgetPush } = vi.hoisted(() => ({
  mockInstance: Object.assign(vi.fn(), {
    post: vi.fn(),
    get: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
    interceptors: {
      request: { use: vi.fn() },
      response: { use: vi.fn() }
    }
  }),
  mockSessionAPI: { post: vi.fn() },
  forgetPush: vi.fn(() => Promise.resolve())
}));

vi.mock('axios', () => ({
  default: { create: ({ baseURL }) => baseURL === '/api' ? mockSessionAPI : mockInstance }
}));

vi.mock('@/services/pushNotifications', () => ({ forgetPushOnThisBrowser: forgetPush }));

const [prepareRequest] = mockInstance.interceptors.request.use.mock.calls[0];
const [handleResponse, handleResponseError] = mockInstance.interceptors.response.use.mock.calls[0];

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

  it('sends register_creator only when the scheduling payload asks for it', async () => {
    await api.createMatch({ Date: '2026-03-15' }, 'group-uuid', {
      scheduledAt: '2026-09-06T20:30:00+02:00',
      registrationOpensAt: '2026-09-01T12:00:00+02:00',
      maxPlayers: 16,
      registerCreator: true
    });

    expect(mockInstance.post).toHaveBeenCalledWith('/matches', {
      Date: '2026-03-15',
      group_id: 'group-uuid',
      scheduled_at: '2026-09-06T20:30:00+02:00',
      registration_opens_at: '2026-09-01T12:00:00+02:00',
      max_players: 16,
      register_creator: true
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
    mockInstance.patch.mockReset();
    mockInstance.patch.mockResolvedValue({ status: 200, data: [] });
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

  it('changes the confirmed cap with a PATCH and no group_id', async () => {
    const recomputed = [{ PlayerID: 'p1', Name: 'marco', Position: 1, IsWaiting: false }];
    mockInstance.patch.mockResolvedValue({ status: 200, data: recomputed });

    const entries = await api.setMatchMaxPlayers('match-uuid', 10);

    expect(mockInstance.patch).toHaveBeenCalledWith(
      '/matches/match-uuid/registrations/max-players',
      { max_players: 10 }
    );
    // It answers with the recomputed list, which is what the caller renders —
    // the confirmed/waiting split is never re-derived client-side.
    expect(entries).toEqual(recomputed);
  });

  it('rethrows so a 409 reaches the caller that has to explain it', async () => {
    const conflict = { response: { status: 409, data: { error: 'already registered' } } };
    mockInstance.post.mockRejectedValue(conflict);

    await expect(api.registerForMatch('match-uuid')).rejects.toBe(conflict);
  });
});

describe('deleteMyAccount', () => {
  beforeEach(() => {
    mockInstance.delete.mockReset();
    mockInstance.delete.mockResolvedValue({ status: 200, data: { deleted: true } });
  });

  it('sends the password as the DELETE request body', async () => {
    await expect(api.deleteMyAccount('s3cret-pass')).resolves.toEqual({ deleted: true });

    // axios only accepts a body on DELETE via the config's `data` key.
    expect(mockInstance.delete).toHaveBeenCalledWith('/players/me', { data: { password: 's3cret-pass' } });
  });

  it('rethrows so a wrong-password 400 reaches the caller', async () => {
    const rejection = { response: { status: 400, data: { error: 'invalid email or password' } } };
    mockInstance.delete.mockRejectedValue(rejection);

    await expect(api.deleteMyAccount('wrong-pass')).rejects.toBe(rejection);
  });
});

// Same reasoning as the sign-up calls above: the group is derived from the
// match named in the path, so none of these three carry a group_id either.
describe('the Man of the Match vote calls', () => {
  beforeEach(() => {
    mockInstance.get.mockReset();
    mockInstance.post.mockReset();
    mockInstance.delete.mockReset();
    mockInstance.get.mockResolvedValue({ status: 200, data: { Tally: [], MyVoteFor: null } });
    mockInstance.post.mockResolvedValue({ status: 200, data: { Tally: [], MyVoteFor: null } });
    mockInstance.delete.mockResolvedValue({ status: 200, data: { unvoted: true } });
  });

  it('casts a vote with the candidate id in the body and no group_id', async () => {
    mockInstance.post.mockResolvedValue({
      status: 200,
      data: { Tally: [{ PlayerID: 'p1', Name: 'alice', Votes: 1 }], MyVoteFor: 'p1' }
    });

    const summary = await api.voteForMotm('match-uuid', 'p1');

    expect(mockInstance.post).toHaveBeenCalledWith('/matches/match-uuid/votes', { voted_for_id: 'p1' });
    expect(summary).toEqual({ Tally: [{ PlayerID: 'p1', Name: 'alice', Votes: 1 }], MyVoteFor: 'p1' });
  });

  it('removes the caller\'s vote with a bare DELETE', async () => {
    await expect(api.removeMotmVote('match-uuid')).resolves.toEqual({ unvoted: true });

    expect(mockInstance.delete).toHaveBeenCalledWith('/matches/match-uuid/votes');
  });

  it('fetches the tally with a bare GET', async () => {
    await api.getMatchVotes('match-uuid');

    expect(mockInstance.get).toHaveBeenCalledWith('/matches/match-uuid/votes');
  });

  it('rethrows so a 400 (self-vote / not on roster) reaches the caller', async () => {
    const badRequest = { response: { status: 400, data: { error: 'not on roster' } } };
    mockInstance.post.mockRejectedValue(badRequest);

    await expect(api.voteForMotm('match-uuid', 'p1')).rejects.toBe(badRequest);
  });
});

describe('getMotmStandings', () => {
  beforeEach(() => {
    mockInstance.get.mockReset();
    mockInstance.get.mockResolvedValue({ status: 200, data: [] });
  });

  it('queries /standings/motm with season and group_id', async () => {
    await api.getMotmStandings('2025-2026', 'group-uuid');

    expect(mockInstance.get).toHaveBeenCalledWith('/standings/motm?season=2025-2026&group_id=group-uuid');
  });
});

// Same rule again: the group comes from the match in the path, so neither
// prediction call on a match carries a group_id.
describe('the prediction calls', () => {
  beforeEach(() => {
    mockInstance.get.mockReset();
    mockInstance.post.mockReset();
    mockInstance.get.mockResolvedValue({ status: 200, data: { Entries: [], Odds: null, MyPrediction: null } });
    mockInstance.post.mockResolvedValue({ status: 200, data: { Entries: [], Odds: null, MyPrediction: null } });
  });

  it('casts a team prediction with the team id in the body and no group_id', async () => {
    await api.predictForMatch('match-uuid', 'team-a');

    expect(mockInstance.post).toHaveBeenCalledWith('/matches/match-uuid/predictions', { predicted_team_id: 'team-a' });
  });

  // The key is always sent, null for a draw, so the payload says "draw"
  // outright rather than relying on an absent key binding to nil.
  it('casts a draw as an explicit null predicted_team_id', async () => {
    await api.predictForMatch('match-uuid', null);

    expect(mockInstance.post).toHaveBeenCalledWith('/matches/match-uuid/predictions', { predicted_team_id: null });
  });

  it('fetches the summary with a bare GET', async () => {
    await api.getMatchPredictions('match-uuid');

    expect(mockInstance.get).toHaveBeenCalledWith('/matches/match-uuid/predictions');
  });

  it('rethrows so a 409 (window closed) reaches the caller', async () => {
    const conflict = { response: { status: 409, data: { error: 'predictions are closed' } } };
    mockInstance.post.mockRejectedValue(conflict);

    await expect(api.predictForMatch('match-uuid', 'team-a')).rejects.toBe(conflict);
  });
});

describe('getPredictionStandings', () => {
  beforeEach(() => {
    mockInstance.get.mockReset();
    mockInstance.get.mockResolvedValue({ status: 200, data: [] });
  });

  it('queries /standings/predictions with season and group_id', async () => {
    await api.getPredictionStandings('2025-2026', 'group-uuid');

    expect(mockInstance.get).toHaveBeenCalledWith('/standings/predictions?season=2025-2026&group_id=group-uuid');
  });
});

describe('getMyFeatures', () => {
  beforeEach(() => mockInstance.get.mockReset());

  it('returns the features list from GET /players/me/features', async () => {
    mockInstance.get.mockResolvedValue({ data: { features: ['pwa-install'] } });
    await expect(api.getMyFeatures()).resolves.toEqual(['pwa-install']);
    expect(mockInstance.get).toHaveBeenCalledWith('/players/me/features');
  });

  it('falls back to an empty list on an unexpected body', async () => {
    mockInstance.get.mockResolvedValue({ data: {} });
    await expect(api.getMyFeatures()).resolves.toEqual([]);
  });
});

describe('the passkey login hint', () => {
  const bothFlags = ['design-v2', 'passkeys'];
  const featureRequest = () => prepareRequest({ headers: {}, url: '/players/me/features' });

  afterEach(() => localStorage.clear());

  it('remembers both flags on logout and starts over for the next sign-in', () => {
    api.setToken('first-player');
    const config = featureRequest();
    handleResponse({ config, data: { features: ['design-v2'] } });
    expect(shouldOfferPasskeyLogin()).toBe(false);
    handleResponse({ config, data: { features: ['passkeys'] } });
    expect(shouldOfferPasskeyLogin()).toBe(false);
    handleResponse({ config, data: { features: bothFlags } });
    expect(shouldOfferPasskeyLogin()).toBe(true);
    api.clearToken();
    expect(shouldOfferPasskeyLogin()).toBe(true);
    api.setToken('next-player');
    expect(shouldOfferPasskeyLogin()).toBe(false);
  });

  it('ignores a feature response from a previous account', () => {
    api.setToken('first-player');
    const config = featureRequest();
    api.setToken('next-player');
    handleResponse({ config, data: { features: bothFlags } });
    expect(shouldOfferPasskeyLogin()).toBe(false);
  });
});

describe('the push notification calls', () => {
  beforeEach(() => {
    mockInstance.get.mockReset();
    mockInstance.post.mockReset();
    mockInstance.put.mockReset();
    mockInstance.delete.mockReset();
  });

  it('reads the public key, empty when the backend has none', async () => {
    mockInstance.get.mockResolvedValue({ data: { public_key: 'BEl62' } });
    await expect(api.getPushPublicKey()).resolves.toBe('BEl62');
    expect(mockInstance.get).toHaveBeenCalledWith('/push/public-key');
    mockInstance.get.mockResolvedValue({ data: {} });
    await expect(api.getPushPublicKey()).resolves.toBe('');
  });

  it('posts the subscription as is and deletes by endpoint in the body', async () => {
    mockInstance.post.mockResolvedValue({ data: {} });
    mockInstance.delete.mockResolvedValue({ data: {} });
    const subscription = { endpoint: 'https://push.test/x', keys: { p256dh: 'k', auth: 'a' } };
    await api.savePushSubscription(subscription);
    expect(mockInstance.post).toHaveBeenCalledWith('/players/me/push-subscription', subscription);
    await api.deletePushSubscription('https://push.test/x');
    expect(mockInstance.delete).toHaveBeenCalledWith('/players/me/push-subscription', { data: { endpoint: 'https://push.test/x' } });
  });

  it('reads and updates notification preferences without a group id', async () => {
    const preferences = { kickoff_reminder: false, registration_open: true };
    mockInstance.get.mockResolvedValue({ data: { preferences } });
    await expect(api.getNotificationPreferences()).resolves.toEqual(preferences);
    expect(mockInstance.get).toHaveBeenCalledWith('/players/me/notification-preferences');

    mockInstance.put.mockResolvedValue({ data: { preferences } });
    await expect(api.updateNotificationPreferences({ kickoff_reminder: false })).resolves.toEqual(preferences);
    expect(mockInstance.put).toHaveBeenCalledWith('/players/me/notification-preferences', {
      preferences: { kickoff_reminder: false }
    });

    mockInstance.get.mockResolvedValue({ data: {} });
    await expect(api.getNotificationPreferences()).resolves.toEqual({});
    mockInstance.put.mockResolvedValue({ data: {} });
    await expect(api.updateNotificationPreferences({})).resolves.toEqual({});
  });
});

describe('a session that ends with a 401', () => {
  const failedRequest = () => ({
    response: { status: 401 },
    config: prepareRequest({ headers: {}, url: '/groups/me' }),
  });
  const tokenFor = (version) => `header.${btoa(JSON.stringify({ session_id: 'device-session', version }))}.signature`;

  beforeEach(() => {
    forgetPush.mockClear();
    window.localStorage.clear();
    mockSessionAPI.post.mockReset();
    mockInstance.mockReset();
    // Already on /login, so the interceptor doesn't navigate (jsdom can't).
    window.history.pushState({}, '', '/login');
  });

  it('drops this browser\'s push subscription when a session was open', async () => {
    api.setToken('old-token');
    await expect(handleResponseError(failedRequest())).rejects.toBeTruthy();
    expect(forgetPush).toHaveBeenCalledTimes(1);
    expect(api.getToken()).toBeFalsy();
  });

  it('leaves it alone on a 401 with no session (a wrong password at login)', async () => {
    await expect(handleResponseError({ response: { status: 401 } })).rejects.toBeTruthy();
    expect(forgetPush).not.toHaveBeenCalled();
  });

  it('renews once for concurrent requests and keeps the push subscription', async () => {
    const renewed = tokenFor(2);
    api.setToken(tokenFor(1));
    mockSessionAPI.post.mockResolvedValue({ data: { token: renewed } });
    mockInstance.mockResolvedValue({ status: 200 });

    await Promise.all([handleResponseError(failedRequest()), handleResponseError(failedRequest())]);

    expect(mockSessionAPI.post).toHaveBeenCalledTimes(1);
    expect(mockSessionAPI.post).toHaveBeenCalledWith('/auth/refresh', null, {
      headers: { 'X-Session-ID': 'device-session' },
    });
    expect(mockInstance).toHaveBeenCalledTimes(2);
    expect(mockInstance).toHaveBeenCalledWith(expect.objectContaining({ calciottoRefreshed: true }));
    expect(api.getToken()).toBe(renewed);
    expect(forgetPush).not.toHaveBeenCalled();
  });

  it('preserves the session and notifications while renewal is temporarily unavailable', async () => {
    const original = tokenFor(1);
    api.setToken(original);
    mockSessionAPI.post.mockRejectedValue({ response: { status: 503 } });
    await expect(handleResponseError(failedRequest())).rejects.toMatchObject({ response: { status: 503 } });
    expect(api.getToken()).toBe(original);
    expect(forgetPush).not.toHaveBeenCalled();
  });

  it('drops this browser subscription after renewal confirms the session ended', async () => {
    api.setToken(tokenFor(1));
    mockSessionAPI.post.mockRejectedValue({ response: { status: 401 } });
    await expect(handleResponseError(failedRequest())).rejects.toMatchObject({ response: { status: 401 } });
    expect(api.getToken()).toBeFalsy();
    expect(forgetPush).toHaveBeenCalledTimes(1);
  });
});
