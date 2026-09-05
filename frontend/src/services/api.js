import axios from 'axios';

const API_BASE_URL = process.env.VUE_APP_API_BASE_URL || 'http://127.0.0.1:8080';

const TOKEN_KEY = 'calciotto-token';

export const getToken = () => localStorage.getItem(TOKEN_KEY);
export const setToken = (token) => localStorage.setItem(TOKEN_KEY, token);
export const clearToken = () => localStorage.removeItem(TOKEN_KEY);

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  }
});

api.interceptors.request.use((config) => {
  const token = getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.status === 401) {
      clearToken();
      if (window.location.pathname !== '/login') {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);

// Auth
export const login = async (email, password) => {
  const response = await api.post('/auth/login', { email, password });
  return response.data;
};

export const signup = async (name, email, password, inviteCode = '') => {
  const response = await api.post('/auth/signup', { name, email, password, invite_code: inviteCode });
  return response.data;
};

// Always resolves with the same generic message whether or not the email is
// registered — the backend deliberately doesn't say, so callers must not try to
// infer it either.
export const forgotPassword = async (email) => {
  const response = await api.post('/auth/forgot-password', { email });
  return response.data;
};

// 400 covers every token failure alike (unknown, expired, already used) — the
// backend gives one generic message on purpose.
export const resetPassword = async (token, newPassword) => {
  const response = await api.post('/auth/reset-password', { token, new_password: newPassword });
  return response.data;
};

// Builds a `?a=b&c=d` suffix out of the params that actually have a value.
//
// Both params it carries are optional in the same way — leaving one out asks
// the backend for its own default rather than for nothing. No season means
// every season mixed together (the behaviour before seasons existed); no
// group_id means the caller's first group (resolveGroupID's fallback), which
// is only ever the right answer for a player who belongs to a single group.
const query = (params) => {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value) {
      search.append(key, value);
    }
  });
  const queryString = search.toString();
  return queryString ? `?${queryString}` : '';
};

// Matches
// season is optional and behaves exactly like it does on the standings
// endpoints below: leaving it out asks for every season at once.
export const getMatchesDetails = async (groupId, season) => {
  try {
    const response = await api.get(`/matches/details${query({ group_id: groupId, season })}`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch matches details');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching matches details:', error);
    throw error;
  }
};

// New function for getting single match details by ID
export const getMatchDetailsByID = async (matchId, groupId) => {
  try {
    const response = await api.get(`/matches/${matchId}/details${query({ group_id: groupId })}`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch match details');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching match details:', error);
    throw error;
  }
};

// The match is created in groupId when one is given. The key has to be the
// snake_case `group_id`: CreateMatch binds the body into models.Match, whose
// tag is `json:"group_id"`, and Go's case-insensitive fallback does not see
// past the underscore — a `GroupID` key would silently bind to nothing and
// the match would land in the player's first group instead.
//
// `scheduling` is optional and, when given, must be complete: the backend
// treats scheduled_at/registration_opens_at/max_players as all-or-nothing and
// 400s on one or two of the three. Omitting it entirely creates an ordinary
// match, exactly as before — the payload is then byte-identical to the
// unscheduled one, since nothing is spread in.
//
// The two timestamps must be RFC3339 carrying the browser's own UTC offset
// (see services/datetime.js): the backend derives the match's `date` from
// scheduled_at's calendar day in the offset it was sent in, so a `Z`-suffixed
// value would file a late-evening kick-off under the wrong day. `date` is only
// read for an unscheduled match.
export const createMatch = async (matchData, groupId, scheduling) => {
  try {
    let payload = groupId ? { ...matchData, group_id: groupId } : matchData;
    if (scheduling) {
      payload = {
        ...payload,
        scheduled_at: scheduling.scheduledAt,
        registration_opens_at: scheduling.registrationOpensAt,
        max_players: scheduling.maxPlayers,
      };
    }
    const response = await api.post(`/matches`, payload);
    if (response.status !== 200) {
      throw new Error('Failed to create match');
    }

    return response.data;
  } catch (error) {
    console.error('Error creating match:', error);
    throw error;
  }
};

