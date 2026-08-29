<template>
  <!-- Teleported to <body>: opened from Profile.vue's "Your groups" card,
       same reasoning as CreateGroupModal/GroupSettingsModal (position:fixed
       escaping an ancestor's backdrop-filter, and isDarkModeSnapshot
       reapplying .dark-mode since this DOM subtree is no longer under
       #app). -->
  <Teleport to="body">
    <div class="modal-overlay" :class="{ 'dark-mode': isDarkModeSnapshot }" @click="close">
      <div class="modal-container join-group-modal" @click.stop>
        <div class="modal-header">
          <h3>Join a group</h3>
          <button class="modal-close" @click="close" aria-label="Close" :disabled="isJoining">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <form @submit.prevent="submitJoin">
          <div class="modal-body">
            <p class="modal-hint">Ask an admin of the group for its invite code.</p>

            <div class="form-group">
              <label for="join-group-code">Invite code</label>
              <input
                id="join-group-code"
                v-model="code"
                class="form-input join-code-input"
                type="text"
                placeholder="Invite code"
                autocapitalize="characters"
                :disabled="isJoining"
              >
            </div>

            <p v-if="joinError" class="error-message">{{ joinError }}</p>
          </div>

          <div class="modal-footer">
            <button class="btn-base btn-cancel" type="button" @click="close" :disabled="isJoining">
              Cancel
            </button>
            <button class="btn-base btn-primary" type="submit" :disabled="isJoining || !code.trim()">
              {{ isJoining ? 'Joining...' : 'Join group' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>

<script>
import { joinGroup } from '@/services/api';
import { setActiveGroupId } from '@/services/activeGroup';
import { isDarkModeActive } from '@/services/theme';

export default {
  name: 'JoinGroupModal',
  emits: ['close'],
  data() {
    return {
      isDarkModeSnapshot: isDarkModeActive(),
      code: '',
      isJoining: false,
      joinError: ''
    };
  },
  methods: {
    async submitJoin() {
      const code = this.code.trim();
      if (!code) {
        return;
      }
      this.isJoining = true;
      this.joinError = '';
      try {
        const group = await joinGroup(code);
        // Same reasoning as creating: switch straight into the group just
        // joined rather than leaving the player on whatever was active.
        setActiveGroupId(group.id);
        window.location.reload();
      } catch (error) {
        this.joinError = this.backendMessage(error, 'Failed to join the group.');
        this.isJoining = false;
      }
    },
    close() {
      if (this.isJoining) {
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
.join-group-modal {
  max-width: 26rem;
}

.modal-hint {
  color: var(--text-secondary);
  font-size: 0.875rem;
  margin: 0 0 1.25rem;
}

.join-code-input {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}
</style>
