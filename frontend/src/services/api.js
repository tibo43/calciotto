import axios from 'axios';

const API_URL = 'http://backend:8080'; // Replace with your Go API URL

export const getPlayers = () => axios.get(`${API_URL}/players/all`);
export const getPlayer = (id) => axios.get(`${API_URL}/players/${id}`);
export const addPlayer = (player) => axios.post(`${API_URL}/players`, player);
export const updatePlayer = (id, player) => axios.put(`${API_URL}/players/${id}`, player);
export const deletePlayer = (id) => axios.delete(`${API_URL}/players/${id}`);
