<template>
  <div class="auth-page">
    <div class="auth-card card-base card-large">
      <h1 class="page-title">Reset password</h1>
      <p class="auth-hint">Choose a new password for your account.</p>

      <form @submit.prevent="submit">
        <div class="form-group">
          <label for="password">New password</label>
          <input id="password" v-model="newPassword" type="password" class="form-input" required
            autocomplete="new-password" />
        </div>

        <p v-if="error" class="error-message">{{ error }}</p>

        <button type="submit" class="btn-base btn-primary btn-large" :disabled="isSubmitting"
          style="width: 100%; justify-content: center;">
          {{ isSubmitting ? 'Saving...' : 'Set new password' }}
        </button>
      </form>

      <p class="auth-switch">
        <router-link to="/forgot-password">Request a new link</router-link>
      </p>
    </div>
  </div>
</template>

<script>
import { resetPassword } from '@/services/api';

export default {
  name: 'ResetPasswordPage',
  data() {
    return {
      newPassword: '',
      error: '',
      isSubmitting: false,
    };
  },
  computed: {
    // The token rides in the query string of the emailed link, not in the path.
    token() {
      return this.$route.query.token || '';
    },
  },
  methods: {
    async submit() {
      this.error = '';
      if (!this.token) {
        this.error = 'This reset link is incomplete. Request a new one.';
        return;
      }
      this.isSubmitting = true;
      try {
        await resetPassword(this.token, this.newPassword);
        // Same pattern as Signup.vue: hand the user off to the login page,
        // with a flag Login.vue turns into a confirmation notice.
        this.$router.push({ path: '/login', query: { reset: 'success' } });
      } catch (err) {
        this.error = err.response?.data?.error || 'Could not reset the password. Please try again.';
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
