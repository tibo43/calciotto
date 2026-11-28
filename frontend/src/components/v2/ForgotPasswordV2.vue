<template>
  <main class="page is-auth auth-v2 has-toast-room forgot-password-v2">
    <h1 class="auth-title">{{ $t('auth.forgot.title') }}</h1>
    <p class="auth-lead">{{ $t('auth.forgot.lead') }}</p>
    <p v-if="notice" class="auth-notice" role="status">{{ showText(notice) }}</p>

    <form @submit.prevent="submit">
      <div class="field">
        <label class="label" for="email">{{ $t('auth.email') }}</label>
        <input id="email" v-model="email" type="email" required autocomplete="email">
      </div>
      <button class="btn btn-block" type="submit" :disabled="isSubmitting">{{ buttonLabel }}</button>
    </form>

    <div class="auth-links">
      <router-link to="/login">{{ $t('auth.forgot.backToLogin') }}</router-link>
    </div>
    <AppToast ref="toast" />
  </main>
</template>

<script>
import AppToast from '@/components/AppToast.vue';
import { forgotPassword } from '@/services/api';
import { messageKey, showText } from '@/i18n';
import { asSentence, authErrorMessage } from '@/services/v2Auth';

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
      if (this.isSubmitting) return this.$t('auth.forgot.sending');
      return this.notice ? this.$t('auth.forgot.sendAgain') : this.$t('auth.forgot.send');
    },
  },
  methods: {
    showText,
    async submit() {
      if (this.isSubmitting) return;
      this.$refs.toast?.dismiss();
      this.isSubmitting = true;
      try {
        // Shown as the backend words it, as a sentence: nothing here may
        // read as a confirmation that the account exists.
        const { message } = await forgotPassword(this.email);
        // The fallback is the backend's answer, the same for a registered
        // and an unregistered email.
        this.notice = asSentence(message) || messageKey('auth.forgot.requested');
      } catch (error) {
        this.notice = '';
        this.$refs.toast?.show(authErrorMessage(error, this.$t('auth.forgot.failed')), 'error');
      } finally {
        this.isSubmitting = false;
      }
    },
  },
};
</script>

<style scoped src="./authPageV2.css"></style>
