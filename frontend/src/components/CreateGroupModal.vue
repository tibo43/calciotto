<template>
  <!-- Teleported to <body>: opened from Profile.vue's "Your groups" card,
       and any ancestor with backdrop-filter/filter would otherwise hijack
       this modal's position:fixed (see GroupSettingsModal.vue for the fuller
       explanation, and for why isDarkModeSnapshot below — not a prop — is
       what reapplies .dark-mode on this now-detached root). -->
  <Teleport to="body">
    <div class="modal-overlay" :class="{ 'dark-mode': isDarkModeSnapshot }" @click="close">
      <div class="modal-container create-group-modal" @click.stop>
        <div class="modal-header">
          <h3>Create a group</h3>
          <button class="modal-close" @click="close" aria-label="Close" :disabled="isCreating">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <form @submit.prevent="submitCreate">
          <div class="modal-body">
            <p class="modal-hint">You become its first member, and get an invite code to share.</p>

            <div class="form-group">
              <label for="new-group-name">Group name</label>
              <input
                id="new-group-name"
                v-model="name"
                class="form-input"
                type="text"
                placeholder="e.g. Tuesday night calciotto"
                :disabled="isCreating"
              >
            </div>

            <div class="form-group team-spec-group" v-for="(team, index) in teams" :key="index">
              <label :for="'create-group-team-name-' + index">Team {{ index + 1 }} name</label>
              <div class="team-spec-inputs">
                <input
                  :id="'create-group-team-name-' + index"
                  v-model="team.name"
                  class="form-input team-name-input"
                  type="text"
                  placeholder="e.g. Les Rouges"
                  :disabled="isCreating"
                >
                <TeamColourPicker v-model="team.colour" :disabled="isCreating" />
              </div>
            </div>

            <p v-if="createError" class="error-message">{{ createError }}</p>
          </div>

          <div class="modal-footer">
            <button class="btn-base btn-cancel" type="button" @click="close" :disabled="isCreating">
              Cancel
            </button>
            <button class="btn-base btn-primary" type="submit" :disabled="isCreating || !canSubmit">
              {{ isCreating ? 'Creating...' : 'Create group' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>

<script>
import { createGroup } from '@/services/api';
import { setActiveGroupId } from '@/services/activeGroup';
import { isDarkModeActive } from '@/services/theme';
import TeamColourPicker from '@/components/TeamColourPicker.vue';

export default {
  name: 'CreateGroupModal',
  components: { TeamColourPicker },
  emits: ['close'],
  data() {
    return {
      // Snapshot rather than reactive: this component is created fresh each
      // time it's opened (v-if), so it only needs the theme as it is right
      // now.
      isDarkModeSnapshot: isDarkModeActive(),
      name: '',
      teams: [
        { name: '', colour: '#1f2937' },
        { name: '', colour: '#f8fafc' }
      ],
      isCreating: false,
      createError: ''
    };
  },
  computed: {
    canSubmit() {
      return Boolean(
        this.name.trim() &&
        this.teams.every((team) => team.name.trim() && team.colour)
      );
    }
  },
  methods: {
    async submitCreate() {
      const name = this.name.trim();
      if (!name || !this.canSubmit) {
        return;
      }
      const teams = this.teams.map((team) => ({ name: team.name.trim(), colour: team.colour }));
      this.isCreating = true;
      this.createError = '';
      try {
        const group = await createGroup(name, teams);
        // Same reasoning as joining: switch straight into the group just
        // created rather than leaving the player on whatever was active.
        setActiveGroupId(group.id);
        window.location.reload();
      } catch (error) {
        this.createError = this.backendMessage(error, 'Failed to create the group.');
        this.isCreating = false;
      }
    },
    close() {
      if (this.isCreating) {
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
.create-group-modal {
  max-width: 32rem;
}

.modal-hint {
  color: var(--text-secondary);
  font-size: 0.875rem;
  margin: 0 0 1.25rem;
}

.team-spec-group {
  margin-bottom: 1rem;
}

.team-spec-inputs {
  display: flex;
  gap: 0.5rem;
}

.team-spec-inputs .team-name-input {
  flex: 1;
}
</style>
