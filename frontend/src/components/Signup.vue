<template>
  <div class="auth-page">
    <div class="auth-card card-base card-large">
      <h1 class="page-title">Sign up</h1>
      <p class="auth-hint">Pick your name from the existing player list, then set an email and password.</p>

      <form @submit.prevent="submit">
        <div class="form-group">
          <label for="player">You are</label>
          <select id="player" v-model="playerId" class="form-input" required :disabled="isLoadingPlayers">
            <option value="" disabled>{{ isLoadingPlayers ? 'Loading players...' : 'Select your name' }}</option>
            <option v-for="player in players" :key="player.ID" :value="player.ID">{{ player.Name }}</option>
          </select>
        </div>

        <div class="form-group">
          <label for="email">Email</label>
          <input id="email" v-model="email" type="email" class="form-input" required autocomplete="email" />
        </div>

        <div class="form-group">
          <label for="password">Password</label>
          <input id="password" v-model="password" type="password" class="form-input" required
            autocomplete="new-password" />
        </div>

        <p v-if="error" class="error-message">{{ error }}</p>

        <button type="submit" class="btn-base btn-primary btn-large" :disabled="isSubmitting"
          style="width: 100%; justify-content: center;">
          {{ isSubmitting ? 'Signing up...' : 'Sign up' }}
        </button>
      </form>

      <p class="auth-switch">
        Already have an account? <router-link to="/login">Log in</router-link>
      </p>
    </div>
  </div>
</template>

<script>
import { getPlayers, signup } from '@/services/api';

export default {
  name: 'SignupPage',
  data() {
    return {
      players: [],
      isLoadingPlayers: true,
      playerId: '',
      email: '',
      password: '',
      error: '',
      isSubmitting: false,
    };
  },
  async created() {
    try {
      this.players = await getPlayers();
    } catch (err) {
      this.error = 'Failed to load the player list.';
    } finally {
      this.isLoadingPlayers = false;
    }
  },
  methods: {
    async submit() {
      this.error = '';
      this.isSubmitting = true;
      try {
        await signup(this.playerId, this.email, this.password);
        this.$router.push('/login');
      } catch (err) {
        this.error = err.response?.data?.error || 'Signup failed. Please try again.';
      } finally {
        this.isSubmitting = false;
      }
    },
  },
};
</script>

<style scoped>
.auth-page {
  min-height: calc(100vh - var(--navbar-height));
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem 1rem;
}

.auth-card {
  width: 100%;
  max-width: 400px;
}

.auth-card .page-title {
  margin-bottom: 0.5rem;
  text-align: center;
}

.auth-hint {
  color: var(--text-secondary);
  font-size: 0.9rem;
  margin-bottom: 1.5rem;
  text-align: center;
}

.auth-switch {
  margin-top: 1.5rem;
  text-align: center;
  color: var(--text-secondary);
}
</style>
