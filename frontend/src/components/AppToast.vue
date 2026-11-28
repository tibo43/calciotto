<template>
  <!-- The app's one toast: bottom right (above the mobile tab bar), a real
       close button, gone on its own after TOAST_DURATION_MS. A parent holds
       it by ref and calls show(message, type) / dismiss(). -->
  <transition name="app-toast">
    <div v-if="message" class="toast-container toast-bottom" role="status" aria-live="polite">
      <div class="toast-message" :class="type" :data-label="type === 'error' ? $t('common.error') : $t('common.done')">
        <span>{{ message }}</span>
        <button type="button" class="toast-close" :aria-label="$t('common.dismiss')" @click="dismiss">×</button>
      </div>
    </div>
  </transition>
</template>

<script>
// Long enough to read a sentence, short enough not to linger.
export const TOAST_DURATION_MS = 5000;

export default {
  name: 'AppToast',
  data() {
    return {
      message: '',
      type: 'success',
      timeout: null
    };
  },
  beforeUnmount() {
    clearTimeout(this.timeout);
  },
  methods: {
    // Showing a new message replaces the current one and restarts the timer.
    show(message, type = 'success') {
      clearTimeout(this.timeout);
      this.message = message;
      this.type = type;
      this.timeout = setTimeout(this.dismiss, TOAST_DURATION_MS);
    },
    dismiss() {
      clearTimeout(this.timeout);
      this.timeout = null;
      this.message = '';
    }
  }
};
</script>

<style scoped>
/* A 44px close button, flush with the right edge, with the text kept
   clear of it. */
.toast-container .toast-message {
  padding-right: 56px;
}

.toast-message .toast-close {
  right: 0;
  width: 44px;
  height: 44px;
}

.app-toast-leave-active {
  animation: toastSlideOut 0.3s cubic-bezier(0.4, 0, 1, 1) forwards;
}
</style>
