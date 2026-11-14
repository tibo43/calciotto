<template>
  <!-- Teleported to <body>, same reasoning as JoinGroupModal/CreateGroupModal/
       GroupSettingsModal (position:fixed escaping an ancestor's
       backdrop-filter, and isDarkModeSnapshot reapplying .dark-mode since this
       DOM subtree is no longer under #app). -->
  <Teleport to="body">
    <div class="modal-overlay" :class="{ 'dark-mode': isDarkModeSnapshot }" @click="close">
      <div class="modal-container delete-account-modal" @click.stop>
        <div class="modal-header">
          <h3>Delete my account</h3>
          <button class="modal-close" @click="close" aria-label="Close" :disabled="isDeleting">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <form @submit.prevent="submitDelete">
          <div class="modal-body">
            <!-- This is the one truly irreversible self-service action in the
                 app, unlike e.g. leaving a group or reopening sign-ups, both
                 of which can be undone — the warning is deliberately explicit
                 about what does and doesn't disappear. -->
            <p class="delete-warning">
              This cannot be undone. Your name, email and password will be permanently removed, and you'll be
              taken out of every group you belong to. Match history you're part of (goals, Man of the Match
              awards) is kept for the other players, just no longer linked to a real account of yours.
            </p>

            <div class="form-group">
              <label for="delete-account-password">Confirm your password</label>
              <input
                id="delete-account-password"
                v-model="password"
                class="form-input"
                type="password"
                placeholder="Current password"
                autocomplete="current-password"
                :disabled="isDeleting"
              >
            </div>

            <p v-if="deleteError" class="error-message">{{ deleteError }}</p>
          </div>

          <div class="modal-footer">
            <button class="btn-base btn-cancel" type="button" @click="close" :disabled="isDeleting">
              Cancel
            </button>
            <button class="btn-base btn-danger" type="submit" :disabled="isDeleting || !password">
              {{ isDeleting ? 'Deleting...' : 'Delete my account' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>

<script>
import { deleteMyAccount, clearToken } from '@/services/api';
import { isDarkModeActive } from '@/services/theme';

export default {
  name: 'DeleteAccountModal',
  emits: ['close'],
  data() {
    return {
      isDarkModeSnapshot: isDarkModeActive(),
      password: '',
      isDeleting: false,
      deleteError: ''
    };
  },
  methods: {
    async submitDelete() {
      if (!this.password) {
        return;
      }
      this.isDeleting = true;
      this.deleteError = '';
      try {
        await deleteMyAccount(this.password);
        // The account no longer exists in any usable form on this device —
        // clear the token and leave the app entirely, the same "session is
        // over" contract api.js's own 401 interceptor already follows
        // elsewhere, rather than leaving a routed view to fail on its next
        // request.
        clearToken();
        window.location.href = '/login?accountDeleted=success';
      } catch (error) {
        this.deleteError = this.backendMessage(error, 'Failed to delete your account.');
        this.isDeleting = false;
      }
    },
    close() {
      if (this.isDeleting) {
        return;
      }
      this.$emit('close');
    },
    backendMessage(error, fallback) {
      return error.response?.data?.error || fallback;
    }
  }
};
</script>

<style scoped>
.delete-account-modal {
  max-width: 28rem;
}

.delete-warning {
  color: var(--text-secondary);
  font-size: 0.875rem;
  margin: 0 0 1.25rem;
}
</style>