// New function for updating a match
export const updateMatch = async (matchId, matchData) => {
  try {
    const response = await api.put(`/matches/${matchId}`, matchData);
    if (response.status !== 200) {
      throw new Error('Failed to update match');
    }

    return response.data;
  } catch (error) {
    console.error('Error updating match:', error);
    throw error;
  }
};

// Admin-only: permanently removes a match and its rosters. groupId travels as
// a query param (a DELETE has no body), same as getMatchesDetails/
// getMatchDetailsByID above.
export const deleteMatch = async (matchId, groupId) => {
  try {
    const response = await api.delete(`/matches/${matchId}${query({ group_id: groupId })}`);
    if (response.status !== 200) {
      throw new Error('Failed to delete match');
    }
    return response.data;
  } catch (error) {
    console.error('Error deleting match:', error);
    throw error;
  }
};

// Sign-ups on a scheduled match.
//
// None of these five takes a group_id, unlike every other match call above:
// the backend resolves the group from the match named in the path and
// authorizes against *that* (requireGroupMemberByMatchID /
// requireGroupAdminByMatchID in main.go). Passing one would be the hole that
// design closes — a caller could name a match in group A while presenting a
// group B they happen to belong to.
//
// A non-member gets 404, not 403, so this API surface never confirms that a
// match id exists to someone with no business knowing.

// Signs the *caller* up — the player comes from the JWT, which is why there is
// no body and no player argument.
//
// Reaching MaxPlayers is not an error: the surplus sign-up succeeds with
// IsWaiting true. The resolved entry
// ({ PlayerID, Name, Position, IsWaiting, RegisteredAt }) is therefore the
// only place the caller learns "you are #17, on the waiting list", so callers
// must read it rather than assume a flat success.
export const registerForMatch = async (matchId) => {
  try {
    const response = await api.post(`/matches/${matchId}/registrations`);
    if (response.status !== 200) {
      throw new Error('Failed to sign up for the match');
    }
    return response.data;
  } catch (error) {
    console.error('Error signing up for match:', error);
    throw error;
  }
};

// Withdraws the caller's own sign-up. Gated on the same window as signing up:
// once an admin has closed the list (or kick-off has passed) this returns 409
// too, which is why the UI hides the button rather than letting it fail.
export const unregisterFromMatch = async (matchId) => {
  try {
    const response = await api.delete(`/matches/${matchId}/registrations`);
    if (response.status !== 200) {
      throw new Error('Failed to withdraw from the match');
    }
    return response.data;
  } catch (error) {
    console.error('Error withdrawing from match:', error);
    throw error;
  }
};

// The ordered list, `[]` when empty. Position is 1-based and contiguous, and
// IsWaiting is server-derived: the first MaxPlayers entries are the confirmed
// roster. Never recompute that split from MaxPlayers on this side — the two
// would disagree the moment an admin changes the cap.
export const getMatchRegistrations = async (matchId) => {
  try {
    const response = await api.get(`/matches/${matchId}/registrations`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch match registrations');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching match registrations:', error);
    throw error;
  }
};

// Admin-only: freezes the roster so the teams can be composed.
export const closeMatchRegistrations = async (matchId) => {
  try {
    const response = await api.post(`/matches/${matchId}/registrations/close`);
    if (response.status !== 200) {
      throw new Error('Failed to close sign-ups');
    }
    return response.data;
  } catch (error) {
    console.error('Error closing match sign-ups:', error);
    throw error;
  }
};

// Admin-only: undoes a mis-clicked close. It only clears the closed flag — it
// cannot bring back a list that kick-off has already closed.
export const reopenMatchRegistrations = async (matchId) => {
  try {
    const response = await api.post(`/matches/${matchId}/registrations/reopen`);
    if (response.status !== 200) {
      throw new Error('Failed to reopen sign-ups');
    }
    return response.data;
  } catch (error) {
    console.error('Error reopening match sign-ups:', error);
    throw error;
  }
};

// Man of the Match voting. Like the five sign-up calls above, none of these
// three carry a group_id: the backend derives the group from the match named
// in the path (RequireGroupMembershipByMatchPathParam), so sending one would
// be the same hole that design already closes.

