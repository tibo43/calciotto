<template>
  <main class="page is-auth auth-v2 has-toast-room reset-password-v2">
    <h1 class="auth-title">{{ $t('auth.reset.title') }}</h1>
    <p class="auth-lead">{{ $t('auth.reset.lead') }}</p>

    <form @submit.prevent="submit">
      <div class="field">
        <label class="label" for="password">{{ $t('auth.reset.newPassword') }}</label>
        <input id="password" v-model="newPassword" type="password" required autocomplete="new-password">
      </div>
      <button class="btn btn-block" type="submit" :disabled="isSubmitting">{{ isSubmitting ? $t('common.saving') : $t('auth.reset.save') }}</button>
    </form>

    <div class="auth-links">
      <router-link to="/forgot-password">{{ $t('auth.reset.requestNew') }}</router-link>
    </div>
    <AppToast ref="toast" />
  </main>
</template>

<script>
import AppToast from '@/components/AppToast.vue';
import { resetPassword } from '@/services/api';
import { expiredResetLink, resetErrorMessage } from '@/services/v2Auth';

// The page the emailed reset link opens (/reset-password?token=…). One
// field; a link without a token, or one the backend no longer accepts, says
// so in the shared toast.
export default {
  name: 'ResetPasswordV2',
  components: { AppToast },
  data() {
    return {
      newPassword: '',
      isSubmitting: false,
    };
  },
  computed: {
    // The token rides in the query string of the emailed link, not in the path.
    token() {
      const { token } = this.$route.query;
      return typeof token === 'string' ? token : '';
    },
  },
  methods: {
    async submit() {
      if (this.isSubmitting) return;
      this.$refs.toast?.dismiss();
      if (!this.token) {
        this.$refs.toast?.show(expiredResetLink(), 'error');
        return;
      }
      this.isSubmitting = true;
      try {
        await resetPassword(this.token, this.newPassword);
        // LoginV2 turns the flag into a confirmation line.
        this.$router.push({ path: '/login', query: { reset: 'success' } });
      } catch (error) {
        this.$refs.toast?.show(resetErrorMessage(error), 'error');
      } finally {
        this.isSubmitting = false;
      }
    },
  },
};
</script>

<style scoped src="./authPageV2.css"></style>
