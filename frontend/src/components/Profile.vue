<template>
  <div class="profile-container">
    <!-- Header Section -->
    <section class="profile-header">
      <div class="container">
        <div class="header-content">
          <div class="title-section">
            <h1 class="page-title">My profile</h1>
            <p class="page-subtitle">Your record across every group you belong to</p>
          </div>
        </div>
      </div>
    </section>

    <section class="profile-section">
      <div class="container">
        <!-- Loading State -->
        <div v-if="isLoading" class="loading-container">
          <div class="loading-spinner"></div>
          <p class="loading-text">Loading your stats...</p>
        </div>

        <div v-else-if="loadFailed" class="standings-table-container card-base card-large">
          <div class="empty-state">
            <div class="empty-content">
              <h3 class="empty-title">Couldn't load your profile</h3>
              <p class="empty-description">Please try again in a moment.</p>
            </div>
          </div>
        </div>

        <div v-else class="profile-layout">
          <!-- Overall stats, all groups combined -->
          <div class="overall-card card-base card-large">
            <div class="overall-identity">
              <div class="player-avatar-small">{{ getPlayerInitials(overall.Name) }}</div>
              <div>
                <h2 class="overall-name">{{ formatPlayerNameForDisplay(overall.Name) }}</h2>
                <p class="overall-scope">{{ scopeLabel }}</p>
              </div>
            </div>
            <div class="overall-stats">
              <div v-for="stat in overallStats" :key="stat.label"
                :class="['overall-stat', { highlight: stat.highlight }]">
                <span class="overall-stat-value">{{ stat.value }}</span>
                <span class="overall-stat-label">{{ stat.label }}</span>
              </div>
            </div>
          </div>

          <!-- Groups: a card per group you belong to, with your own stats in
               it — replaces the old standalone Groups page. Click a card to
               see (and, if you're an admin there, manage) its roster below. -->
          <div class="groups-card card-base card-large">
            <h2 class="section-title">Your groups</h2>

            <div v-if="perGroup.length === 0" class="empty-state">
              <div class="empty-content">
                <h3 class="empty-title">No group yet</h3>
                <p class="empty-description">Use the group switcher in the top navigation to create one or join an existing one with its invite code.</p>
              </div>
            </div>

            <template v-else>
              <div class="groups-bar-container">
                <div class="groups-bar hide-scrollbar">
                  <div v-for="row in perGroup" :key="row.GroupID" class="group-card-horizontal"
                    :class="{ active: selectedGroupId === row.GroupID }" @click="selectGroup(row.GroupID)">
                    <div class="group-card-header">
                      <span class="group-card-name">{{ row.GroupName }}</span>
                      <span v-if="isGroupAdmin(row.GroupID)" class="admin-badge">Admin</span>
                    </div>
                    <div class="group-card-stats">
                      <div class="group-card-stat">
                        <span class="group-card-stat-value">{{ row.Points }}</span>
                        <span class="group-card-stat-label">Pts</span>
                      </div>
                      <div class="group-card-stat">
                        <span class="group-card-stat-value">{{ row.Played }}</span>
                        <span class="group-card-stat-label">Played</span>
                      </div>
                      <div class="group-card-stat">
                        <span class="group-card-stat-value">{{ row.GoalsFor }}</span>
                        <span class="group-card-stat-label">Goals</span>
                      </div>
                    </div>
                    <!-- Invite code + team management: a separate dialog
                         rather than folded into this card or the roster
                         panel below, since it's an occasional admin action
                         and not part of browsing stats/roster. -->
                    <button v-if="isGroupAdmin(row.GroupID)" class="group-settings-btn"
                      @click.stop="openSettings(row)" aria-label="Group settings">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="3" />
                        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
                      </svg>
                    </button>
                  </div>
                </div>
              </div>

              <!-- Roster panel for the selected group. Open to any member
                   (viewing is never gated) — only an admin of *this*
                   specific group sees the action buttons. -->
              <transition name="roster-panel">
                <div v-if="selectedGroupId" class="roster-panel-container card-base">
                  <h3 class="roster-title">{{ selectedGroupName }} — members</h3>
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
                          <button v-if="isGroupAdmin(selectedGroupId) && !member.email && inviteFormFor !== member.id"
                            class="btn-base btn-cancel btn-small" @click="openInviteForm(member)">
                            Invite
                          </button>

                          <template v-if="isGroupAdmin(selectedGroupId) && member.id !== currentPlayerId">
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

                      <p v-if="inviteErrors[member.id]" class="error-message">{{ inviteErrors[member.id] }}</p>
                      <p v-if="inviteSuccess[member.id]" class="success-message">{{ inviteSuccess[member.id] }}</p>
                      <p v-if="memberActionErrors[member.id]" class="error-message">{{ memberActionErrors[member.id] }}</p>
                    </li>
                  </ul>
                </div>
              </transition>
            </template>
          </div>
        </div>
      </div>
    </section>

    <GroupSettingsModal v-if="settingsForGroup" :group-id="settingsForGroup.id" :group-name="settingsForGroup.name"
      @close="settingsForGroup = null" />
  </div>
</template>

<script>
import {
  getPlayerProfile, getMyGroups, getGroupMembers,
  updateMemberRole, removeMember, invitePlayer, getToken
} from '@/services/api';
import GroupSettingsModal from '@/components/GroupSettingsModal.vue';

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

export default {
  name: 'PlayerProfile',
  components: { GroupSettingsModal },
  data() {
    return {
      overall: { Name: '', Played: 0, Won: 0, Drawn: 0, Lost: 0, GoalsFor: 0, Points: 0 },
      perGroup: [],
      isLoading: true,
      loadFailed: false,

      // GroupID -> role ("admin"/"member"), resolved separately from the
      // profile call (GET /players/me/stats has no reason to know about
      // roles) — same GetGroupsWithRoleByPlayerID data every other
      // role-gated view in this app already reads.
      groupRoles: {},

      // The one group whose roster is currently expanded, plus a
      // fetch-on-demand cache per group (switching between two groups you
      // already opened this session re-uses the cached roster).
      selectedGroupId: '',
      membersByGroup: {},
      membersLoading: false,
      membersError: '',
      memberActionLoading: {},
      memberActionErrors: {},
      currentPlayerId: '',

      // Ghost-player invite — at most one row's form open at a time.
      inviteFormFor: '',
      inviteEmailInput: '',
      inviteLoading: {},
      inviteErrors: {},
      inviteSuccess: {},

      // Non-null while the invite-code/teams dialog for one group is open.
      settingsForGroup: null
    };
  },
  computed: {
    overallStats() {
      return [
        { label: 'Played', value: this.overall.Played },
        { label: 'Won', value: this.overall.Won },
        { label: 'Drawn', value: this.overall.Drawn },
        { label: 'Lost', value: this.overall.Lost },
        { label: 'Goals', value: this.overall.GoalsFor },
        { label: 'Points', value: this.overall.Points, highlight: true }
      ];
    },
    scopeLabel() {
      // With a single group the "all groups" wording would be misleading, so
      // say how many groups these totals actually cover.
      const count = this.perGroup.length;
      if (count === 0) {
        return 'All seasons — no group yet';
      }
      return count === 1
        ? `All seasons — ${this.perGroup[0].GroupName}`
        : `All seasons — ${count} groups combined`;
    },
    members() {
      return this.membersByGroup[this.selectedGroupId] || [];
    },
    selectedGroupName() {
      return this.perGroup.find(row => row.GroupID === this.selectedGroupId)?.GroupName || '';
    }
  },
  async created() {
    this.currentPlayerId = currentPlayerIdFromToken();
    await Promise.all([this.loadProfile(), this.loadGroupRoles()]);
  },
  methods: {
    async loadProfile() {
      this.isLoading = true;
      this.loadFailed = false;
      try {
        // No season passed: the profile shows the all-time record for now.
        const profile = await getPlayerProfile();
        this.overall = profile.Overall || this.overall;
        this.perGroup = Array.isArray(profile.PerGroup) ? profile.PerGroup : [];
      } catch (error) {
        console.error('Error fetching player profile:', error);
        this.loadFailed = true;
        this.perGroup = [];
      } finally {
        this.isLoading = false;
      }
    },
    async loadGroupRoles() {
      try {
        const groups = await getMyGroups();
        const roles = {};
        (Array.isArray(groups) ? groups : []).forEach(group => {
          roles[group.id] = group.role;
        });
        this.groupRoles = roles;
      } catch (error) {
        // Non-fatal: this only gates admin-only controls, which just stay
        // hidden (the backend's own 403 is the real boundary anyway).
        console.error('Error fetching group roles:', error);
      }
    },
    isGroupAdmin(groupId) {
      return this.groupRoles[groupId] === 'admin';
    },
    async selectGroup(groupId) {
      if (this.selectedGroupId === groupId) {
        this.selectedGroupId = '';
        return;
      }
      this.selectedGroupId = groupId;
      if (this.membersByGroup[groupId]) {
        // Already fetched earlier in this session — no need to refetch.
        return;
      }
      await this.loadMembers(groupId);
    },
    async loadMembers(groupId) {
      this.membersLoading = true;
      this.membersError = '';
      try {
        const members = await getGroupMembers(groupId);
        this.membersByGroup[groupId] = Array.isArray(members) ? members : [];
      } catch (error) {
        this.membersError = this.backendMessage(error, 'Failed to load members.');
      } finally {
        this.membersLoading = false;
      }
    },
    async toggleMemberRole(member) {
      const groupId = this.selectedGroupId;
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
    async confirmRemoveMember(member) {
      const groupId = this.selectedGroupId;
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
      const groupId = this.selectedGroupId;
      const email = this.inviteEmailInput.trim();
      if (!email) {
        return;
      }
      this.inviteLoading[member.id] = true;
      this.inviteErrors[member.id] = '';
      try {
        await invitePlayer(groupId, member.id, email);
        this.inviteSuccess[member.id] = `Invite sent to ${email}. There is no email delivery configured yet in this environment — the link is logged on the server.`;
        this.inviteFormFor = '';
        this.inviteEmailInput = '';
        // Re-fetch so member.email reflects the claim and the "Invite"
        // action disappears for this row.
        await this.loadMembers(groupId);
      } catch (error) {
        this.inviteErrors[member.id] = this.backendMessage(error, 'Failed to send the invite.');
      } finally {
        this.inviteLoading[member.id] = false;
      }
    },
    openSettings(row) {
      this.settingsForGroup = { id: row.GroupID, name: row.GroupName };
    },
    backendMessage(error, fallback) {
      return error.response?.data?.error || fallback;
    },
    getPlayerInitials(name) {
      return (name || '')
        .split(' ')
        .map(word => word.charAt(0).toUpperCase())
        .join('')
        .slice(0, 2);
    },
    formatPlayerNameForDisplay(name) {
      return (name || '')
        .split(' ')
        .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
        .join(' ');
    }
  }
};
</script>

<style scoped>
/* Layout mirrors MatchesAndStandings.vue — same container/header/table treatment, so
   the two ranking pages read as one screen family. */
.profile-container {
  background-color: var(--bg-secondary);
}

.profile-header {
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--secondary-color) 100%);
  color: white;
  padding: 1rem 0;
}

