<template>
  <div class="groups-container">
    <!-- Header Section -->
    <section class="groups-header">
      <div class="container">
        <div class="header-content">
          <div class="title-section">
            <h1 class="page-title">{{ groupName || 'Group' }}</h1>
            <p class="page-subtitle">Invite code, teams, and members for your active group</p>
          </div>
        </div>
      </div>
    </section>

    <section class="groups-section">
      <div class="container">
        <!-- Loading State -->
        <div v-if="isLoading" class="loading-container">
          <div class="loading-spinner"></div>
          <p class="loading-text">Loading your group...</p>
        </div>

        <div v-else class="groups-layout">
          <div class="group-detail-container card-base card-large">
            <div v-if="loadFailed" class="empty-state">
              <div class="empty-content">
                <h3 class="empty-title">Couldn't load your group</h3>
                <p class="empty-description">Please try again in a moment.</p>
                <button class="btn-base btn-cancel btn-small" @click="loadActiveGroup">Retry</button>
              </div>
            </div>

            <!-- No group selected: either the player belongs to none yet, or
                 resolveActiveGroup() couldn't resolve one — either way, the
                 top-right group switcher is the one place to create/join. -->
            <div v-else-if="!activeGroupId" class="empty-state">
              <div class="empty-content">
                <h3 class="empty-title">No group selected</h3>
                <p class="empty-description">Use the group switcher in the top navigation to create one or join an existing one with its invite code.</p>
              </div>
            </div>

            <template v-else>
              <div class="group-actions">
                <!-- The code is never part of the group JSON, so it is
                     fetched only when asked for. -->
                <button v-if="!inviteCode" class="btn-base btn-cancel btn-small"
                  :disabled="loadingCode" @click="showInviteCode">
                  {{ loadingCode ? 'Loading...' : 'Show invite code' }}
                </button>

                <div v-else class="invite-code-box">
                  <code class="invite-code">{{ inviteCode }}</code>
                  <button class="btn-base btn-primary btn-small" @click="copyInviteCode">
                    {{ copied ? 'Copied!' : 'Copy' }}
                  </button>
                </div>

                <!-- Teams aren't part of the group JSON either, so they are
                     fetched on demand the first time this is opened. Label
                     and edit controls both follow the caller's own role: a
                     non-admin gets a read-only "Team" view, matching the
                     actual backend authorization (PATCH .../teams/:teamId is
                     admin-only) instead of an editable form that only fails
                     on save. -->
                <button class="btn-base btn-cancel btn-small" @click="toggleManageTeams">
                  {{ teamsExpanded
                    ? (isAdmin ? 'Hide teams' : 'Hide team')
                    : (isAdmin ? 'Manage teams' : 'Team') }}
                </button>

                <!-- The roster isn't part of the group JSON either — same
                     fetch-on-demand pattern as invite code/teams above. -->
                <button class="btn-base btn-cancel btn-small" @click="toggleManageMembers">
                  {{ membersExpanded ? 'Hide members' : 'Members' }}
                </button>
              </div>

              <p v-if="codeError" class="error-message group-error">{{ codeError }}</p>

              <div v-if="teamsExpanded" class="manage-teams-box">
                <p v-if="teamsLoading" class="loading-text">Loading teams...</p>
                <p v-else-if="teamsError" class="error-message">{{ teamsError }}</p>
                <div v-else-if="isAdmin" class="team-edit-list">
                  <div v-for="team in teams" :key="team.id" class="team-edit-row">
                    <input v-model="team.name" class="form-input team-name-input" type="text"
                      placeholder="Team name" :disabled="teamSaving[team.id]">
                    <TeamColourPicker v-model="team.colour" :disabled="teamSaving[team.id]" />
                    <button class="btn-base btn-primary btn-small"
                      :disabled="teamSaving[team.id] || !team.name.trim()"
                      @click="saveTeam(team)">
                      {{ teamSaving[team.id] ? 'Saving...' : 'Save' }}
                    </button>
                    <p v-if="teamSaveErrors[team.id]" class="error-message group-error">{{ teamSaveErrors[team.id] }}</p>
                    <p v-if="teamSaveSuccess[team.id]" class="success-message">{{ teamSaveSuccess[team.id] }}</p>
                  </div>
                </div>
                <!-- Non-admin: same data, read-only — no inputs that would
                     only fail on save with a 403. -->
                <ul v-else class="team-view-list">
                  <li v-for="team in teams" :key="team.id" class="team-view-row">
                    <span class="team-view-swatch" :style="{ backgroundColor: toHexColour(team.colour) }"></span>
                    <span class="team-view-name">{{ team.name }}</span>
                  </li>
                </ul>
              </div>

              <!-- Members: visible to any member, role-change/remove/invite
                   controls only rendered for the caller if they're an admin
                   of this group — and role-change/remove never on the
                   caller's own row, since the backend itself refuses
                   self-targeting (ErrCannotChangeOwnRole/ErrCannotRemoveSelf). -->
              <div v-if="membersExpanded" class="manage-members-box">
                <p v-if="membersLoading" class="loading-text">Loading members...</p>
                <p v-else-if="membersError" class="error-message">{{ membersError }}</p>
                <ul v-else class="member-list">
                  <li v-for="member in members" :key="member.id" class="member-row">
                    <div class="member-row-main">
                      <div class="member-identity">
                        <span class="member-name">{{ member.name }}</span>
                        <span v-if="member.role === 'admin'" class="admin-badge">Admin</span>
                        <span v-if="!member.email" class="ghost-badge">No account yet</span>
                      </div>

                      <div class="member-actions">
                        <!-- A "ghost" player (admin-created, no email) can be
                             invited to claim their own account — reuses the
                             password-reset flow (AuthService.InviteExistingPlayer)
                             behind POST /groups/:id/members/:playerId/invite. -->
                        <button v-if="isAdmin && !member.email && inviteFormFor !== member.id"
                          class="btn-base btn-cancel btn-small" @click="openInviteForm(member)">
                          Invite
                        </button>

                        <template v-if="isAdmin && member.id !== currentPlayerId">
                          <button class="btn-base btn-cancel btn-small" :disabled="memberActionLoading[member.id]"
                            @click="toggleMemberRole(member)">
                            {{ member.role === 'admin' ? 'Make member' : 'Make admin' }}
                          </button>
                          <button class="btn-base btn-danger btn-small" :disabled="memberActionLoading[member.id]"
                            @click="confirmRemoveMember(member)">
                            Remove
                          </button>
                        </template>
                      </div>
                    </div>

                    <div v-if="inviteFormFor === member.id" class="invite-form">
                      <input v-model="inviteEmailInput" class="form-input invite-email-input" type="email"
                        placeholder="their@email.com" :disabled="inviteLoading[member.id]">
                      <button class="btn-base btn-primary btn-small"
                        :disabled="inviteLoading[member.id] || !inviteEmailInput.trim()"
                        @click="sendInvite(member)">
                        {{ inviteLoading[member.id] ? 'Sending...' : 'Send invite' }}
                      </button>
                      <button class="btn-base btn-cancel btn-small" :disabled="inviteLoading[member.id]"
                        @click="closeInviteForm">
                        Cancel
                      </button>
                    </div>

                    <p v-if="inviteErrors[member.id]" class="error-message group-error">{{ inviteErrors[member.id] }}</p>
                    <p v-if="inviteSuccess[member.id]" class="success-message">{{ inviteSuccess[member.id] }}</p>
                    <p v-if="memberActionErrors[member.id]" class="error-message group-error">{{ memberActionErrors[member.id] }}</p>
                  </li>
                </ul>
              </div>
            </template>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