// Casts or replaces the *caller's* vote — the voter comes from the JWT, so
// there is no voter argument, only the id of the player being voted for.
// Unlike registerForMatch, calling this again after already voting is not an
// error: it is an upsert, and the response (the same shape getMatchVotes
// returns) reflects the *new* choice.
export const voteForMotm = async (matchId, votedForId) => {
  try {
    const response = await api.post(`/matches/${matchId}/votes`, { voted_for_id: votedForId });
    if (response.status !== 200) {
      throw new Error('Failed to cast Man of the Match vote');
    }
    return response.data;
  } catch (error) {
    console.error('Error casting Man of the Match vote:', error);
    throw error;
  }
};

// Removes the caller's own vote, if any. A no-op success when they had not
// voted — never a 404 — so this never needs special-casing "nothing to undo".
export const removeMotmVote = async (matchId) => {
  try {
    const response = await api.delete(`/matches/${matchId}/votes`);
    if (response.status !== 200) {
      throw new Error('Failed to remove Man of the Match vote');
    }
    return response.data;
  } catch (error) {
    console.error('Error removing Man of the Match vote:', error);
    throw error;
  }
};

// The tally ({ Tally: [{ PlayerID, Name, Votes }], MyVoteFor }), open to any
// member — seeing who is leading isn't privileged, only voting is scoped to
// the caller's own choice.
export const getMatchVotes = async (matchId) => {
  try {
    const response = await api.get(`/matches/${matchId}/votes`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch Man of the Match votes');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching Man of the Match votes:', error);
    throw error;
  }
};

// New function for updating a match
export const createPlayer = async (playerData) => {
  try {
    const response = await api.post(`/players`, playerData);
    if (response.status !== 200) {
      throw new Error('Failed to create player');
    }

    return response.data;
  } catch (error) {
    console.error('Error creating player:', error);
    throw error;
  }
};

// Standings
export const getPointsStandings = async (season, groupId) => {
  try {
    const response = await api.get(`/standings/points${query({ season, group_id: groupId })}`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch points standings');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching points standings:', error);
    throw error;
  }
};

export const getScorers = async (season, groupId) => {
  try {
    const response = await api.get(`/standings/scorers${query({ season, group_id: groupId })}`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch scorers');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching scorers:', error);
    throw error;
  }
};

export const getMotmStandings = async (season, groupId) => {
  try {
    const response = await api.get(`/standings/motm${query({ season, group_id: groupId })}`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch Man of the Match standings');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching Man of the Match standings:', error);
    throw error;
  }
};

export const getSeasons = async (groupId) => {
  try {
    const response = await api.get(`/standings/seasons${query({ group_id: groupId })}`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch seasons');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching seasons:', error);
    throw error;
  }
};

// Player profile
// Cross-group stats of the authenticated player themselves — the backend
// takes the player from the JWT, so there is no id to pass here. season is
// optional and applies to both the overall totals and the per-group rows.
export const getPlayerProfile = async (season) => {
  try {
    const response = await api.get(`/players/me/stats${query({ season })}`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch player profile');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching player profile:', error);
    throw error;
  }
};

// Lets the authenticated player rename themselves. Same "no id needed" shape
// as getPlayerProfile — the backend takes the player from the JWT — but this
// one rejects with a 400 (surfaced via error.response.data.error) if another
// player anywhere already holds the requested name.
export const updateMyName = async (name) => {
  const response = await api.patch('/players/me', { name });
  return response.data;
};

// Groups
// Only /groups/me is scoped to the caller — the plain GET /groups is public
// and lists every group in the system, which is never what this app wants.
export const getMyGroups = async () => {
  try {
    const response = await api.get(`/groups/me`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch my groups');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching my groups:', error);
    throw error;
  }
};

// The caller becomes the new group's first member, server-side. teams must
// be exactly the group's two team specs, e.g.
// [{ name: 'Les Rouges', colour: 'red' }, { name: 'Les Bleus', colour: 'blue' }]
// — the backend rejects anything but exactly 2.
export const createGroup = async (name, teams) => {
  try {
    const response = await api.post(`/groups`, { name, teams });
    if (response.status !== 200) {
      throw new Error('Failed to create group');
    }
    return response.data;
  } catch (error) {
    console.error('Error creating group:', error);
    throw error;
  }
};

