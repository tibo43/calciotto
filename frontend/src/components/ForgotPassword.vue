<template>
  <div class="auth-page">
    <div class="auth-card card-base card-large">
      <h1 class="page-title">Forgot password</h1>
      <p class="auth-hint">Enter your email and we'll send you a link to set a new password.</p>

      <form @submit.prevent="submit">
        <div class="form-group">
          <label for="email">Email</label>
          <input id="email" v-model="email" type="email" class="form-input" required autocomplete="email" />
        </div>

        <p v-if="error" class="error-message">{{ error }}</p>
        <p v-if="message" class="auth-notice">{{ message }}</p>

        <button type="submit" class="btn-base btn-primary btn-large" :disabled="isSubmitting"
          style="width: 100%; justify-content: center;">
          {{ isSubmitting ? 'Sending...' : 'Send reset link' }}
        </button>
      </form>

      <p class="auth-switch">
        Remembered it? <router-link to="/login">Log in</router-link>
      </p>
    </div>
  </div>
</template>

<script>
import { forgotPassword } from '@/services/api';

export default {
  name: 'ForgotPasswordPage',
  data() {
    return {
      email: '',
      message: '',
      error: '',
      isSubmitting: false,
    };
  },
  methods: {
    async submit() {
      this.error = '';
      this.message = '';
      this.isSubmitting = true;
      try {
        // The backend answers the same thing for a registered and an
        // unregistered email, so its message is displayed verbatim rather than
        // being rephrased into anything that might read as a confirmation that
        // the account exists.
        const { message } = await forgotPassword(this.email);
        this.message = message;
      } catch (err) {
        this.error = err.response?.data?.error || 'Could not send the reset link. Please try again.';
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

.auth-notice {
  color: var(--text-secondary);
  font-size: 0.875rem;
  margin-top: 0.5rem;
}

.auth-switch {
  margin-top: 1.5rem;
  text-align: center;
  color: var(--text-secondary);
}
</style>