.profile-section {
  padding: 2rem 0;
}

.profile-layout {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

/* Overall card */
.overall-card {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.overall-identity {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.overall-name {
  font-size: 1.35rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.overall-scope {
  margin: 0.125rem 0 0;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.overall-stats {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 0.75rem;
}

.overall-stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
  padding: 0.75rem 0.5rem;
  border-radius: var(--border-radius);
  background-color: var(--bg-tertiary);
}

.overall-stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-primary);
}

.overall-stat.highlight .overall-stat-value {
  color: var(--primary-color);
}

.overall-stat-label {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--text-secondary);
}

/* Groups card */
.section-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 1rem;
}

.groups-bar-container {
  position: relative;
}

.groups-bar {
  display: flex;
  gap: 1rem;
  overflow-x: auto;
  scroll-behavior: smooth;
  padding-bottom: 0.25rem;
}

.group-card-horizontal {
  position: relative;
  flex: 0 0 200px;
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  padding: 1rem;
  cursor: pointer;
  border: 2px solid transparent;
  transition: all var(--transition-smooth);
}

.group-card-horizontal:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
  background-color: var(--bg-primary);
}

.group-card-horizontal.active {
  border-color: var(--primary-color);
  background-color: var(--bg-primary);
  box-shadow: var(--shadow-lg);
}

