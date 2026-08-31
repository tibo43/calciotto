<template>
  <div class="match-detail-container">
    <!-- Loading State -->
    <div v-if="isLoading" class="loading-container">
      <div class="loading-spinner"></div>
      <p class="loading-text">Loading match details...</p>
    </div>

    <!-- Match Content -->
    <div v-else-if="match && match.Teams && match.Teams.length > 0" class="match-content">
      <div class="container">
        <!-- Match Score Overview -->
        <div class="score-overview card-base">
          <div class="match-title">
            <div class="match-title-left">
              <button @click="goBack" class="match-back-btn" aria-label="Back to Matches">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="15,18 9,12 15,6" />
                </svg>
              </button>
            </div>
            <div class="match-title-right">
              <span class="match-date">{{ formatDate(match?.Date) }}</span>
              <div class="match-status-badge" :class="getMatchStatus()">
                {{ getMatchStatusText() }}
              </div>
            </div>
          </div>

          <div class="teams-score">
            <!-- Team 1 -->
            <div class="team-score">
              <div class="team-info">
                <div class="team-color" :style="{ backgroundColor: getTeamColor(match.Teams[0].Colour) }"></div>
                <h3 class="team-name">{{ match.Teams[0].Name }}</h3>
              </div>
              <div class="score">{{ match.Teams[0].Score || 0 }}</div>
            </div>

            <!-- VS Divider -->
            <div class="vs-divider">
              <div class="vs-circle">
                <span>VS</span>
              </div>
            </div>

            <!-- Team 2 -->
            <div class="team-score">
              <div class="team-info">
                <div class="team-color" :style="{ backgroundColor: getTeamColor(match.Teams[1].Colour) }"></div>
                <h3 class="team-name">{{ match.Teams[1].Name }}</h3>
              </div>
              <div class="score">{{ match.Teams[1].Score || 0 }}</div>
            </div>
          </div>
        </div>

        <!-- Team Management Section -->
        <div v-if="match.Teams && match.Teams.length > 0" class="team-management card-base card-large">
          <div class="management-header">
            <!-- Global actions — they apply to the whole match, not to
                 whichever team tab happens to be selected, so they come
                 before the team switcher rather than being grouped with it. -->
            <div class="action-buttons">
              <button v-if="isAdmin" @click="saveChanges" class="btn-base btn-primary btn-small" :disabled="isSaving">
                <svg v-if="!isSaving" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
                  <polyline points="17,21 17,13 7,13 7,21" />
                  <polyline points="7,3 7,8 15,8" />
                </svg>
                <div v-else class="loading-spinner-small"></div>
                {{ isSaving ? 'Saving...' : 'Save Changes' }}
              </button>
              <!-- Clearly separated from Save Changes so a click can't be
                   mistaken between the two destructive/non-destructive actions. -->
              <button v-if="isAdmin" @click="confirmDeleteMatch" class="btn-base btn-danger btn-small delete-match-btn"
                :disabled="isDeleting">
                <svg v-if="!isDeleting" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3,6 5,6 21,6" />
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                </svg>
                <div v-else class="loading-spinner-small"></div>
                {{ isDeleting ? 'Deleting...' : 'Delete Match' }}
              </button>
            </div>

            <!-- Team switcher + "Add player to this team" right next to it —
                 Add Player acts on whichever tab is active, so it lives here
                 rather than among the global actions above. -->
            <div class="tabs-buttons">
              <button v-for="(team, index) in match.Teams" :key="team.ID" @click="activeTeam = index"
                :class="['tab-button', { active: activeTeam === index }]">
                <div class="team-color-small" :style="{ backgroundColor: getTeamColor(team.Colour) }"></div>
                {{ team.Name }} ({{ team.Players ? team.Players.length : 0 }})
              </button>
              <button v-if="isAdmin" @click="showModal" class="add-player-icon-btn" aria-label="Add player">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19" />
                  <line x1="5" y1="12" x2="19" y2="12" />
                </svg>
              </button>
            </div>
          </div>

          <!-- Players List — one compact row per player (avatar, name, goal
               counter, remove) instead of a header row plus a separate goal
               row, so the list needs far less scrolling. -->
          <div v-if="match.Teams[activeTeam] && match.Teams[activeTeam].Players" class="players-grid">
            <div v-for="(player, playerIndex) in match.Teams[activeTeam].Players" :key="playerIndex"
              class="player-card">
              <div class="player-info">
                <h4 class="player-name">{{ formatPlayerNameForDisplay(player.Name) }}</h4>
              </div>

              <div class="goal-management">
                <button v-if="isAdmin" @click="updateGoals(playerIndex, -1)"
                  :disabled="!player.GoalNumber || player.GoalNumber <= 0" class="goal-btn decrease">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="5" y1="12" x2="19" y2="12" />
                  </svg>
                </button>

                <span class="goal-count">{{ player.GoalNumber || 0 }}</span>

                <button v-if="isAdmin" @click="updateGoals(playerIndex, 1)" class="goal-btn increase">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="12" y1="5" x2="12" y2="19" />
                    <line x1="5" y1="12" x2="19" y2="12" />
                  </svg>
                </button>
              </div>

              <button v-if="isAdmin" @click="removePlayer(playerIndex)" class="btn-danger-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Error State -->
    <div v-else class="error-state">
      <div class="empty-content">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <line x1="15" y1="9" x2="9" y2="15" />
          <line x1="9" y1="9" x2="15" y2="15" />
        </svg>
        <h3 class="empty-title">Match not found</h3>
        <p class="empty-description">The match you're looking for doesn't exist or failed to load.</p>
        <button @click="goBack" class="btn-base btn-primary">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="15,18 9,12 15,6" />
          </svg>
          Go Back
        </button>
      </div>
    </div>

    <!-- Enhanced Multi-Player Modal -->
    <div v-if="showAddPlayerModal" class="modal-overlay" @click="closeModal">
      <div class="enhanced-multi-player-modal" @click.stop>
        <div class="modal-header enhanced-modal-header">
          <h3>Add Players to {{ match.Teams[activeTeam].Name }} Team</h3>
          <button @click="closeModal" class="modal-close">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div class="modal-body enhanced-modal-body">
          <!-- Search Section -->
          <div class="search-section">
            <div class="form-group">
              <label for="playerSearch">Search Players</label>
              <div class="input-wrapper">
                <input v-model="playerSearchTerm" type="text" id="playerSearch" class="form-input"
                  placeholder="Search for players to add..." @input="onPlayerSearch">
                <div v-if="isLoadingPlayers" class="search-loading">
                  <div class="spinner-small"></div>
                </div>
              </div>
              <div v-if="showCreatePlayerOption" class="create-player-section">
                <div class="create-player-prompt">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="info-icon">
                    <circle cx="12" cy="12" r="10" />
                    <line x1="12" y1="16" x2="12" y2="12" />
                    <line x1="12" y1="8" x2="12.01" y2="8" />
                  </svg>
                  <span>Player "{{ formatPlayerNameForDisplay(playerSearchTerm) }}" not found.</span>
                </div>
                <button @click="createNewPlayer" :disabled="isCreatingPlayer" class="create-player-btn">
                  <svg v-if="!isCreatingPlayer" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                    <circle cx="8.5" cy="7" r="4" />
                    <line x1="20" y1="8" x2="20" y2="14" />
                    <line x1="23" y1="11" x2="17" y2="11" />
                  </svg>
                  <div v-else class="spinner-small"></div>
                  {{ isCreatingPlayer ? 'Creating...' : `Create "${formatPlayerNameForDisplay(playerSearchTerm)}"` }}
                </button>
              </div>
            </div>
          </div>

          <!-- Two-column layout -->
          <div class="players-columns">
            <!-- Selected Players Column -->
            <div class="column selected-column">
              <div class="column-header">
                <h4>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="header-icon">
                    <path d="M9 12l2 2 4-4" />
                    <circle cx="12" cy="12" r="10" />
                  </svg>
                  Selected Players
                  <span class="count-badge">{{ selectedPlayers.length }}</span>
                </h4>
              </div>

              <div class="column-content custom-scrollbar">
                <div v-if="selectedPlayers.length === 0" class="empty-state">
                  <div class="empty-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="12" cy="12" r="10" />
                      <line x1="12" y1="8" x2="12" y2="16" />
                      <line x1="8" y1="12" x2="16" y2="12" />
                    </svg>
                  </div>
                  <p>No players selected</p>
                  <span>Select players from the right to add them</span>
                </div>

                <div v-else class="selected-players-list">
                  <div v-for="(player, index) in selectedPlayers" :key="`selected-${player.ID || player.Name}`"
                    class="selected-player-item">
                    <div class="player-info">
                      <span class="player-name">{{ formatPlayerNameForDisplay(player.Name) }}</span>
                    </div>
                    <button @click="removeSelectedPlayer(index)" class="remove-selected-btn">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <line x1="18" y1="6" x2="6" y2="18" />
                        <line x1="6" y1="6" x2="18" y2="18" />
                      </svg>
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <!-- Available Players Column -->
            <div class="column available-column">
              <div class="column-header">
                <h4>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="header-icon">
                    <circle cx="12" cy="12" r="10" />
                    <path d="M16 12l-4-4-4 4" />
                    <path d="M16 16l-4-4-4 4" />
                  </svg>
                  Available Players
                  <span class="count-badge">{{ filteredAvailablePlayers.length }}</span>
                </h4>
              </div>

              <div class="column-content custom-scrollbar">
                <div v-if="filteredAvailablePlayers.length === 0 && !isLoadingPlayers" class="empty-state">
                  <div class="empty-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="12" cy="12" r="10" />
                      <line x1="4.93" y1="4.93" x2="19.07" y2="19.07" />
                    </svg>
                  </div>
                  <p v-if="playerSearchTerm">No players found</p>
                  <p v-else>No available players</p>
                  <span v-if="playerSearchTerm">Try adjusting your search term</span>
                </div>

                <div v-else class="available-players-list">
                  <button v-for="player in filteredAvailablePlayers" :key="`available-${player.ID || player.Name}`"
                    @click="addPlayerToSelection(player)" class="available-player-item"
                    :disabled="isPlayerSelected(player) || isPlayerInAnyTeam(player.Name)">
                    <div class="player-info">
                      <span class="player-name">{{ formatPlayerNameForDisplay(player.Name) }}</span>
                    </div>
                    <div class="player-status">
                      <span v-if="isPlayerSelected(player)" class="status-selected">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <path d="M9 12l2 2 4-4" />
                          <circle cx="12" cy="12" r="10" />
                        </svg>
                      </span>
                      <span v-else-if="isPlayerInAnyTeam(player.Name)" class="status-in-match">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <circle cx="12" cy="12" r="10" />
                          <line x1="15" y1="9" x2="9" y2="15" />
                          <line x1="9" y1="9" x2="15" y2="15" />
                        </svg>
                      </span>
                      <span v-else class="status-available">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <circle cx="12" cy="12" r="10" />
                          <line x1="12" y1="8" x2="12" y2="16" />
                          <line x1="8" y1="12" x2="16" y2="12" />
                        </svg>
                      </span>
                    </div>
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div class="modal-footer enhanced-footer">
            <div class="footer-info">
              <div class="selection-summary">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="summary-icon">
                  <path d="M9 12l2 2 4-4" />
                  <circle cx="12" cy="12" r="10" />
                </svg>
                <span v-if="selectedPlayers.length > 0" class="selection-count">
                  {{ selectedPlayers.length }} player{{ selectedPlayers.length !== 1 ? 's' : '' }} selected
                </span>
                <span v-else class="selection-count empty">
                  No players selected
                </span>
              </div>
            </div>
            <div class="footer-buttons">
              <button @click="closeModal" class="btn-base btn-cancel">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
                Cancel
              </button>
              <button @click="addSelectedPlayersToTeam" :disabled="selectedPlayers.length === 0"
                class="btn-base btn-primary enhanced-confirm">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M9 12l2 2 4-4" />
                  <circle cx="12" cy="12" r="10" />
                </svg>
                Add {{ selectedPlayers.length || '' }} Player{{ selectedPlayers.length !== 1 ? 's' : '' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Toast Container -->
    <div v-if="message" class="toast-container">
      <div :class="['toast-message', messageType]" @click="dismissMessage" :key="messageKey">
        {{ message }}
      </div>
    </div>
  </div>
</template>

<script>
import { getMatchDetailsByID, updateMatch, deleteMatch, getGroupMembers, createPlayer } from '@/services/api';
import { resolveActiveGroup } from '@/services/activeGroup';

export default {
  name: 'MatchDetail',
  data() {
    return {
      match: null,
      matchSnapshot: null,
      isLoading: true,
      isSaving: false,
      isDeleting: false,
      // Resolved once on load, reused when loading the match (scoped to the
      // same group) and when deleting it.
      activeGroupId: '',
      // Whether the caller is an admin of the active group — gates every
      // editing control (add/remove player, goals, save, delete). Editing and
      // deleting a match are admin-only on the backend (RequireGroupAdmin in
      // main.go); a non-admin can still view the match read-only.
      isAdmin: false,
      activeTeam: 0,
      showAddPlayerModal: false,
      message: '',
      messageType: 'success',
      // New properties for player dropdown
      allPlayers: [],
      selectedPlayers: [],
      filteredAvailablePlayers: [],
      isLoadingPlayers: false,
      playerSearchTerm: '',
      messageKey: 0,
      showCreatePlayerOption: false,
      isCreatingPlayer: false,
    };
  },
  async created() {
    try {
      const { groups, activeGroupId } = await resolveActiveGroup();
      this.activeGroupId = activeGroupId;
      this.isAdmin = groups.find(g => g.id === activeGroupId)?.role === 'admin';
    } catch (error) {
      // Degrade instead of breaking the page: no group_id (backend's own
      // first-group fallback) and isAdmin left false, which just hides the
      // admin-only controls.
      console.error('Error resolving the active group:', error);
    }
    await this.loadMatch();
  },
  beforeRouteLeave(to, from, next) {
    if (this.hasUnsavedChanges()) {
      const leave = window.confirm('You have unsaved changes. Leave without saving?');
      next(leave);
      return;
    }
    next();
  },
  methods: {
    hasUnsavedChanges() {
      if (this.isSaving || !this.match) return false;
      return JSON.stringify(this.match) !== this.matchSnapshot;
    },

    async loadMatch() {
      this.isLoading = true;
      try {
        const matchId = this.$route.params.id;
        // GetMatchDetailsByID filters on (match id, group id), so a match from
        // another group reads as "not found" — which is what we want: the URL
        // alone must not pull a match out of a group we aren't scoped to.
        // updateMatch needs no group of its own; it posts back this.match,
        // whose GroupID already comes from this response.
        this.match = await getMatchDetailsByID(matchId, this.activeGroupId);

        // Ensure each player has GoalNumber property
        if (this.match && this.match.Teams) {
          this.match.Teams.forEach(team => {
            if (team.Players) {
              team.Players.forEach(player => {
                if (!player.GoalNumber) {
                  player.GoalNumber = 0;
                }
              });
            }
          });
        }

        this.matchSnapshot = JSON.stringify(this.match);
      } catch (error) {
        console.error('Error fetching match:', error);
        this.showMessage('Error loading match details', 'error');
      } finally {
        this.isLoading = false;
      }
    },

    // Load all players when modal opens
    async loadAllPlayers() {
      if (this.allPlayers && this.allPlayers.length > 0) {
        this.filterAvailablePlayers();
        return;
      }

      this.isLoadingPlayers = true;
      try {
        await this.reloadAllPlayers();
      } finally {
        this.isLoadingPlayers = false;
      }
    },

    // Filter players to show only those not in current team
    filterAvailablePlayers() {
      if (!this.allPlayers || !Array.isArray(this.allPlayers) || this.allPlayers.length === 0) {
        this.filteredAvailablePlayers = [];
        this.checkCreatePlayerOption();
        return;
      }

      if (!this.match || !this.match.Teams || !this.match.Teams[this.activeTeam]) {
        this.filteredAvailablePlayers = [];
        this.checkCreatePlayerOption();
        return;
      }

      const currentTeamPlayers = this.match.Teams[this.activeTeam].Players || [];
      const currentTeamPlayerNames = currentTeamPlayers.map(p => p.Name.toLowerCase());

      let availablePlayers = this.allPlayers.filter(player =>
        player && player.Name && !currentTeamPlayerNames.includes(player.Name.toLowerCase())
      );

      if (this.playerSearchTerm && this.playerSearchTerm.trim()) {
        availablePlayers = availablePlayers.filter(player =>
          player.Name.toLowerCase().includes(this.playerSearchTerm.toLowerCase().trim())
        );
      }

      this.filteredAvailablePlayers = availablePlayers;
      this.checkCreatePlayerOption();
    },

    // Check if we should show the create player option
    checkCreatePlayerOption() {
      if (!this.playerSearchTerm || this.playerSearchTerm.trim().length < 2) {
        this.showCreatePlayerOption = false;
        return;
      }

      const searchTerm = this.playerSearchTerm.trim().toLowerCase();
      const exactMatch = this.allPlayers.some(player =>
        player.Name.toLowerCase() === searchTerm
      );

      // Show create option if no exact match found and search term is not empty
      this.showCreatePlayerOption = !exactMatch && this.filteredAvailablePlayers.length === 0;
    },

    // Create a new player
    async createNewPlayer() {
      if (!this.playerSearchTerm || this.playerSearchTerm.trim().length < 2) {
        this.showMessage('Please enter a valid player name', 'error');
        return;
      }

      const playerName = this.playerSearchTerm.trim();
      const playerNameLowerCase = playerName.toLowerCase(); // Backend gets lowercase

      // Check if player already exists (case-insensitive)
      const existingPlayer = this.allPlayers.find(player =>
        player.Name.toLowerCase() === playerNameLowerCase
      );

      if (existingPlayer) {
        this.showMessage('Player already exists', 'error');
        return;
      }

      this.isCreatingPlayer = true;
      try {
        const newPlayerData = {
          Name: playerNameLowerCase, // Send lowercase to backend
          // Exact key, lowercase with an underscore: CreatePlayer binds this
          // into a `json:"group_id"` field, and Gin's case-insensitive JSON
          // binding does not bridge the underscore — sending `GroupID`
          // instead would silently fail to bind and the new player would
          // fall through to the backend's own-first-group fallback instead
          // of joining this match's group.
          group_id: this.match.GroupID
        };

        await createPlayer(newPlayerData);

        // RELOAD ALL PLAYERS FROM DATABASE to ensure we have fresh data
        await this.reloadAllPlayers();

        // Find the newly created player in the fresh data
        const freshPlayer = this.allPlayers.find(player =>
          player.Name.toLowerCase() === playerNameLowerCase
        );

        if (freshPlayer) {
          // Add the new player to selection immediately
          this.addPlayerToSelection(freshPlayer);
        }

        // Clear search and hide create option
        this.playerSearchTerm = '';
        this.showCreatePlayerOption = false;
        this.filterAvailablePlayers();

        this.showMessage(`Player "${this.formatPlayerNameForDisplay(playerNameLowerCase)}" created and added to selection!`, 'success');

      } catch (error) {
        console.error('Error creating player:', error);
        // The backend's own per-group duplicate-name check (a case our own
        // client-side check against this.allPlayers can't catch, since that
        // check is global rather than scoped to this match's group) returns a
        // specific message worth showing verbatim, same pattern as
        // ForgotPassword.vue/Profile.vue. Fall back to a generic message for
        // any other kind of failure.
        const backendMessage = error.response?.data?.error;
        this.showMessage(backendMessage || 'Error creating player. Please try again.', 'error');
      } finally {
        this.isCreatingPlayer = false;
      }
    },

    // Scoped to this match's own group (this.activeGroupId, resolved once in
    // created() via resolveActiveGroup()) rather than every player in the
    // database: a player removed from the group — or belonging to a
    // different group entirely — must not be offered here.
    async reloadAllPlayers() {
      if (!this.activeGroupId) {
        // No group resolved (resolveActiveGroup failed) — degrade to an
        // empty list rather than requesting a malformed URL.
        this.allPlayers = [];
        this.filterAvailablePlayers();
        return;
      }
      try {
        const members = await getGroupMembers(this.activeGroupId);
        // getGroupMembers returns PlayerWithRole, which embeds Player's own
        // lowercase JSON fields (id, name) plus role — translate to the
        // PascalCase {ID, Name} shape the rest of this component already
        // uses for players (matching PlayerCustom's convention from the
        // getPlayers() call this replaces), role is not needed here.
        this.allPlayers = Array.isArray(members)
          ? members.map(member => ({ ID: member.id, Name: member.name }))
          : [];
        this.filterAvailablePlayers();
      } catch (error) {
        console.error('Error reloading players:', error);
        this.showMessage('Error reloading players list', 'error');
        // Don't reset allPlayers on error, keep the current data
      }
    },

    // Handle search input
    onPlayerSearch() {
      // Clear any existing timeout
      if (this.searchTimeout) {
        clearTimeout(this.searchTimeout);
      }

      // Debounce the search
      this.searchTimeout = setTimeout(() => {
        this.filterAvailablePlayers();
      }, 300);
    },

    addPlayerToSelection(player) {
      if (this.isPlayerSelected(player) || this.isPlayerInAnyTeam(player.Name)) {
        return;
      }

      const playerToAdd = {
        ID: player.ID,
        Name: player.Name,
        initialGoals: 0  // Default to 0 goals
      };

      this.selectedPlayers.push(playerToAdd);
    },

    // Remove a player from selection
    removeSelectedPlayer(index) {
      this.selectedPlayers.splice(index, 1);
    },

    // Check if player is already selected
    isPlayerSelected(player) {
      return this.selectedPlayers.some(selectedPlayer =>
        selectedPlayer.Name.toLowerCase() === player.Name.toLowerCase()
      );
    },

    // Add all selected players to the team
    async addSelectedPlayersToTeam() {
      if (this.selectedPlayers.length === 0) {
        this.showMessage('Please select at least one player', 'error');
        return;
      }

      if (!this.match.Teams || !this.match.Teams[this.activeTeam]) {
        console.error('Invalid team data');
        return;
      }

      if (!this.match.Teams[this.activeTeam].Players) {
        this.match.Teams[this.activeTeam].Players = [];
      }

      // Add each selected player to the team
      this.selectedPlayers.forEach(selectedPlayer => {
        // Double-check if player is not already in the team
        if (!this.isPlayerInCurrentTeam(selectedPlayer.Name)) {
          const newPlayer = {
            ID: selectedPlayer.ID,
            Name: selectedPlayer.Name,
            GoalNumber: selectedPlayer.initialGoals || 0
          };

          this.match.Teams[this.activeTeam].Players.push(newPlayer);
        }
      });

      // Update team score
      this.updateTeamScore();

      // Close modal and show success message
      this.closeModal();
    },

    // Check if player is in current team
    isPlayerInCurrentTeam(playerName) {
      if (!this.match.Teams || !this.match.Teams[this.activeTeam] || !this.match.Teams[this.activeTeam].Players) {
        return false;
      }

      return this.match.Teams[this.activeTeam].Players.some(player =>
        player.Name.toLowerCase() === playerName.toLowerCase()
      );
    },

    // Check if player is in any team in this match
    isPlayerInAnyTeam(playerName) {
      if (!this.match.Teams) return false;

      return this.match.Teams.some(team =>
        team.Players && team.Players.some(player =>
          player.Name.toLowerCase() === playerName.toLowerCase()
        )
      );
    },

    // Show modal and load players
    async showModal() {
      this.showAddPlayerModal = true;
      await this.loadAllPlayers();
    },

    closeModal() {
      this.showAddPlayerModal = false;
      this.selectedPlayers = [];
      this.playerSearchTerm = '';
      this.filteredAvailablePlayers = [];
      this.showCreatePlayerOption = false;
      this.isCreatingPlayer = false;
      if (this.searchTimeout) {
        clearTimeout(this.searchTimeout);
      }
    },

    // Existing methods remain the same...
    goBack() {
      this.$router.go(-1);
    },

    updateGoals(playerIndex, change) {
      if (!this.match.Teams || !this.match.Teams[this.activeTeam] || !this.match.Teams[this.activeTeam].Players) {
        console.error('Invalid team or players data');
        return;
      }

      const player = this.match.Teams[this.activeTeam].Players[playerIndex];
      if (!player) {
        console.error('Player not found at index:', playerIndex);
        return;
      }

      const newGoals = (player.GoalNumber || 0) + change;

      if (newGoals >= 0) {
        player.GoalNumber = newGoals;
        this.updateTeamScore();
      }
    },

    updateTeamScore() {
      if (!this.match.Teams || !this.match.Teams[this.activeTeam] || !this.match.Teams[this.activeTeam].Players) {
        return;
      }

      const team = this.match.Teams[this.activeTeam];
      team.Score = team.Players.reduce((total, player) => total + (player.GoalNumber || 0), 0);
    },

    removePlayer(playerIndex) {
      if (!this.match.Teams || !this.match.Teams[this.activeTeam] || !this.match.Teams[this.activeTeam].Players) {
        console.error('Invalid team or players data');
        return;
      }

      this.match.Teams[this.activeTeam].Players.splice(playerIndex, 1);
      this.updateTeamScore();
    },

    hasEmptyTeam() {
      if (!this.match || !this.match.Teams) return true;

      return this.match.Teams.some(team =>
        !team.Players || team.Players.length === 0
      );
    },

    async saveChanges() {
      if (this.hasEmptyTeam()) {
        this.showMessage('Each team requires at least 1 player', 'error');
        return;
      }
      this.isSaving = true;
      try {
        await updateMatch(this.match.ID, this.match);
        // Update the snapshot before navigating away: beforeRouteLeave's
        // hasUnsavedChanges() check runs as part of this same navigation,
        // and would otherwise see the just-saved match as still "dirty"
        // against the pre-save snapshot and pop the "leave without
        // saving?" confirm right after a successful save.
        this.matchSnapshot = JSON.stringify(this.match);
        this.goBack();
      } catch (error) {
        console.error('Error saving match:', error);
        this.showMessage('Error saving changes', 'error');
      } finally {
        this.isSaving = false;
      }
    },

    // Same confirm-before-acting pattern as beforeRouteLeave's unsaved-changes
    // guard above — a plain window.confirm rather than a custom modal, since
    // that's the existing precedent in this file for a destructive/irreversible
    // action the user could regret.
    confirmDeleteMatch() {
      if (this.isDeleting) return;
      const confirmed = window.confirm('Delete this match? This cannot be undone.');
      if (!confirmed) return;
      this.deleteMatchNow();
    },

    async deleteMatchNow() {
      this.isDeleting = true;
      try {
        await deleteMatch(this.match.ID, this.activeGroupId);
        this.goBack();
      } catch (error) {
        console.error('Error deleting match:', error);
        this.showMessage('Error deleting match. Please try again.', 'error');
      } finally {
        this.isDeleting = false;
      }
    },

    showMessage(text, type = 'success') {
      this.message = text;
      this.messageType = type;
      this.messageKey++; // Trigger re-render for animations

      // Auto-dismiss after 4 seconds
      if (this.messageTimeout) {
        clearTimeout(this.messageTimeout);
      }

      this.messageTimeout = setTimeout(() => {
        this.dismissMessage();
      }, 3000);
    },

    dismissMessage() {
      const messageEl = document.querySelector('.message');
      if (messageEl) {
        messageEl.classList.add('toast-exit');
        setTimeout(() => {
          this.message = '';
          if (this.messageTimeout) {
            clearTimeout(this.messageTimeout);
          }
        }, 300);
      }
    },

    formatDate(dateString) {
      try {
        const date = new Date(dateString);
        return date.toLocaleDateString('en-US', {
          weekday: 'long',
          year: 'numeric',
          month: 'long',
          day: 'numeric'
        });
      } catch (error) {
        return dateString;
      }
    },

    getTeamColor(colour) {
      if (colour && colour.startsWith('#')) {
        return colour;
      }

      const colorMap = {
        'red': '#ef4444',
        'blue': '#3b82f6',
        'green': '#10b981',
        'yellow': '#f59e0b',
        'purple': '#8b5cf6',
        'orange': '#f97316',
        'pink': '#ec4899',
        'cyan': '#06b6d4',
        'white': '#f8fafc',
        'black': '#1f2937'
      };

      return colorMap[colour.toLowerCase()] || '#6b7280';
    },

    formatPlayerNameForDisplay(name) {
      // Convert to title case for display (capitalize first letter of each word)
      return name.split(' ')
        .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
        .join(' ');
    },

    getTotalGoals() {
      if (!this.match || !this.match.Teams) return 0;
      return this.match.Teams.reduce((total, team) => total + (team.Score || 0), 0);
    },

    getTotalPlayers() {
      if (!this.match || !this.match.Teams) return 0;
      return this.match.Teams.reduce((total, team) => total + (team.Players ? team.Players.length : 0), 0);
    },

    getMatchStatus() {
      const totalGoals = this.getTotalGoals();
      if (totalGoals === 0) return 'upcoming';
      return 'completed';
    },

    getMatchStatusText() {
      const status = this.getMatchStatus();
      switch (status) {
        case 'upcoming': return 'Upcoming';
        case 'completed': return 'Completed';
        default: return 'Unknown';
      }
    },

    formatDateShort(dateString) {
      try {
        const date = new Date(dateString);
        return date.toLocaleDateString('en-US', {
          month: 'short',
          day: 'numeric'
        });
      } catch (error) {
        return dateString;
      }
    }
  }
};
</script>

<style scoped>
/* Component-specific styles that couldn't be moved to global */

.match-detail-container {
  background-color: var(--bg-secondary);
}

/* Match Content */
.match-content {
  padding: 2rem 0;
}

/* Score Overview */
.score-overview {
  margin-bottom: 2rem;
}

.match-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.75rem 1rem;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.match-title-left,
.match-title-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.match-back-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  background: none;
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.match-back-btn:hover {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
}

.match-back-btn svg {
  width: 18px;
  height: 18px;
}

.match-date {
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.match-status-badge {
  padding: 0.3rem 0.65rem;
  border-radius: 20px;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.match-status-badge.upcoming {
  background-color: #fef3c7;
  color: #92400e;
}

.match-status-badge.completed {
  background-color: #d1fae5;
  color: #065f46;
}

.teams-score {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 1.5rem;
}

/* One row per team — name/colour and score side by side — at every
   breakpoint, rather than a large stacked block on desktop and a
   separately-tuned compact row on mobile. */
.team-score {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.85rem 1.1rem;
  background-color: var(--bg-secondary);
  border-radius: var(--border-radius);
  border: 1px solid var(--border-color);
  transition: all var(--transition-fast);
}

.team-score:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.team-info {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 0.6rem;
  text-align: left;
  min-width: 0;
}

.team-info .team-color {
  width: 18px;
  height: 18px;
  border-width: 2px;
  flex-shrink: 0;
}

.team-name {
  font-size: 1.05rem;
  font-weight: 700;
  color: var(--text-primary);
  text-transform: capitalize;
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.score {
  font-size: 1.75rem;
  font-weight: 900;
  color: var(--primary-color);
  text-shadow: 0 2px 4px rgba(16, 185, 129, 0.2);
  line-height: 1;
  flex-shrink: 0;
}

.vs-divider {
  display: flex;
  align-items: center;
  justify-content: center;
}

.vs-circle {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 800;
  font-size: 0.7rem;
  flex-shrink: 0;
  box-shadow: var(--shadow-lg);
}

/* Team Management */
.team-management {
  margin-bottom: 2rem;
}

.management-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  margin-bottom: 2rem;
  padding: 1rem;
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  border: 1px solid var(--border-color);
}

.tabs-buttons {
  display: flex;
  align-items: center;
  gap: 1rem;
}

/* "Add player to this team" — sits right next to the team switcher it acts
   on, rather than among the global Save/Delete actions. Dashed border
   echoes the "+" add-match card used in the matches list, for a consistent
   "+" affordance across the app. */
.add-player-icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2.5rem;
  height: 2.5rem;
  flex-shrink: 0;
  border: 2px dashed var(--border-color);
  border-radius: var(--border-radius);
  background: none;
  color: var(--primary-color);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.add-player-icon-btn:hover {
  border-color: var(--primary-color);
  background-color: var(--bg-tertiary);
}

.add-player-icon-btn svg {
  width: 20px;
  height: 20px;
}

.tab-button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem 1.5rem;
  background-color: var(--bg-primary);
  border: 2px solid var(--border-color);
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-weight: 500;
  color: var(--text-secondary);
}

.tab-button:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.tab-button.active {
  background-color: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.action-buttons {
  display: flex;
  gap: 1rem;
  align-items: center;
}

/* Pushed to the far end of the action bar and separated with extra margin so
   a click can't land here by mistake while reaching for Save Changes. */
.delete-match-btn {
  margin-left: 1.5rem;
}

/* Players list — one compact row per player instead of a taller card with
   a header row and a separate goal row, so the list needs far less
   scrolling to see everyone on a team. */
.players-grid {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.player-card {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  padding: 0.6rem 0.85rem;
  border: 1px solid var(--border-color);
  transition: all var(--transition-fast);
}

.player-card:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.player-info {
  flex: 1 1 auto;
  min-width: 0;
}

.player-info .player-name {
  /* It's an <h4>, which carries its own default vertical margin (unlike
     the <span> used for a player name elsewhere in this file) — left in
     place, that margin box (not the visible text) is what .player-card's
     align-items:center actually centers, throwing off the row's height
     and vertical alignment for no visual benefit. */
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.btn-danger-icon {
  background-color: var(--danger-color);
  color: white;
  border: none;
  border-radius: 50%;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-danger-icon:hover {
  background-color: var(--danger-hover);
}

.btn-danger-icon svg {
  width: 16px;
  height: 16px;
}

/* Goal Management */
.goal-management {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.goal-btn {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 2px solid var(--primary-color);
  background-color: var(--bg-primary);
  color: var(--primary-color);
  cursor: pointer;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.goal-btn:hover:not(:disabled) {
  background-color: var(--primary-color);
  color: white;
}

.goal-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.goal-btn svg {
  width: 14px;
  height: 14px;
}

.goal-count {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--primary-color);
  min-width: 24px;
  text-align: center;
}

/* Enhanced Multi-Player Modal */
.enhanced-multi-player-modal {
  background-color: var(--bg-primary);
  border-radius: var(--border-radius-lg);
  box-shadow: var(--shadow-xl);
  max-width: 900px;
  width: 95%;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-color);
}

.enhanced-modal-header {
  background: linear-gradient(135deg, var(--primary-color), var(--secondary-color)) !important;
  color: white !important;
  border-radius: var(--border-radius-lg) var(--border-radius-lg) 0 0 !important;
  border-bottom: none !important;
}

.enhanced-modal-header h3 {
  color: white !important;
}

.enhanced-modal-header .modal-close {
  color: rgba(255, 255, 255, 0.8) !important;
}

.enhanced-modal-header .modal-close:hover {
  background-color: rgba(255, 255, 255, 0.1) !important;
  color: white !important;
}

.enhanced-modal-body {
  flex: 1;
  padding: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* Search Section */
.search-section {
  flex-shrink: 0;
  padding: 1.5rem;
  background-color: var(--bg-secondary);
  border-bottom: 2px solid var(--border-color);
}

.search-section .form-group {
  margin-bottom: 0;
}

.search-section label {
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 0.75rem;
}

.input-wrapper {
  position: relative;
}

.search-loading {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
}

/* Create Player Section */
.create-player-section {
  margin-top: 1rem;
  padding: 1rem;
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  border: 1px solid var(--border-color);
}

.create-player-prompt {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.info-icon {
  width: 16px;
  height: 16px;
  color: var(--secondary-color);
}

.create-player-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background-color: var(--accent-color);
  color: white;
  border: none;
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-weight: 500;
  font-size: 0.875rem;
}

.create-player-btn:hover:not(:disabled) {
  background-color: #d97706;
  transform: translateY(-1px);
}

.create-player-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.create-player-btn svg {
  width: 16px;
  height: 16px;
}

/* Two-column layout */
.players-columns {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  min-height: 0;
}

.column {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.selected-column {
  border-right: 1px solid var(--border-color);
}

.column-header {
  flex-shrink: 0;
  padding: 1rem 1.5rem;
  background: linear-gradient(135deg, var(--bg-tertiary), var(--bg-secondary));
  border-bottom: 1px solid var(--border-color);
}

.column-header h4 {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.header-icon {
  width: 18px;
  height: 18px;
  color: var(--primary-color);
}

.count-badge {
  background-color: var(--primary-color);
  color: white;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.25rem 0.5rem;
  border-radius: 12px;
  min-width: 20px;
  text-align: center;
}

.column-content {
  flex: 1;
  padding: 1rem;
  overflow-y: auto;
  min-height: 0;
}

/* Selected Players List */
.selected-players-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.selected-player-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem;
  background-color: var(--bg-secondary);
  border-radius: var(--border-radius);
  border: 1px solid var(--border-color);
  transition: all var(--transition-fast);
}

.selected-player-item:hover {
  background-color: var(--bg-tertiary);
  transform: translateX(-2px);
}

.remove-selected-btn {
  background: none;
  border: none;
  color: var(--danger-color);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: 50%;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
}

.remove-selected-btn:hover {
  background-color: rgba(239, 68, 68, 0.1);
}

.remove-selected-btn svg {
  width: 16px;
  height: 16px;
}

/* Available Players List */
.available-players-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.available-player-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem;
  background: none;
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
  text-align: left;
  width: 100%;
}

.available-player-item:hover:not(:disabled) {
  background-color: var(--bg-tertiary);
  border-color: var(--primary-color);
  transform: translateX(2px);
}

.available-player-item:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.player-status {
  display: flex;
  align-items: center;
}

.status-selected {
  color: var(--primary-color);
}

.status-in-match {
  color: var(--danger-color);
}

.status-available {
  color: var(--text-light);
}

.player-status svg {
  width: 18px;
  height: 18px;
}

/* Enhanced Footer */
.enhanced-footer {
  flex-shrink: 0;
  background-color: var(--bg-secondary);
  border-top: 2px solid var(--border-color);
  padding: 1.5rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
}

.footer-info {
  flex: 1;
}

.selection-summary {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--text-secondary);
}

.summary-icon {
  width: 16px;
  height: 16px;
  color: var(--primary-color);
}

.selection-count {
  font-weight: 500;
}

.selection-count.empty {
  color: var(--text-light);
}

.footer-buttons {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.enhanced-confirm {
  min-width: 120px;
}

/* Error State */
.error-state {
  padding: 4rem 0;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 50vh;
}

/* Responsive Design */
@media (max-width: 768px) {
  .match-content {
    padding: 1rem 0;
  }

  /* Score overview: card-base/card-large's own padding is sized for
     desktop — override it directly here rather than fighting it. */
  .score-overview {
    padding: 1rem;
    margin-bottom: 1rem;
  }

  /* No more "Match" heading taking up half the row (see .match-title-left
     in the template) — the back button, date and status badge now fit on
     one row even on a narrow screen, so the wrap the base rule allows for
     is no longer needed. */
  .match-title {
    flex-wrap: nowrap;
    margin-bottom: 1rem;
    padding-bottom: 0.75rem;
  }

  /* The full "Sunday, August 23, 2026" format (see formatDate()) is long
     next to the back button and status badge sharing this row on a narrow
     screen — shrink it rather than truncate or reflow the row. */
  .match-date {
    font-size: 0.75rem;
  }

  /* Both teams already sit on one row at every breakpoint (see .team-score
     above) — mobile just gets a bit tighter to fit the narrower width. */
  .teams-score {
    gap: 0.5rem;
  }

  .team-score {
    padding: 0.6rem 0.75rem;
    gap: 0.5rem;
  }

  .team-name {
    font-size: 1.15rem;
  }

  .team-info .team-color {
    width: 20px;
    height: 20px;
  }

  .score {
    font-size: 1.65rem;
  }

  .vs-circle {
    width: 32px;
    height: 32px;
    font-size: 0.7rem;
  }

  .team-management {
    padding: 1rem;
    margin-bottom: 1rem;
  }

  .management-header {
    flex-direction: column;
    align-items: stretch;
    gap: 0.75rem;
    margin-bottom: 1rem;
    padding: 0.75rem;
  }

  .tabs-buttons {
    justify-content: center;
    gap: 0.5rem;
  }

  .tab-button {
    padding: 0.6rem 1rem;
    font-size: 0.9rem;
  }

  /* Side by side on one row instead of stacked full-width — each button
     shares the row equally. */
  .action-buttons {
    flex-wrap: nowrap;
    gap: 0.5rem;
  }

  .action-buttons .btn-base {
    flex: 1;
    justify-content: center;
  }

  /* They're already visually separated by sharing the row 50/50 with a
     gap, so the extra desktop-only margin meant to prevent a misclick
     would just throw the 50/50 split off here. */
  .delete-match-btn {
    margin-left: 0;
  }

  /* Roster row: player name rendered as an <h4> with no font-size of its
     own (see .player-name in global-styles.css), so it falls back to the
     browser's default ~1rem heading size. 0.85rem (an earlier pass) read
     as too small next to the rest of the row — 1rem is the actual target,
     just without the <h4>'s own oversized default margin (stripped in the
     base rule above, not this override). */
  .player-info .player-name {
    font-size: 1rem;
  }

  .enhanced-multi-player-modal {
    max-width: 95%;
    max-height: 95vh;
  }

  .search-section {
    padding: 1rem;
  }

  .column-header {
    padding: 0.75rem 1rem;
  }

  .column-content {
    padding: 0.75rem;
  }

  .players-columns {
    grid-template-columns: 1fr;
  }

  .selected-column {
    border-right: none;
    border-bottom: 1px solid var(--border-color);
  }

  .enhanced-footer {
    flex-direction: column;
    align-items: stretch;
    gap: 0.75rem;
    padding: 1rem;
  }

  .footer-buttons {
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .match-content {
    padding: 0.75rem 0;
  }

  .tab-button {
    padding: 0.6rem 0.85rem;
    font-size: 0.85rem;
  }

  .team-color-small {
    width: 12px;
    height: 12px;
  }
}
</style>