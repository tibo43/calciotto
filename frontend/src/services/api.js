import axios from 'axios';

const API_BASE_URL = process.env.VUE_APP_API_BASE_URL

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  }
});

export default api;

// Matches
export const getMatchesDetails = async () => {
  try {
    const response = await api.get(`/api/matches/details`);
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
    const response = await api.get(`/api/matches/${matchId}/details`);
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
    const response = await api.post(`/api/matches`, matchData);
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
    const response = await api.put(`/api/matches/${matchId}`, matchData);
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
    const response = await api.patch(`/api/matches/${matchId}`, updates);
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
    const response = await api.post(`/api/players`, playerData);
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
    const response = await api.get(`/api/players`);
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
    const response = await api.get(`/api/players/search?name=${encodeURIComponent(playerName)}`);
    if (response.status !== 200) {
      throw new Error('Player search failed');
    }
    return response.data;
  } catch (error) {
    console.error('Error searching for player:', error);
    throw error;
  }
}
