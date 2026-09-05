<template>
  <!-- Teleported to <body>: opened from Profile.vue's "Your groups" card,
       and any ancestor with backdrop-filter/filter would otherwise hijack
       this modal's position:fixed. Escaping #app also means it can't
       inherit .dark-mode — see src/services/theme.js's isDarkModeActive()
       for why isDarkModeSnapshot below reads the class directly rather than
       relying on inheritance or a prop. -->
  <Teleport to="body">
    <div class="modal-overlay" :class="{ 'dark-mode': isDarkModeSnapshot }" @click="close">
      <div class="modal-container group-settings-modal" @click.stop>
        <div class="modal-header">
          <h3>{{ groupName }} settings</h3>
          <button class="modal-close" @click="close" aria-label="Close">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div class="modal-body">
          <section class="settings-section">
            <h4 class="settings-section-title">Invite code</h4>
            <button v-if="!inviteCode" class="btn-base btn-cancel btn-small"
              :disabled="loadingCode" @click="showInviteCode">
              {{ loadingCode ? 'Loading...' : 'Show invite code' }}
            </button>
            <div v-else class="invite-code-box">
              <code class="invite-code">{{ inviteCode }}</code>
              <button class="btn-base btn-primary btn-small" @click="copyInviteCode">
                {{ copied ? 'Copied!' : 'Copy' }}
              </button>
              <!-- A plain <a>, not a click handler — it's genuinely just a
                   URL, same convention as MatchDetails.vue's own
                   .whatsapp-share-btn. -->
              <a v-if="whatsAppInviteUrl" :href="whatsAppInviteUrl" target="_blank" rel="noopener"
                class="btn-base btn-primary btn-small">
                Invite via WhatsApp
              </a>
            </div>
            <p v-if="codeError" class="error-message">{{ codeError }}</p>
          </section>

          <section class="settings-section">
            <h4 class="settings-section-title">Teams</h4>
            <p v-if="teamsLoading" class="loading-text">Loading teams...</p>
            <p v-else-if="teamsError" class="error-message">{{ teamsError }}</p>
            <div v-else class="team-edit-list">
              <div v-for="team in teams" :key="team.id" class="team-edit-row">
                <input v-model="team.name" class="form-input team-name-input" type="text"
                  placeholder="Team name" :disabled="teamSaving[team.id]">
                <TeamColourPicker v-model="team.colour" :disabled="teamSaving[team.id]" />
                <button class="btn-base btn-primary btn-small"
                  :disabled="teamSaving[team.id] || !team.name.trim()"
                  @click="saveTeam(team)">
                  {{ teamSaving[team.id] ? 'Saving...' : 'Save' }}
                </button>
                <p v-if="teamSaveErrors[team.id]" class="error-message">{{ teamSaveErrors[team.id] }}</p>
                <p v-if="teamSaveSuccess[team.id]" class="success-message">{{ teamSaveSuccess[team.id] }}</p>
              </div>
            </div>
          </section>
        </div>

        <div class="modal-footer">
          <button class="btn-base btn-cancel" @click="close">Close</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script>
import { getInviteCode, getTeamsByGroup, updateTeam } from '@/services/api';
import { isDarkModeActive } from '@/services/theme';
import { buildGroupInviteShareText, buildWhatsAppShareUrl } from '@/services/whatsappShare';
import TeamColourPicker from '@/components/TeamColourPicker.vue';

const LEGACY_TEAM_COLOUR_MAP = {
  red: '#ef4444', blue: '#3b82f6', green: '#10b981', yellow: '#f59e0b',
  purple: '#8b5cf6', orange: '#f97316', pink: '#ec4899', cyan: '#06b6d4',
  white: '#f8fafc', black: '#1f2937'
};

function toHexColour(colour) {
  if (colour && colour.startsWith('#')) {
    return colour;
  }
  return LEGACY_TEAM_COLOUR_MAP[(colour || '').toLowerCase()] || '#6b7280';
}

