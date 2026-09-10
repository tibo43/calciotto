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
              <button class="btn-base btn-primary btn-small copy-invite-btn" @click="copyInviteCode">
                {{ copied ? 'Copied!' : 'Copy' }}
              </button>
              <!-- A plain <a>, not a click handler — it's genuinely just a
                   URL, same convention as MatchDetails.vue's own
                   .whatsapp-share-btn, whose look (icon + label, collapsing
                   to icon-only on mobile) this now matches exactly rather
                   than being a plain solid-green text button — the same
                   WhatsApp action should look the same wherever it appears.
                   The icon/label markup and the CSS a couple of screens down
                   are duplicated from there rather than shared, the same
                   "small pieces are duplicated across components" convention
                   getTeamColor()/formatPlayerNameForDisplay() already follow
                   in this codebase — there's no shared component library to
                   put a single copy in. -->
              <a v-if="whatsAppInviteUrl" :href="whatsAppInviteUrl" target="_blank" rel="noopener"
                class="btn-base btn-cancel btn-small whatsapp-share-btn" aria-label="Invite via WhatsApp">
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path
                    d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.472-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.148-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z" />
                  <path
                    d="M12.004 2c-5.514 0-9.997 4.483-9.997 9.997 0 1.762.464 3.485 1.346 5.002L2 22l5.14-1.334a9.958 9.958 0 0 0 4.862 1.237h.004c5.514 0 9.997-4.483 9.997-9.997 0-2.67-1.04-5.182-2.928-7.07A9.933 9.933 0 0 0 12.004 2zm0 18.183h-.003a8.19 8.19 0 0 1-4.17-1.142l-.299-.178-3.05.793.814-2.973-.195-.306a8.18 8.18 0 0 1-1.256-4.38c0-4.523 3.68-8.203 8.203-8.203 2.19 0 4.25.853 5.799 2.404a8.146 8.146 0 0 1 2.403 5.803c0 4.523-3.681 8.202-8.246 8.202z" />
                </svg>
                <span class="whatsapp-share-label">Invite via WhatsApp</span>
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
                <button class="btn-base btn-primary btn-small save-team-btn"
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
  flex-wrap: wrap;
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

/* Icon-only on a narrow screen, same as MatchDetails.vue's two
   .whatsapp-share-btn instances — kept in sync by hand (see the template
   comment above) rather than shared, since there's no component library
   here to put a single copy in. */
@media (max-width: 768px) {
  /* Undoes global-styles.css's .btn-base { width: 100% } (aimed at modal
     footers, not this row) — without it Copy would claim the whole row
     width and push Invite via WhatsApp onto a line of its own, even though
     invite-code-box lays them out side by side. height: 3rem matches
     whatsapp-share-btn's own explicit height so the two sit at the same
     level instead of Copy floating taller/shorter next to it. */
  .copy-invite-btn {
    width: auto;
    height: 3rem;
  }

  .whatsapp-share-label {
    display: none;
  }

  .whatsapp-share-btn {
    width: 3rem;
    height: 3rem;
    padding: 0;
    justify-content: center;
  }

  /* Same global .btn-base { width: 100% } problem as Copy above: without
     this, Save claims the whole row and drops onto its own line below the
     team name input and colour swatch instead of sitting on the same line
     as them. The input's own min-width shrinks a little too (10rem left no
     room for Save next to it at a phone width), and the row's gap tightens
     to match — the swatch is a fixed 40px regardless. */
  .team-edit-row {
    gap: 0.4rem;
  }

  .team-name-input {
    min-width: 7rem;
  }

  .save-team-btn {
    width: auto;
    flex-shrink: 0;
  }

  .whatsapp-share-btn svg {
    width: 24px;
    height: 24px;
  }
}
</style>
