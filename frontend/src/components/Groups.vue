<template>
  <div class="groups-container">
    <!-- Header Section -->
    <section class="groups-header">
      <div class="container">
        <div class="header-content">
          <div class="title-section">
            <h1 class="page-title">My groups</h1>
            <p class="page-subtitle">Create a group, or join one with an invite code</p>
          </div>
        </div>
      </div>
    </section>

    <section class="groups-section">
      <div class="container">
        <!-- Loading State -->
        <div v-if="isLoading" class="loading-container">
          <div class="loading-spinner"></div>
          <p class="loading-text">Loading your groups...</p>
        </div>

        <div v-else class="groups-layout">
          <!-- The groups the player belongs to -->
          <div class="groups-list-container card-base card-large">
            <h2 class="section-title">Groups you belong to</h2>

            <div v-if="loadFailed" class="empty-state">
              <div class="empty-content">
                <h3 class="empty-title">Couldn't load your groups</h3>
                <p class="empty-description">Please try again in a moment.</p>
                <button class="btn-base btn-cancel btn-small" @click="loadGroups">Retry</button>
              </div>
            </div>

            <div v-else-if="groups.length === 0" class="empty-state">
              <div class="empty-content">
                <h3 class="empty-title">No group yet</h3>
                <p class="empty-description">Create one below, or join an existing one with its invite code.</p>
              </div>
            </div>

            <ul v-else class="group-list">
              <li v-for="group in groups" :key="group.id" class="group-row">
                <div class="group-identity">
                  <div class="player-avatar-small">{{ getGroupInitials(group.name) }}</div>
                  <span class="group-name">{{ group.name }}</span>
                </div>

                <div class="group-actions">
                  <!-- The code is never part of the group JSON, so it is
                       fetched one group at a time, only when asked for. -->
                  <button v-if="!inviteCodes[group.id]" class="btn-base btn-cancel btn-small"
                    :disabled="loadingCodeFor === group.id" @click="showInviteCode(group.id)">
                    {{ loadingCodeFor === group.id ? 'Loading...' : 'Show invite code' }}
                  </button>

                  <div v-else class="invite-code-box">
                    <code class="invite-code">{{ inviteCodes[group.id] }}</code>
                    <button class="btn-base btn-primary btn-small" @click="copyInviteCode(group.id)">
                      {{ copiedGroupId === group.id ? 'Copied!' : 'Copy' }}
                    </button>
                  </div>
                </div>

                <p v-if="codeErrors[group.id]" class="error-message group-error">{{ codeErrors[group.id] }}</p>
              </li>
            </ul>
          </div>

          <!-- Create / join -->
          <div class="forms-grid">
            <div class="form-card card-base card-large">
              <h2 class="section-title">Create a group</h2>
              <p class="section-hint">You become its first member, and get an invite code to share.</p>
              <form @submit.prevent="submitCreate">
                <div class="form-group">
                  <label for="group-name">Group name</label>
                  <input id="group-name" v-model="newGroupName" class="form-input" type="text"
                    placeholder="e.g. Tuesday night calciotto" :disabled="isCreating">
                </div>
                <button class="btn-base btn-primary" type="submit" :disabled="isCreating || !newGroupName.trim()">
                  {{ isCreating ? 'Creating...' : 'Create group' }}
                </button>
                <p v-if="createError" class="error-message">{{ createError }}</p>
                <p v-if="createSuccess" class="success-message">{{ createSuccess }}</p>
              </form>
            </div>

            <div class="form-card card-base card-large">
              <h2 class="section-title">Join a group</h2>
              <p class="section-hint">Ask a member for the group's invite code.</p>
              <form @submit.prevent="submitJoin">
                <div class="form-group">
                  <label for="invite-code-input">Invite code</label>
                  <input id="invite-code-input" v-model="inviteCodeInput" class="form-input invite-code-input"
                    type="text" placeholder="ABCD2345" autocapitalize="characters" :disabled="isJoining">
                </div>
                <button class="btn-base btn-primary" type="submit" :disabled="isJoining || !inviteCodeInput.trim()">
                  {{ isJoining ? 'Joining...' : 'Join group' }}
                </button>
                <p v-if="joinError" class="error-message">{{ joinError }}</p>
                <p v-if="joinSuccess" class="success-message">{{ joinSuccess }}</p>
              </form>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
import { getMyGroups, createGroup, joinGroup, getInviteCode } from '@/services/api';