export default {
  name: 'GroupSettingsModal',
  components: { TeamColourPicker },
  props: {
    groupId: { type: String, required: true },
    groupName: { type: String, default: 'Group' }
  },
  emits: ['close'],
  data() {
    return {
      // Snapshot rather than reactive: this component is created fresh each
      // time it's opened (v-if), so it only needs the theme as it is right
      // now — toggling theme while the dialog happens to be open is not a
      // case worth a live DOM observer for.
      isDarkModeSnapshot: isDarkModeActive(),
      inviteCode: '',
      codeError: '',
      loadingCode: false,
      copied: false,

      teams: [],
      teamsLoading: true,
      teamsError: '',
      teamSaving: {},
      teamSaveErrors: {},
      teamSaveSuccess: {}
    };
  },
  async created() {
    await this.loadTeams();
  },
  computed: {
    // Self-service group creation/joining is disabled (see CLAUDE.md), so an
    // admin sharing this link — /signup?invite=CODE — is now the only way a
    // new player gets in; Signup.vue's created() reads the query param to
    // prefill the (now mandatory) invite-code field.
    whatsAppInviteUrl() {
      if (!this.inviteCode) {
        return '';
      }
      const inviteUrl = `${window.location.origin}/signup?invite=${this.inviteCode}`;
      const text = buildGroupInviteShareText({ inviteUrl, groupInviteCode: this.inviteCode });
      return buildWhatsAppShareUrl(text);
    }
  },
  methods: {
    async showInviteCode() {
      this.loadingCode = true;
      this.codeError = '';
      try {
        const data = await getInviteCode(this.groupId);
        this.inviteCode = data.invite_code || '';
        if (!this.inviteCode) {
          this.codeError = 'This group has no invite code yet.';
        }
      } catch (error) {
        this.codeError = this.backendMessage(error, 'Failed to load the invite code.');
      } finally {
        this.loadingCode = false;
      }
    },
    async copyInviteCode() {
      if (!this.inviteCode) {
        return;
      }
      try {
        await navigator.clipboard.writeText(this.inviteCode);
        this.copied = true;
        setTimeout(() => {
          this.copied = false;
        }, 2000);
      } catch (error) {
        console.error('Error copying invite code:', error);
        this.codeError = 'Copying failed — select the code and copy it manually.';
      }
    },
    async loadTeams() {
      this.teamsLoading = true;
      this.teamsError = '';
      try {
        const teams = await getTeamsByGroup(this.groupId);
        this.teams = (teams || []).map(team => ({
          id: team.id,
          name: team.name,
          colour: toHexColour(team.colour)
        }));
      } catch (error) {
        this.teamsError = this.backendMessage(error, 'Failed to load teams.');
      } finally {
        this.teamsLoading = false;
      }
    },
    async saveTeam(team) {
      const name = team.name.trim();
      if (!name) {
        return;
      }
      this.teamSaving[team.id] = true;
      this.teamSaveErrors[team.id] = '';
      this.teamSaveSuccess[team.id] = '';
      try {
        const updated = await updateTeam(this.groupId, team.id, name, team.colour);
        team.name = updated.name;
        team.colour = updated.colour;
        this.teamSaveSuccess[team.id] = 'Saved.';
      } catch (error) {
        this.teamSaveErrors[team.id] = this.backendMessage(error, 'Failed to update the team.');
      } finally {
        this.teamSaving[team.id] = false;
      }
    },
    close() {
      this.$emit('close');
    },
    backendMessage(error, fallback) {
      return error.response?.data?.error || fallback;
    }
  }
};
</script>

<style scoped>
.group-settings-modal {
  max-width: 32rem;
}

.settings-section + .settings-section {
  margin-top: 1.5rem;
}

.settings-section-title {
  margin: 0 0 0.75rem;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
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

.team-edit-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.team-edit-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.team-name-input {
  flex: 1;
  min-width: 10rem;
}

.success-message {
  color: var(--primary-color);
  font-size: 0.875rem;
  margin-top: 0.5rem;
  width: 100%;
}
</style>