import {
  getInviteCode, getTeamsByGroup, updateTeam,
  getGroupMembers, updateMemberRole, removeMember, invitePlayer, getToken
} from '@/services/api';
import { resolveActiveGroup } from '@/services/activeGroup';
import TeamColourPicker from '@/components/TeamColourPicker.vue';

// The JWT's own player_id claim is the only place the caller's player id is
// available on this page (there's no "who am I" endpoint) — decoded locally,
// read-only, just to hide the role-change/remove controls on the caller's
// own member row (the backend already refuses self-targeting regardless).
function currentPlayerIdFromToken() {
  const token = getToken();
  if (!token) {
    return '';
  }
  try {
    const payload = token.split('.')[1];
    const decoded = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')));
    return decoded.player_id || '';
  } catch (error) {
    console.error('Error decoding token:', error);
    return '';
  }
}

// Same 10-entry keyword-to-hex palette getTeamColor() in
// MatchesPanel.vue/MatchDetails.vue know about — duplicated here rather than
// factored into a shared module, consistent with how it's already
// duplicated between those two files. Used only to translate a legacy
// keyword colour (from a team created before the colour picker existed)
// into the hex value the <input type="color"> needs.
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
  name: 'PlayerGroups',
  components: { TeamColourPicker },
  data() {
    return {
      isLoading: true,
      loadFailed: false,
      // The group this whole page is scoped to — the one currently active
      // in the top-right switcher, not every group the player belongs to.
      // Switching groups goes through a full page reload (GroupSwitcher), so
      // this only needs to be resolved once, on mount.
      activeGroupId: '',
      groupName: '',
      isAdmin: false,

      inviteCode: '',
      codeError: '',
      loadingCode: false,
      copied: false,

      teamsExpanded: false,
      teams: [],
      teamsLoading: false,
      teamsError: '',
      teamSaving: {},
      teamSaveErrors: {},
      teamSaveSuccess: {},

      membersExpanded: false,
      members: [],
      membersLoading: false,
      membersError: '',
      // Keyed by playerId rather than a single value: several rows could in
      // principle be mid-action at once (there's no reason to serialize them).
      memberActionLoading: {},
      memberActionErrors: {},
      currentPlayerId: '',

      // Ghost-player invite — at most one row's form open at a time.
      inviteFormFor: '',
      inviteEmailInput: '',
      inviteLoading: {},
      inviteErrors: {},
      inviteSuccess: {}
    };
  },
  async created() {
    this.currentPlayerId = currentPlayerIdFromToken();
    await this.loadActiveGroup();
  },
  methods: {
    // Thin wrapper so the module-level toHexColour() is reachable from the
    // template (used by the non-admin read-only team swatch below).
    toHexColour,
    async loadActiveGroup() {
      this.isLoading = true;
      this.loadFailed = false;
      try {
        const { groups, activeGroupId } = await resolveActiveGroup();
        this.activeGroupId = activeGroupId;
        const active = groups.find(g => g.id === activeGroupId);
        this.groupName = active?.name || '';
        this.isAdmin = active?.role === 'admin';
      } catch (error) {
        console.error('Error resolving the active group:', error);
        this.loadFailed = true;
      } finally {
        this.isLoading = false;
      }
    },
    async showInviteCode() {
      this.loadingCode = true;
      this.codeError = '';
      try {
        const data = await getInviteCode(this.activeGroupId);
        // A group created before invite codes existed has none: say so
        // instead of rendering an empty box that looks broken.
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
        // Clipboard access can be refused (insecure origin, denied
        // permission) — the code is on screen, so this is not fatal.
        console.error('Error copying invite code:', error);
        this.codeError = 'Copying failed — select the code and copy it manually.';
      }
    },
    async toggleManageTeams() {
      this.teamsExpanded = !this.teamsExpanded;
      if (!this.teamsExpanded || this.teams.length) {
        // Collapsing, or already fetched earlier in this session.
        return;
      }
      this.teamsLoading = true;
      this.teamsError = '';
      try {
        const teams = await getTeamsByGroup(this.activeGroupId);
        // Local editable copies, keyed by lower-case field names to match
        // what the input v-model bind to. The colour picker requires a hex
        // value, so a legacy keyword colour (from a team created before the
        // picker existed) is translated up front.
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
        const updated = await updateTeam(this.activeGroupId, team.id, name, team.colour);
        team.name = updated.name;
        team.colour = updated.colour;
        this.teamSaveSuccess[team.id] = 'Saved.';
      } catch (error) {
        this.teamSaveErrors[team.id] = this.backendMessage(error, 'Failed to update the team.');
      } finally {
        this.teamSaving[team.id] = false;
      }
    },
    async toggleManageMembers() {
      this.membersExpanded = !this.membersExpanded;
      if (!this.membersExpanded || this.members.length) {
        return;
      }
      await this.loadMembers();
    },
    async loadMembers() {
      this.membersLoading = true;
      this.membersError = '';
      try {
        const members = await getGroupMembers(this.activeGroupId);
        this.members = Array.isArray(members) ? members : [];
      } catch (error) {
        this.membersError = this.backendMessage(error, 'Failed to load members.');
      } finally {
        this.membersLoading = false;
      }
    },
    async toggleMemberRole(member) {
      const newRole = member.role === 'admin' ? 'member' : 'admin';
      this.memberActionLoading[member.id] = true;
      this.memberActionErrors[member.id] = '';
      try {
        await updateMemberRole(this.activeGroupId, member.id, newRole);
        // Re-fetch rather than patch in place: a role change can affect more
        // than just this row (e.g. this was the last admin, which the
        // backend would have refused anyway, but re-fetching keeps the list
        // authoritative either way).
        await this.loadMembers();
      } catch (error) {
        this.memberActionErrors[member.id] = this.backendMessage(error, 'Failed to update role.');
      } finally {
        this.memberActionLoading[member.id] = false;
      }
    },
    // Confirm-before-acting, same pattern as MatchDetails.vue's "Delete
    // Match" button — removal is a destructive action the admin could regret.
    async confirmRemoveMember(member) {
      const confirmed = window.confirm(
        `Remove ${member.name} from this group? Their match history will be kept.`
      );
      if (!confirmed) {
        return;
      }
      this.memberActionLoading[member.id] = true;
      this.memberActionErrors[member.id] = '';
      try {
        await removeMember(this.activeGroupId, member.id);
        await this.loadMembers();
      } catch (error) {
        this.memberActionErrors[member.id] = this.backendMessage(error, 'Failed to remove member.');
      } finally {
        this.memberActionLoading[member.id] = false;
      }
    },
    openInviteForm(member) {
      this.inviteFormFor = member.id;
      this.inviteEmailInput = '';
      this.inviteErrors[member.id] = '';
      this.inviteSuccess[member.id] = '';
    },
    closeInviteForm() {
      this.inviteFormFor = '';
      this.inviteEmailInput = '';
    },
    async sendInvite(member) {
      const email = this.inviteEmailInput.trim();
      if (!email) {
        return;
      }
      this.inviteLoading[member.id] = true;
      this.inviteErrors[member.id] = '';
      try {
        await invitePlayer(this.activeGroupId, member.id, email);
        this.inviteSuccess[member.id] = `Invite sent to ${email}. There is no email delivery configured yet in this environment — the link is logged on the server.`;
        this.inviteFormFor = '';
        this.inviteEmailInput = '';
        // Re-fetch so member.email reflects the claim and the "Invite"
        // action disappears for this row.
        await this.loadMembers();
      } catch (error) {
        this.inviteErrors[member.id] = this.backendMessage(error, 'Failed to send the invite.');
      } finally {
        this.inviteLoading[member.id] = false;
      }
    },
    backendMessage(error, fallback) {
      return error.response?.data?.error || fallback;
    }
  }
};
</script>

