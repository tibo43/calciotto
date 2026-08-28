import axios from 'axios';

const API_BASE_URL = process.env.VUE_API_BASE_URL || 'http://127.0.0.1:8080';

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

export const signup = async (playerId, email, password) => {
  const response = await api.post('/auth/signup', { player_id: playerId, email, password });
  return response.data;
};

// Matches
export const getMatchesDetails = async () => {
  try {
    const response = await api.get(`/matches/details`);
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
export const getMatchDetailsByID = async (matchId) => {
  try {
    const response = await api.get(`/matches/${matchId}/details`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch match details');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching match details:', error);
    throw error;
  }
};

// New function for updating a match
export const createMatch = async (matchData) => {
  try {
    const response = await api.post(`/matches`, matchData);
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

// Alternative: PATCH for partial updates (if your API supports it)
export const updateMatchPartial = async (matchId, updates) => {
  try {
    const response = await api.patch(`/matches/${matchId}`, updates);
    if (response.status !== 200) {
      throw new Error('Failed to update match');
    }

    return response.data;
  } catch (error) {
    console.error('Error updating match:', error);
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

// Existing function (referenced in PlayersAll.vue)
export const getPlayers = async () => {
  try {
    const response = await api.get(`/players`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch players');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching players:', error);
    throw error;
  }
};

export async function searchPlayerByName(playerName) {
  try {
    const response = await api.get(`/players/search?name=${encodeURIComponent(playerName)}`);
    if (response.status !== 200) {
      throw new Error('Player search failed');
    }
    return response.data;
  } catch (error) {
    console.error('Error searching for player:', error);
    throw error;
  }
}

// Standings
// season is optional: leaving it out asks the backend for every match, all
// seasons mixed together (the behaviour before seasons existed).
const seasonQuery = (season) => (season ? `?season=${encodeURIComponent(season)}` : '');

export const getPointsStandings = async (season) => {
  try {
    const response = await api.get(`/standings/points${seasonQuery(season)}`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch points standings');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching points standings:', error);
    throw error;
  }
};

export const getScorers = async (season) => {
  try {
    const response = await api.get(`/standings/scorers${seasonQuery(season)}`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch scorers');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching scorers:', error);
    throw error;
  }
};

export const getSeasons = async () => {
  try {
    const response = await api.get(`/standings/seasons`);
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
    const response = await api.get(`/players/me/stats${seasonQuery(season)}`);
    if (response.status !== 200) {
      throw new Error('Failed to fetch player profile');
    }
    return response.data;
  } catch (error) {
    console.error('Error fetching player profile:', error);
    throw error;
  }
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

// The caller becomes the new group's first member, server-side.
export const createGroup = async (name) => {
  try {
    const response = await api.post(`/groups`, { name });
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
