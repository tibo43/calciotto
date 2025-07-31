<template>
  <div>
    <h1>Players</h1>
    <ul>
      <li v-for="player in players" :key="player.id">
        {{ player.name }}
        <router-link :to="`/players/${player.id}`">Details</router-link>
      </li>
    </ul>
  </div>
</template>

<script>
import { getPlayers } from '@/services/api';

export default {
  data() {
    return {
      players: [],
    };
  },
  async created() {
    try {
      const response = await getPlayers();
      this.players = response.data;
    } catch (error) {
      console.error('Error fetching players:', error);
    }
  },
};
</script>
