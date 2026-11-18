<template>
  <details v-if="expiresAt" class="invitation-actions">
    <summary>Manage invitation</summary>
    <p v-if="code">Expires {{ new Date(expiresAt).toLocaleString() }}</p>
    <p v-else>No active invitation.</p>
    <div class="invitation-buttons">
      <button class="btn-base btn-cancel btn-small" type="button" :disabled="saving" @click="update(true)">
        Generate new code
      </button>
      <button v-if="code" class="btn-base btn-cancel btn-small" type="button" :disabled="saving" @click="update(false)">
        Revoke code
      </button>
    </div>
    <p v-if="error" class="error-message" role="alert">{{ error }}</p>
  </details>
</template>

<script>
import { rotateGroupInvitation, revokeGroupInvitation } from '@/services/api';

export default {
  name: 'InvitationActions',
  props: {
    groupId: { type: String, required: true },
    code: { type: String, default: '' },
    expiresAt: { type: String, default: '' }
  },
  emits: ['updated'],
  data() {
    return { saving: false, error: '' };
  },
  methods: {
    async update(rotate) {
      const groupId = this.groupId;
      this.saving = true;
      this.error = '';
      try {
        const change = rotate ? rotateGroupInvitation : revokeGroupInvitation;
        const invitation = await change(groupId);
        if (this.groupId === groupId) this.$emit('updated', invitation);
      } catch {
        this.error = 'Could not update the invitation.';
      } finally {
        this.saving = false;
      }
    }
  }
};
</script>

<style scoped>
.invitation-actions {
  margin-top: 0.75rem;
}
.invitation-actions summary {
  cursor: pointer;
}
.invitation-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