.group-card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
  padding-right: 1.5rem;
}

.group-card-name {
  font-weight: 700;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-card-stats {
  display: flex;
  gap: 0.75rem;
}

.group-card-stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
}

.group-card-stat-value {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--text-primary);
}

.group-card-stat-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--text-secondary);
}

.group-settings-btn {
  position: absolute;
  top: 0.75rem;
  right: 0.75rem;
  background: none;
  border: none;
  padding: 0.2rem;
  cursor: pointer;
  color: var(--text-secondary);
  border-radius: 50%;
  transition: all var(--transition-fast);
}

.group-settings-btn:hover {
  color: var(--primary-color);
  background-color: var(--bg-secondary);
}

.group-settings-btn svg {
  width: 16px;
  height: 16px;
  display: block;
}

/* Roster panel */
.roster-panel-container {
  margin-top: 1.5rem;
  padding: 1.25rem;
  background-color: var(--bg-tertiary);
}

.roster-title {
  margin: 0 0 1rem;
  font-size: 1.05rem;
  font-weight: 700;
  color: var(--text-primary);
}

.roster-panel-enter-active,
.roster-panel-leave-active {
  transition: all var(--transition-smooth);
}

.roster-panel-enter-from,
.roster-panel-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

.member-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-height: 22rem;
  overflow-y: auto;
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
  flex-shrink: 0;
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

.success-message {
  color: var(--primary-color);
  font-size: 0.875rem;
  margin-top: 0.5rem;
}

/* Table — same rules as PointsStandingsTable.vue's ranking */
.standings-table-container {
  overflow-x: auto;
}

/* Responsive */
@media (max-width: 768px) {
  .profile-header {
    padding: 2rem 0;
  }

  .overall-stats {
    grid-template-columns: repeat(3, 1fr);
  }

  .group-card-horizontal {
    flex-basis: 170px;
  }
}
</style>