export default {
  name: 'PlayerGroups',
  data() {
    return {
      groups: [],
      // groupId -> code, filled in on demand by showInviteCode.
      inviteCodes: {},
      codeErrors: {},
      loadingCodeFor: '',
      copiedGroupId: '',
      isLoading: true,
      loadFailed: false,
      newGroupName: '',
      isCreating: false,
      createError: '',
      createSuccess: '',
      inviteCodeInput: '',
      isJoining: false,
      joinError: '',
      joinSuccess: ''
    };
  },
  async created() {
    await this.loadGroups();
  },
  methods: {
    async loadGroups() {
      this.isLoading = true;
      this.loadFailed = false;
      try {
        const groups = await getMyGroups();
        this.groups = Array.isArray(groups) ? groups : [];
      } catch (error) {
        console.error('Error fetching my groups:', error);
        this.loadFailed = true;
        this.groups = [];
      } finally {
        this.isLoading = false;
      }
    },
    async showInviteCode(groupId) {
      this.loadingCodeFor = groupId;
      this.codeErrors[groupId] = '';
      try {
        const data = await getInviteCode(groupId);
        // A group created before invite codes existed has none: say so
        // instead of rendering an empty box that looks broken.
        this.inviteCodes[groupId] = data.invite_code || '';
        if (!data.invite_code) {
          this.codeErrors[groupId] = 'This group has no invite code yet.';
        }
      } catch (error) {
        this.codeErrors[groupId] = this.backendMessage(error, 'Failed to load the invite code.');
      } finally {
        this.loadingCodeFor = '';
      }
    },
    async copyInviteCode(groupId) {
      const code = this.inviteCodes[groupId];
      if (!code) {
        return;
      }
      try {
        await navigator.clipboard.writeText(code);
        this.copiedGroupId = groupId;
        setTimeout(() => {
          if (this.copiedGroupId === groupId) {
            this.copiedGroupId = '';
          }
        }, 2000);
      } catch (error) {
        // Clipboard access can be refused (insecure origin, denied
        // permission) — the code is on screen, so this is not fatal.
        console.error('Error copying invite code:', error);
        this.codeErrors[groupId] = 'Copying failed — select the code and copy it manually.';
      }
    },
    async submitCreate() {
      const name = this.newGroupName.trim();
      if (!name) {
        return;
      }
      this.isCreating = true;
      this.createError = '';
      this.createSuccess = '';
      try {
        const group = await createGroup(name);
        // Append rather than reload: the response is the created group, and
        // the caller is already its first member server-side.
        this.groups.push(group);
        this.newGroupName = '';
        this.createSuccess = `Group "${group.name}" created.`;
      } catch (error) {
        this.createError = this.backendMessage(error, 'Failed to create the group.');
      } finally {
        this.isCreating = false;
      }
    },
    async submitJoin() {
      const code = this.inviteCodeInput.trim();
      if (!code) {
        return;
      }
      this.isJoining = true;
      this.joinError = '';
      this.joinSuccess = '';
      try {
        const group = await joinGroup(code);
        if (!this.groups.some(existing => existing.id === group.id)) {
          this.groups.push(group);
        }
        this.inviteCodeInput = '';
        this.joinSuccess = `You joined "${group.name}".`;
      } catch (error) {
        // The backend already tells the two failure modes apart — 404 for an
        // unknown code, 400 for "you are already a member" — so show its own
        // message rather than collapsing both into one generic error.
        this.joinError = this.backendMessage(error, 'Failed to join the group.');
      } finally {
        this.isJoining = false;
      }
    },
    backendMessage(error, fallback) {
      return error.response?.data?.error || fallback;
    },
    getGroupInitials(name) {
      return (name || '')
        .split(' ')
        .map(word => word.charAt(0).toUpperCase())
        .join('')
        .slice(0, 2);
    }
  }
};
</script>

<style scoped>
/* Same container/header/card treatment as Standings.vue and Profile.vue. */
.groups-container {
  background-color: var(--bg-secondary);
}

.groups-header {
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--secondary-color) 100%);
  color: white;
  padding: 1rem 0;
}

.groups-section {
  padding: 2rem 0;
}

.groups-layout {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.section-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 0.25rem;
}

.section-hint {
  color: var(--text-secondary);
  font-size: 0.875rem;
  margin: 0 0 1.25rem;
}

/* Group list */
.group-list {
  list-style: none;
  margin: 1.25rem 0 0;
  padding: 0;
}

.group-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.875rem 0;
  border-bottom: 1px solid var(--border-color);
}

.group-row:last-child {
  border-bottom: none;
}

.group-identity {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.group-name {
  font-weight: 600;
  color: var(--text-primary);
}

.group-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.invite-code-box {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.invite-code {
  padding: 0.5rem 0.75rem;
  border-radius: var(--border-radius);
  background-color: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-weight: 700;
  letter-spacing: 0.12em;
}

.invite-code-input {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.group-error {
  flex-basis: 100%;
  margin-top: 0;
}

/* Create / join forms */
.forms-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1.5rem;
  align-items: start;
}

.form-card .form-group:last-of-type {
  margin-bottom: 1rem;
}

.success-message {
  color: var(--primary-color);
  font-size: 0.875rem;
  margin-top: 0.5rem;
}

/* Responsive */
@media (max-width: 768px) {
  .groups-header {
    padding: 2rem 0;
  }

  .forms-grid {
    grid-template-columns: 1fr;
  }

  .group-row {
    align-items: flex-start;
    flex-direction: column;
  }

  .group-actions,
  .invite-code-box {
    width: 100%;
  }

  .invite-code {
    flex: 1;
    text-align: center;
  }
}
</style>
