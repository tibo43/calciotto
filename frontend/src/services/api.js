import axios from 'axios';

const API_URL = 'http://backend:8080'; // Replace with your Go API URL

// Players
export const getPlayers = () => axios.get(`${API_URL}/players/all`);
export const getPlayer = async (id) => {
    const response = await axios.get(`${API_URL}/players/${id}`);
    return response.data; // Ensure this returns an array
};
export const addPlayer = (player) => axios.post(`${API_URL}/players`, player);
// export const updatePlayer = (id, player) => axios.put(`${API_URL}/players/${id}`, player);
// export const deletePlayer = (id) => axios.delete(`${API_URL}/players/${id}`);
// Matches
export const getMatches = async () => {
    const response = await axios.get(`${API_URL}/matches/all`);
    return response.data; // Ensure this returns an array
};
export const getMatchesDetails = async () => {
    const response = await axios.get(`${API_URL}/matches/details/all`);
    return response.data; // Ensure this returns an array
};
export const getMatchDetailsByID = async (matchId) => {
  try {
    const response = await fetch(`${API_URL}/matches/details/${matchId}`);
    if (!response.ok) {
      throw new Error('Failed to fetch match details');
    }
    return await response.json();
  } catch (error) {
    console.error('Error fetching match details:', error);
    throw error;
  }
};
// Update match
export const updateMatch = async (matchId, matchData) => {
  try {
    const response = await fetch(`${API_URL}/matches/${matchId}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(matchData),
    });
    
    if (!response.ok) {
      throw new Error('Failed to update match');
    }
    
    return await response.json();
  } catch (error) {
    console.error('Error updating match:', error);
    throw error;
  }
};

// Alternative: If you prefer PATCH for partial updates
export const updateMatchPartial = async (matchId, updates) => {
  try {
    const response = await fetch(`${API_URL}/matches/${matchId}`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(updates),
    });
    
    if (!response.ok) {
      throw new Error('Failed to update match');
    }
    
    return await response.json();
  } catch (error) {
    console.error('Error updating match:', error);
    throw error;
  }
};
export const getMatch = (id) => axios.get(`${API_URL}/matches/${id}`);
export const addMatch = (match) => axios.post(`${API_URL}/players`, match);
// export const updatePlayer = (id, player) => axios.put(`${API_URL}/players/${id}`, player);
// export const deletePlayer = (id) => axios.delete(`${API_URL}/players/${id}`);
// Teams
export const getTeams = () => axios.get(`${API_URL}/teams/all`);
export const getTeam = async (id) => {
    const response = await axios.get(`${API_URL}/teams/${id}`);
    return response.data; // Ensure this returns an array
};
export const addTeam = (team) => axios.post(`${API_URL}/teams`, team);
// export const updateTeam = (id, team) => axios.put(`${API_URL}/teams/${id}`, team);
// export const deleteTeam = (id) => axios.delete(`${API_URL}/teams/${id}`);
// Compositions
export const getCompositions = async () => {
    const response = await axios.get(`${API_URL}/compositions/all`);
    return response.data; // Ensure this returns an array
};
export const getCompositionByMatchID = async (id) => {
    const response = await axios.get(`${API_URL}/compositions/match/${id}`);
    return response.data; // Ensure this returns an array
};
export const getCompositionByTeamID = async (id) => {
    const response = await axios.get(`${API_URL}/compositions/team/${id}`);
    return response.data; // Ensure this returns an array
};
