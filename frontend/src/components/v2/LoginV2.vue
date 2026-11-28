<template>
  <main class="page is-auth auth-v2 login-v2" :class="{ 'is-waiting': isUsingPasskey }">
    <h1 class="auth-title">{{ $t('auth.login.title') }}</h1>
    <p class="auth-lead">{{ $t('auth.login.lead') }}</p>
    <p v-if="resetConfirmed" class="auth-notice" role="status">{{ $t('auth.login.resetConfirmed') }}</p>
    <p v-if="accountDeleted" class="auth-notice" role="status">{{ $t('auth.login.accountDeleted') }}</p>

    <template v-if="passkeysAvailable">
      <button class="btn btn-ghost btn-block pk-btn" type="button" :disabled="busy" @click="submitPasskey">
        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.8"
          aria-hidden="true"><circle cx="8" cy="15" r="4" /><path d="M11 12l9-9M17 6l3 3M14 9l2 2" /></svg>
        {{ isUsingPasskey ? $t('auth.login.waitingDevice') : $t('auth.login.withPasskey') }}
      </button>
      <div class="or" aria-hidden="true"><span>{{ $t('auth.login.or') }}</span></div>
    </template>

    <form @submit.prevent="submit">
      <div class="field">
        <label class="label" for="email">{{ $t('auth.email') }}</label>
        <input id="email" v-model="email" type="email" required autocomplete="username webauthn">
      </div>
      <div class="field">
        <label class="label" for="password">{{ $t('auth.password') }}</label>
        <input id="password" v-model="password" type="password" required autocomplete="current-password">
      </div>
      <button class="btn btn-block" type="submit" :disabled="busy">{{ isSubmitting ? $t('auth.login.submitting') : $t('auth.login.submit') }}</button>
    </form>

    <div class="auth-links">
      <router-link to="/forgot-password">{{ $t('auth.login.forgot') }}</router-link>
      <router-link to="/signup">{{ $t('auth.login.signup') }}</router-link>
    </div>
    <AppToast ref="toast" />
  </main>
</template>

<script>
import AppToast from '@/components/AppToast.vue';
import { login, setToken } from '@/services/api';
import { signInWithPasskey, supportsPasskeys } from '@/services/passkeys';
import { shouldOfferPasskeyLogin } from '@/services/passkeyAccess';
import { authErrorMessage } from '@/services/v2Auth';

// The login page. Failures go to the shared toast instead of a line on the
// page.
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
    // Same query flags as Login.vue had (ResetPasswordV2.vue, DeleteAccountModal.vue).
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
        this.$refs.toast?.show(authErrorMessage(error, this.$t('auth.login.failed')), 'error');
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
        this.$refs.toast?.show(this.$t('auth.login.passkeyFailed'), 'error');
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

<style scoped src="./authPageV2.css"></style>

<style scoped>
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
</style>
