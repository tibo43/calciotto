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
export const getMatchDetailsByID = async (id) => {
    const response = await axios.get(`${API_URL}/matches//details/${id}`);
    return response.data; // Ensure this returns an array
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
