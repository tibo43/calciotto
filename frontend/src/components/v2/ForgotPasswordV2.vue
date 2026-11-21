<template>
  <main class="page is-auth auth-v2 has-toast-room forgot-password-v2">
    <h1 class="auth-title">Forgot password</h1>
    <p class="auth-lead">Enter your email. We'll send you a link to set a new password.</p>
    <p v-if="notice" class="auth-notice" role="status">{{ notice }}</p>

    <form @submit.prevent="submit">
      <div class="field">
        <label class="label" for="email">Email</label>
        <input id="email" v-model="email" type="email" required autocomplete="email">
      </div>
      <button class="btn btn-block" type="submit" :disabled="isSubmitting">{{ buttonLabel }}</button>
    </form>

    <div class="auth-links">
      <router-link to="/login">Back to log in</router-link>
    </div>
    <AppToast ref="toast" />
  </main>
</template>

<script>
import AppToast from '@/components/AppToast.vue';
import { forgotPassword } from '@/services/api';
import { asSentence, authErrorMessage } from '@/services/v2Auth';

// The backend's answer, the same for a registered and an unregistered email.
const REQUESTED = 'If an account exists for this email, a reset link has been sent.';

// The reset request page. The backend's answer becomes a grey line above the
// form and the button offers to send the link again; failures go to the
// shared toast.
export default {
  name: 'ForgotPasswordV2',
  components: { AppToast },
  data() {
    return {
      email: '',
      notice: '',
      isSubmitting: false,
    };
  },
  computed: {
    buttonLabel() {
      if (this.isSubmitting) return 'Sending…';
      return this.notice ? 'Send again' : 'Send the link';
    },
  },
  methods: {
    async submit() {
      if (this.isSubmitting) return;
      this.$refs.toast?.dismiss();
      this.isSubmitting = true;
      try {
        // Shown as the backend words it, as a sentence: nothing here may
        // read as a confirmation that the account exists.
        const { message } = await forgotPassword(this.email);
        this.notice = asSentence(message) || REQUESTED;
      } catch (error) {
        this.notice = '';
        this.$refs.toast?.show(authErrorMessage(error, 'Could not send the reset link. Please try again.'), 'error');
      } finally {
        this.isSubmitting = false;
      }
    },
  },
};
</script>

<style scoped src="./authPageV2.css"></style>