<style scoped>
/* Same container/header/card treatment as MatchesAndStandings.vue and Profile.vue. */
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

.group-actions {
  display: flex;
  flex-wrap: wrap;
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

.group-error {
  margin-top: 0.75rem;
}

/* Manage teams */
.manage-teams-box {
  margin-top: 1rem;
  padding: 0.875rem;
  border-radius: var(--border-radius);
  background-color: var(--bg-tertiary);
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

.team-view-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  list-style: none;
  padding: 0;
  margin: 0;
}

.team-view-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.team-view-swatch {
  width: 1.25rem;
  height: 1.25rem;
  border-radius: 50%;
  border: 1px solid var(--border-color);
  flex-shrink: 0;
}

.team-view-name {
  font-weight: 500;
  color: var(--text-primary);
}

.success-message {
  color: var(--primary-color);
  font-size: 0.875rem;
  margin-top: 0.5rem;
}

/* Members */
.manage-members-box {
  margin-top: 1rem;
  padding: 0.875rem;
  border-radius: var(--border-radius);
  background-color: var(--bg-tertiary);
}

.member-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.member-row {
  padding: 0.5rem 0.75rem;
  background-color: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
}

.member-row-main {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.member-identity {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.member-name {
  font-weight: 500;
  color: var(--text-primary);
}

.admin-badge {
  padding: 0.125rem 0.5rem;
  border-radius: 999px;
  background-color: var(--primary-color);
  color: white;
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.ghost-badge {
  padding: 0.125rem 0.5rem;
  border-radius: 999px;
  background-color: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.member-actions {
  display: flex;
  gap: 0.5rem;
}

.invite-form {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.invite-email-input {
  flex: 1;
  min-width: 12rem;
}

/* Responsive */
@media (max-width: 768px) {
  .groups-header {
    padding: 2rem 0;
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