// 404 means "unknown code", 400 means "already a member" — callers are
// expected to surface the backend's own message rather than flatten both into
// one generic failure.
export const joinGroup = async (inviteCode) => {
  try {
    const response = await api.post(`/groups/join`, { invite_code: inviteCode });
    if (response.status !== 200) {
      throw new Error('Failed to join group');
    }
    return response.data;
  } catch (error) {
    console.error('Error joining group:', error);
    throw error;
  }
};

// The invite code never rides along in the group JSON (it's json:"-" on the
// Go model), so it has to be fetched per group, on demand.
export const getInviteCode = async (groupId) => {
  try {
    const response = await api.get(`/groups/${groupId}/invite-code`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch invite code');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching invite code:', error);
    throw error;
  }
};

// Members — fetched on demand per group, same pattern as the invite code and
// teams above: a group's roster isn't part of the group JSON either. Each
// entry is a Player (lowercase fields: id, name, email) plus its role
// ("admin"/"member") in this group — a different casing convention than
// getMyGroups' entries, since this mirrors Player's own JSON tags rather than
// PlayerCustom's PascalCase ones.
export const getGroupMembers = async (groupId) => {
  try {
    const response = await api.get(`/groups/${groupId}/players`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch group members');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching group members:', error);
    throw error;
  }
};

// Admin-only: promotes/demotes a member. role must be "admin" or "member" —
// the backend rejects anything else, plus self-targeting and demoting the
// group's last admin (services.ErrCannotChangeOwnRole/ErrLastAdmin).
export const updateMemberRole = async (groupId, playerId, role) => {
  try {
    const response = await api.patch(`/groups/${groupId}/members/${playerId}/role`, { role });
    if (response.status !== 200) {
      throw new Error('Failed to update member role');
    }
    return response.data;
  } catch (error) {
    console.error('Error updating member role:', error);
    throw error;
  }
};

// Admin-only: removes a member from the group. This only ever deletes the
// GroupMembership row — it never touches the player's match history
// (MatchPlayer rows are untouched), so their past goals/standings survive.
export const removeMember = async (groupId, playerId) => {
  try {
    const response = await api.delete(`/groups/${groupId}/members/${playerId}`);
    if (response.status !== 200) {
      throw new Error('Failed to remove member');
    }
    return response.data;
  } catch (error) {
    console.error('Error removing member:', error);
    throw error;
  }
};

// Admin-only: attaches an email to a "ghost" member (one with a Name but no
// Email, created via createPlayer's admin-only ghost flow) and sends them a
// link to set their own password — the frontend counterpart to
// AuthService.InviteExistingPlayer. Rejects with the backend's own message
// for an already-claimed player, an email already used elsewhere, etc.
export const invitePlayer = async (groupId, playerId, email) => {
  try {
    const response = await api.post(`/groups/${groupId}/members/${playerId}/invite`, { email });
    if (response.status !== 200) {
      throw new Error('Failed to invite player');
    }
    return response.data;
  } catch (error) {
    console.error('Error inviting player:', error);
    throw error;
  }
};

// Self-service: marks groupId as the caller's favorite — the group
// resolveActiveGroup() falls back to on a fresh device/session instead of an
// arbitrary "first group" ordering. Every group a player belongs to keeps
// exactly one favorite; this only moves the flag, it never turns it off.
export const setFavoriteGroup = async (groupId) => {
  try {
    const response = await api.patch(`/groups/${groupId}/favorite`);
    if (response.status !== 200) {
      throw new Error('Failed to set favorite group');
    }
    return response.data;
  } catch (error) {
    console.error('Error setting favorite group:', error);
    throw error;
  }
};

// Teams — fetched on demand per group, same pattern as the invite code
// above: a group's teams aren't part of the group JSON either.
export const getTeamsByGroup = async (groupId) => {
  try {
    const response = await api.get(`/groups/${groupId}/teams`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch teams');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching teams:', error);
    throw error;
  }
};

// Admin-only: renames a team and/or changes its colour. The backend replaces
// both fields wholesale (no partial patch), so both must always be sent.
export const updateTeam = async (groupId, teamId, name, colour) => {
  try {
    const response = await api.patch(`/groups/${groupId}/teams/${teamId}`, { name, colour });
    if (response.status !== 200) {
      throw new Error('Failed to update team');
    }
    return response.data;
  } catch (error) {
    console.error('Error updating team:', error);
    throw error;
  }
};
