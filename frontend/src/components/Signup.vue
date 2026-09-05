<template>
  <div class="auth-page">
    <div class="auth-card card-base card-large">
      <h1 class="page-title">Sign up</h1>
      <p class="auth-hint">Create your account to start joining and organizing matches.</p>

      <form @submit.prevent="submit">
        <div class="form-group">
          <label for="name">Name</label>
          <input id="name" v-model="name" type="text" class="form-input" required autocomplete="name" />
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

        <div class="form-group">
          <label for="invite-code">Invite code</label>
          <input id="invite-code" v-model="inviteCode" type="text" class="form-input" placeholder="e.g. AB2N7TQR"
            autocapitalize="characters" required />
          <p class="field-hint">Required — ask your group admin for the invite code.</p>
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
import { signup, setToken } from '@/services/api';

export default {
  name: 'SignupPage',
  data() {
    return {
      name: '',
      email: '',
      password: '',
      inviteCode: '',
      error: '',
      isSubmitting: false,
    };
  },
  created() {
    // Prefills from an admin's shared invite link (/signup?invite=CODE, see
    // GroupSettingsModal.vue's WhatsApp invite button) so a new player never
    // has to type the code by hand.
    const invite = this.$route.query.invite;
    if (typeof invite === 'string' && invite) {
      this.inviteCode = invite;
    }
  },
  methods: {
    async submit() {
      this.error = '';
      // Self-service group creation/joining is disabled, so signup is now
      // the only way into a group — an empty code would leave a player
      // stranded with no group, so this is checked client-side before ever
      // reaching the network (the backend rejects it too, with the same
      // message, via ErrInviteCodeRequired).
      if (!this.inviteCode.trim()) {
        this.error = 'Invite code is required.';
        return;
      }
      this.isSubmitting = true;
      try {
        // POST /auth/signup now returns a ready-to-use token alongside the
        // player id — the same shape as Login — so signing up logs the
        // player straight in, the same way Login.vue does, instead of
        // bouncing them to /login to re-enter the password they just typed.
        const { token } = await signup(this.name, this.email, this.password, this.inviteCode);
        setToken(token);
        this.$router.push('/');
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

.field-hint {
  color: var(--text-secondary);
  font-size: 0.8rem;
  margin: 0.35rem 0 0;
}
</style>
