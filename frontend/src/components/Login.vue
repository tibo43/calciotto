<template>
  <div class="auth-page">
    <div class="auth-card card-base card-large">
      <h1 class="page-title">Log in</h1>

      <p v-if="resetConfirmed" class="auth-notice">Your password has been updated. Log in with it below.</p>

      <form @submit.prevent="submit">
        <div class="form-group">
          <label for="email">Email</label>
          <input id="email" v-model="email" type="email" class="form-input" required autocomplete="email" />
        </div>

        <div class="form-group">
          <label for="password">Password</label>
          <input id="password" v-model="password" type="password" class="form-input" required
            autocomplete="current-password" />
        </div>

        <p v-if="error" class="error-message">{{ error }}</p>

        <button type="submit" class="btn-base btn-primary btn-large" :disabled="isSubmitting"
          style="width: 100%; justify-content: center;">
          {{ isSubmitting ? 'Logging in...' : 'Log in' }}
        </button>
      </form>

      <p class="auth-switch">
        <router-link to="/forgot-password">Forgot password?</router-link>
      </p>

      <p class="auth-switch auth-switch-secondary">
        No account yet? <router-link to="/signup">Sign up</router-link>
      </p>
    </div>
  </div>
</template>

<script>
import { login, setToken } from '@/services/api';

export default {
  name: 'LoginPage',
  data() {
    return {
      email: '',
      password: '',
      error: '',
      isSubmitting: false,
    };
  },
  computed: {
    // Set by ResetPassword.vue when it hands the user back here after a
    // successful reset.
    resetConfirmed() {
      return this.$route.query.reset === 'success';
    },
  },
  methods: {
    async submit() {
      this.error = '';
      this.isSubmitting = true;
      try {
        const { token } = await login(this.email, this.password);
        setToken(token);
        this.$router.push('/');
      } catch (err) {
        this.error = err.response?.data?.error || 'Login failed. Please check your credentials.';
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
  margin-bottom: 1.5rem;
  text-align: center;
}

.auth-notice {
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

.auth-switch-secondary {
  margin-top: 0.5rem;
}
</style>
