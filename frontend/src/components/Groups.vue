<template>
  <div class="groups-container">
    <!-- Header Section -->
    <section class="groups-header">
      <div class="container">
        <div class="header-content">
          <div class="title-section">
            <h1 class="page-title">My groups</h1>
            <p class="page-subtitle">See invite codes and manage teams for the groups you belong to</p>
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
                <p class="empty-description">Use the group switcher in the top navigation to create one or join an existing one with its invite code.</p>
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

                  <!-- Teams aren't part of the group JSON either, so they are
                       fetched on demand the first time this is opened for a
                       given group — same fetch-on-demand pattern as the
                       invite code above. There is no role-based hiding here:
                       a non-admin can open this and try to save, and the
                       backend's 403 is what actually stops them. -->
                  <button class="btn-base btn-cancel btn-small" @click="toggleManageTeams(group.id)">
                    {{ expandedTeamsFor === group.id ? 'Hide teams' : 'Manage teams' }}
                  </button>

                  <!-- The roster isn't part of the group JSON either — same
                       fetch-on-demand pattern as invite code/teams above. -->
                  <button class="btn-base btn-cancel btn-small" @click="toggleManageMembers(group.id)">
                    {{ expandedMembersFor === group.id ? 'Hide members' : 'Members' }}
                  </button>
                </div>

                <p v-if="codeErrors[group.id]" class="error-message group-error">{{ codeErrors[group.id] }}</p>

                <div v-if="expandedTeamsFor === group.id" class="manage-teams-box">
                  <p v-if="teamsLoadingFor === group.id" class="loading-text">Loading teams...</p>
                  <p v-else-if="teamsErrors[group.id]" class="error-message">{{ teamsErrors[group.id] }}</p>
                  <div v-else class="team-edit-list">
                    <div v-for="team in teamsByGroup[group.id]" :key="team.id" class="team-edit-row">
                      <input v-model="team.name" class="form-input team-name-input" type="text"
                        placeholder="Team name" :disabled="teamSaving[team.id]">
                      <TeamColourPicker v-model="team.colour" :disabled="teamSaving[team.id]" />
                      <button class="btn-base btn-primary btn-small"
                        :disabled="teamSaving[team.id] || !team.name.trim()"
                        @click="saveTeam(group.id, team)">
                        {{ teamSaving[team.id] ? 'Saving...' : 'Save' }}
                      </button>
                      <p v-if="teamSaveErrors[team.id]" class="error-message group-error">{{ teamSaveErrors[team.id] }}</p>
                      <p v-if="teamSaveSuccess[team.id]" class="success-message">{{ teamSaveSuccess[team.id] }}</p>
                    </div>
                  </div>
                </div>

                <!-- Members: visible to any member, role-change/remove
                     controls only rendered for the caller if they're an
                     admin of this group (this.groups already carries the
                     caller's own role per group) — and never on the caller's
                     own row, since the backend itself refuses self-targeting
                     (ErrCannotChangeOwnRole/ErrCannotRemoveSelf). -->
                <div v-if="expandedMembersFor === group.id" class="manage-members-box">
                  <p v-if="membersLoadingFor === group.id" class="loading-text">Loading members...</p>
                  <p v-else-if="membersErrors[group.id]" class="error-message">{{ membersErrors[group.id] }}</p>
                  <ul v-else class="member-list">
                    <li v-for="member in membersByGroup[group.id]" :key="member.id" class="member-row">
                      <div class="member-identity">
                        <span class="member-name">{{ member.name }}</span>
                        <span v-if="member.role === 'admin'" class="admin-badge">Admin</span>
                      </div>

                      <div v-if="isGroupAdmin(group.id) && member.id !== currentPlayerId" class="member-actions">
                        <button class="btn-base btn-cancel btn-small" :disabled="memberActionLoading[member.id]"
                          @click="toggleMemberRole(group.id, member)">
                          {{ member.role === 'admin' ? 'Make member' : 'Make admin' }}
                        </button>
                        <button class="btn-base btn-danger btn-small" :disabled="memberActionLoading[member.id]"
                          @click="confirmRemoveMember(group.id, member)">
                          Remove
                        </button>
                      </div>

                      <p v-if="memberActionErrors[member.id]" class="error-message group-error">{{ memberActionErrors[member.id] }}</p>
                    </li>
                  </ul>
                </div>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
import {
  getMyGroups, getInviteCode, getTeamsByGroup, updateTeam,
  getGroupMembers, updateMemberRole, removeMember, getToken
} from '@/services/api';
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
      groups: [],
      // groupId -> code, filled in on demand by showInviteCode.
      inviteCodes: {},
      codeErrors: {},
      loadingCodeFor: '',
      copiedGroupId: '',
      isLoading: true,
      loadFailed: false,
      // "Manage teams" — id of the one group currently expanded, plus its
      // teams fetched on demand (groupId -> array of {id, name, colour}).
      expandedTeamsFor: '',
      teamsByGroup: {},
      teamsLoadingFor: '',
      teamsErrors: {},
      teamSaving: {},
      teamSaveErrors: {},
      teamSaveSuccess: {},
      // Members — id of the one group currently expanded, plus its members
      // fetched on demand (groupId -> array of PlayerWithRole).
      expandedMembersFor: '',
      membersByGroup: {},
      membersLoadingFor: '',
      membersErrors: {},
      // Keyed by playerId rather than groupId: a role-change/remove action
      // targets one member row, and two different groups never share a
      // playerId collision that would matter here.
      memberActionLoading: {},
      memberActionErrors: {},
      currentPlayerId: ''
    };
  },
  async created() {
    this.currentPlayerId = currentPlayerIdFromToken();
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
    async toggleManageTeams(groupId) {
      if (this.expandedTeamsFor === groupId) {
        this.expandedTeamsFor = '';
        return;
      }
      this.expandedTeamsFor = groupId;
      if (this.teamsByGroup[groupId]) {
        // Already fetched earlier in this session — no need to refetch.
        return;
      }
      this.teamsLoadingFor = groupId;
      this.teamsErrors[groupId] = '';
      try {
        const teams = await getTeamsByGroup(groupId);
        // Local editable copies, keyed by lower-case field names to match
        // what the input v-model bind to. The colour picker requires a hex
        // value, so a legacy keyword colour (from a team created before the
        // picker existed) is translated up front.
        this.teamsByGroup[groupId] = (teams || []).map(team => ({
          id: team.id,
          name: team.name,
          colour: toHexColour(team.colour)
        }));
      } catch (error) {
        this.teamsErrors[groupId] = this.backendMessage(error, 'Failed to load teams.');
      } finally {
        this.teamsLoadingFor = '';
      }
    },
    async saveTeam(groupId, team) {
      const name = team.name.trim();
      if (!name) {
        return;
      }
      this.teamSaving[team.id] = true;
      this.teamSaveErrors[team.id] = '';
      this.teamSaveSuccess[team.id] = '';
      try {
        const updated = await updateTeam(groupId, team.id, name, team.colour);
        team.name = updated.name;
        team.colour = updated.colour;
        this.teamSaveSuccess[team.id] = 'Saved.';
      } catch (error) {
        // No role-based UI hiding anywhere in this app — a non-admin can get
        // this far and only finds out from the backend's own 403 message.
        this.teamSaveErrors[team.id] = this.backendMessage(error, 'Failed to update the team.');
      } finally {
        this.teamSaving[team.id] = false;
      }
    },
    async toggleManageMembers(groupId) {
      if (this.expandedMembersFor === groupId) {
        this.expandedMembersFor = '';
        return;
      }
      this.expandedMembersFor = groupId;
      if (this.membersByGroup[groupId]) {
        // Already fetched earlier in this session — no need to refetch.
        return;
      }
      await this.loadMembers(groupId);
    },
    async loadMembers(groupId) {
      this.membersLoadingFor = groupId;
      this.membersErrors[groupId] = '';
      try {
        const members = await getGroupMembers(groupId);
        this.membersByGroup[groupId] = Array.isArray(members) ? members : [];
      } catch (error) {
        this.membersErrors[groupId] = this.backendMessage(error, 'Failed to load members.');
      } finally {
        this.membersLoadingFor = '';
      }
    },
    // Whether the *caller* is an admin of groupId — this.groups already
    // carries the caller's own role per group (GetGroupsWithRoleByPlayerID),
    // same lookup MatchDetails.vue does for its own isAdmin gating.
    isGroupAdmin(groupId) {
      const group = this.groups.find(g => g.id === groupId);
      return group?.role === 'admin';
    },
    async toggleMemberRole(groupId, member) {
      const newRole = member.role === 'admin' ? 'member' : 'admin';
      this.memberActionLoading[member.id] = true;
      this.memberActionErrors[member.id] = '';
      try {
        await updateMemberRole(groupId, member.id, newRole);
        // Re-fetch rather than patch in place: a role change can affect more
        // than just this row (e.g. this was the last admin, which the
        // backend would have refused anyway, but re-fetching keeps the list
        // authoritative either way).
        await this.loadMembers(groupId);
      } catch (error) {
        this.memberActionErrors[member.id] = this.backendMessage(error, 'Failed to update role.');
      } finally {
        this.memberActionLoading[member.id] = false;
      }
    },
    // Confirm-before-acting, same pattern as MatchDetails.vue's "Delete
    // Match" button — removal is a destructive action the admin could regret.
    async confirmRemoveMember(groupId, member) {
      const confirmed = window.confirm(
        `Remove ${member.name} from this group? Their match history will be kept.`
      );
      if (!confirmed) {
        return;
      }
      this.memberActionLoading[member.id] = true;
      this.memberActionErrors[member.id] = '';
      try {
        await removeMember(groupId, member.id);
        await this.loadMembers(groupId);
      } catch (error) {
        this.memberActionErrors[member.id] = this.backendMessage(error, 'Failed to remove member.');
      } finally {
        this.memberActionLoading[member.id] = false;
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

.section-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 0.25rem;
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

.group-error {
  flex-basis: 100%;
  margin-top: 0;
}

/* Manage teams */
.manage-teams-box {
  flex-basis: 100%;
  margin-top: 0.5rem;
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

.success-message {
  color: var(--primary-color);
  font-size: 0.875rem;
  margin-top: 0.5rem;
}

/* Members */
.manage-members-box {
  flex-basis: 100%;
  margin-top: 0.5rem;
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
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background-color: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
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

.member-actions {
  display: flex;
  gap: 0.5rem;
}

/* Responsive */
@media (max-width: 768px) {
  .groups-header {
    padding: 2rem 0;
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
