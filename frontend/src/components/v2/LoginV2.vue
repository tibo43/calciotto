<template>
  <main class="page is-auth login-v2" :class="{ 'is-waiting': isUsingPasskey }">
    <h1 class="auth-title">Log in</h1>
    <p class="auth-lead">Find your groups and your matches.</p>
    <p v-if="resetConfirmed" class="auth-notice" role="status">Your password has been updated. Log in with it below.</p>
    <p v-if="accountDeleted" class="auth-notice" role="status">Your account has been deleted.</p>

    <template v-if="passkeysAvailable">
      <button class="btn btn-ghost btn-block pk-btn" type="button" :disabled="busy" @click="submitPasskey">
        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.8"
          aria-hidden="true"><circle cx="8" cy="15" r="4" /><path d="M11 12l9-9M17 6l3 3M14 9l2 2" /></svg>
        {{ isUsingPasskey ? 'Waiting for your device…' : 'Log in with a passkey' }}
      </button>
      <div class="or" aria-hidden="true"><span>or</span></div>
    </template>

    <form @submit.prevent="submit">
      <div class="field">
        <label class="label" for="email">Email</label>
        <input id="email" v-model="email" type="email" required autocomplete="username webauthn">
      </div>
      <div class="field">
        <label class="label" for="password">Password</label>
        <input id="password" v-model="password" type="password" required autocomplete="current-password">
      </div>
      <button class="btn btn-block" type="submit" :disabled="busy">{{ isSubmitting ? 'Logging in…' : 'Log in' }}</button>
    </form>

    <div class="auth-links">
      <router-link to="/forgot-password">Forgot password?</router-link>
      <router-link to="/signup">No account yet? Sign up with an invite code</router-link>
    </div>
    <AppToast ref="toast" />
  </main>
</template>

<script>
import AppToast from '@/components/AppToast.vue';
import { login, setToken } from '@/services/api';
import { signInWithPasskey, supportsPasskeys } from '@/services/passkeys';
import { shouldOfferPasskeyLogin } from '@/services/passkeyAccess';
import { actionErrorMessage } from '@/services/v2Match';

// Design v2 login, shown instead of Login.vue when shouldUseLoginV2() says
// so (LoginGate.vue). Same calls, passkey condition and redirect as
// Login.vue; failures go to the shared toast instead of a line on the page.
export default {
  name: 'LoginV2',
  components: { AppToast },
  data() {
    return {
      email: '',
      password: '',
      isSubmitting: false,
      isUsingPasskey: false,
      passkeysAvailable: supportsPasskeys() && shouldOfferPasskeyLogin(),
    };
  },
  computed: {
    busy() {
      return this.isSubmitting || this.isUsingPasskey;
    },
    // Same query flags as Login.vue (ResetPassword.vue, DeleteAccountModal.vue).
    resetConfirmed() {
      return this.$route.query.reset === 'success';
    },
    accountDeleted() {
      return this.$route.query.accountDeleted === 'success';
    },
  },
  methods: {
    async submit() {
      if (this.busy) return;
      this.$refs.toast?.dismiss();
      this.isSubmitting = true;
      try {
        const { token } = await login(this.email, this.password);
        this.openSession(token);
      } catch (error) {
        this.$refs.toast?.show(actionErrorMessage(error, 'Login failed. Please check your credentials.'), 'error');
      } finally {
        this.isSubmitting = false;
      }
    },
    async submitPasskey() {
      if (this.busy || !this.passkeysAvailable) return;
      this.$refs.toast?.dismiss();
      this.isUsingPasskey = true;
      try {
        const { token } = await signInWithPasskey();
        this.openSession(token);
      } catch {
        this.$refs.toast?.show('Could not log in with a passkey. Try again or use your password.', 'error');
      } finally {
        this.isUsingPasskey = false;
      }
    },
    openSession(token) {
      setToken(token);
      this.$router.push('/');
    },
  },
};
</script>

<style scoped>
/* A 320px column on web; the phone layout uses the full width. */
@media (min-width: 900px) {
  .login-v2.page.is-auth {
    max-width: 360px;
  }
}

.auth-title {
  font-family: var(--v2-font-display);
  font-weight: 800;
  font-size: 40px;
  line-height: 0.95;
  text-transform: uppercase;
  margin-bottom: 8px;
}

.auth-lead {
  margin: 8px 0 28px;
  font-size: 15px;
  color: var(--v2-text-muted);
}

.auth-notice {
  margin: -12px 0 24px;
  font-size: 14px;
  color: var(--v2-text-muted);
}

.pk-btn {
  min-height: 48px;
  gap: 10px;
}

.pk-btn svg {
  flex: none;
}

/* While the device prompt is open both buttons are inert, but they keep
   their colours (the mockup), so the page does not look broken. */
.login-v2.is-waiting .btn[disabled] {
  opacity: 1;
}

.or {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 20px 0;
  font-family: var(--v2-font-label);
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--v2-text-muted);
}

.or::before,
.or::after {
  content: "";
  flex: 1;
  border-top: 1px solid currentColor;
  opacity: 0.35;
}

form .btn-block {
  min-height: 48px;
}

.auth-links {
  margin-top: 20px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
  font-size: 14px;
}

.auth-links a {
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  text-decoration: underline;
  text-underline-offset: 3px;
}

/* No tab bar on this page: the phone toast sits 12px from the bottom. */
@media (max-width: 899px) {
  .login-v2 :deep(.toast-container.toast-bottom) {
    bottom: 12px;
  }

  .login-v2 :deep(.toast-message) {
    margin-bottom: 0;
  }
}
</style>
