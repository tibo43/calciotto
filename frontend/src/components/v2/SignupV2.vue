<template>
  <main class="page is-auth auth-v2 has-toast-room signup-v2">
    <h1 class="auth-title">{{ $t('auth.signup.title') }}</h1>
    <p class="auth-lead">{{ $t('auth.signup.lead') }}</p>

    <form @submit.prevent="submit">
      <div class="field">
        <label class="label" for="invite-code">{{ $t('auth.signup.inviteCode') }}</label>
        <input id="invite-code" v-model="inviteCode" class="code-input" type="text" placeholder="AB2N7TQR"
          autocapitalize="characters" autocomplete="off" spellcheck="false" aria-describedby="invite-code-hint">
        <p id="invite-code-hint" class="hint">{{ inviteHint }}</p>
      </div>
      <div class="field">
        <label class="label" for="name">{{ $t('auth.signup.name') }}</label>
        <input id="name" v-model="name" type="text" required autocomplete="name">
      </div>
      <div class="field">
        <label class="label" for="email">{{ $t('auth.email') }}</label>
        <input id="email" v-model="email" type="email" required autocomplete="email">
      </div>
      <div class="field">
        <label class="label" for="password">{{ $t('auth.password') }}</label>
        <input id="password" v-model="password" type="password" required autocomplete="new-password">
      </div>
      <button class="btn btn-block" type="submit" :disabled="isSubmitting">{{ isSubmitting ? $t('auth.signup.submitting') : $t('auth.signup.submit') }}</button>
    </form>

    <div class="auth-links">
      <router-link to="/login">{{ $t('auth.signup.login') }}</router-link>
    </div>
    <AppToast ref="toast" />
  </main>
</template>

<script>
import AppToast from '@/components/AppToast.vue';
import { signup, setToken } from '@/services/api';
import { authErrorMessage } from '@/services/v2Auth';

// The sign up page. An invite code is the only way into a group, so it comes
// first; an admin's shared link (/signup?invite=CODE) fills it in. Failures
// go to the shared toast, as on the login page.
export default {
  name: 'SignupV2',
  components: { AppToast },
  data() {
    const invite = this.$route.query.invite;
    const linkCode = typeof invite === 'string' ? invite : '';
    return {
      linkCode,
      inviteCode: linkCode,
      name: '',
      email: '',
      password: '',
      isSubmitting: false,
    };
  },
  computed: {
    // Says where the code came from while the field still holds the link's.
    inviteHint() {
      return this.linkCode && this.inviteCode === this.linkCode
        ? this.$t('auth.signup.hintFromLink')
        : this.$t('auth.signup.hintAskAdmin');
    },
  },
  methods: {
    async submit() {
      if (this.isSubmitting) return;
      this.$refs.toast?.dismiss();
      // Checked here before any request: the backend answers the same
      // (ErrInviteCodeRequired).
      if (!this.inviteCode.trim()) {
        this.$refs.toast?.show(this.$t('auth.signup.inviteRequired'), 'error');
        return;
      }
      this.isSubmitting = true;
      try {
        // POST /auth/signup returns a session token, as login does: the new
        // player goes straight to the home page.
        const { token } = await signup(this.name, this.email, this.password, this.inviteCode);
        setToken(token);
        this.$router.push('/');
      } catch (error) {
        this.$refs.toast?.show(authErrorMessage(error, this.$t('auth.signup.failed')), 'error');
      } finally {
        this.isSubmitting = false;
      }
    },
  },
};
</script>

<style scoped src="./authPageV2.css"></style>

<style scoped>
.code-input {
  font-family: var(--v2-font-label);
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

/* The placeholder keeps the field's mono capitals. */
.code-input::placeholder {
  letter-spacing: 0.12em;
}
</style>
