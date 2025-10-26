<template>
  <div class="match-detail-container">
    <!-- Header Section -->
    <section class="match-header">
      <div class="container">
        <div class="header-content">
          <button @click="goBack" class="back-button">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15,18 9,12 15,6" />
            </svg>
            Back to Matches
          </button>
          <h1 class="page-title">Match Details</h1>
          <p class="page-subtitle">{{ formatDate(match?.Date) }}</p>
        </div>
      </div>
    </section>

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
            <h2>Match Score</h2>
            <div class="match-status-badge" :class="getMatchStatus()">
              {{ getMatchStatusText() }}
            </div>
          </div>

          <div class="teams-score">
            <!-- Team 1 -->
            <div class="team-score">
              <div class="team-info">
                <div class="team-color" :style="{ backgroundColor: getTeamColor(match.Teams[0].Colour) }"></div>
                <h3 class="team-name">{{ match.Teams[0].Colour }}</h3>
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
                <h3 class="team-name">{{ match.Teams[1].Colour }}</h3>
              </div>
              <div class="score">{{ match.Teams[1].Score || 0 }}</div>
            </div>
          </div>
        </div>

        <!-- Team Management Section -->
        <div v-if="match.Teams && match.Teams.length > 0" class="team-management card-base card-large">
          <div class="management-header">
            <div class="tabs-buttons">
              <button v-for="(team, index) in match.Teams" :key="team.ID" @click="activeTeam = index"
                :class="['tab-button', { active: activeTeam === index }]">
                <div class="team-color-small" :style="{ backgroundColor: getTeamColor(team.Colour) }"></div>
                {{ team.Colour }} Team ({{ team.Players ? team.Players.length : 0 }})
              </button>
            </div>

            <div class="action-buttons">
              <button @click="showModal" class="btn-base btn-secondary btn-small">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="12" y1="8" x2="12" y2="16" />
                  <line x1="8" y1="12" x2="16" y2="12" />
                </svg>
                Add Player
              </button>
              <button @click="saveChanges" class="btn-base btn-primary btn-small" :disabled="isSaving">
                <svg v-if="!isSaving" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
                  <polyline points="17,21 17,13 7,13 7,21" />
                  <polyline points="7,3 7,8 15,8" />
                </svg>
                <div v-else class="loading-spinner-small"></div>
                {{ isSaving ? 'Saving...' : 'Save Changes' }}
              </button>
            </div>
          </div>

          <!-- Players List -->
          <div v-if="match.Teams[activeTeam] && match.Teams[activeTeam].Players" class="players-grid">
            <div v-for="(player, playerIndex) in match.Teams[activeTeam].Players" :key="playerIndex"
              class="player-card">
              <div class="player-header">
                <div class="player-avatar">
                  {{ getPlayerInitials(player.Name) }}
                </div>
                <div class="player-info">
                  <h4 class="player-name">{{ formatPlayerNameForDisplay(player.Name) }}</h4>
                  <span class="player-goals">Goals: {{ player.GoalNumber || 0 }}</span>
                </div>
                <button @click="removePlayer(playerIndex)" class="btn-danger-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="18" y1="6" x2="6" y2="18" />
                    <line x1="6" y1="6" x2="18" y2="18" />
                  </svg>
                </button>
              </div>

              <!-- Goal Management -->
              <div class="goal-management">
                <button @click="updateGoals(playerIndex, -1)" :disabled="!player.GoalNumber || player.GoalNumber <= 0"
                  class="goal-btn decrease">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="5" y1="12" x2="19" y2="12" />
                  </svg>
                </button>

                <span class="goal-count">{{ player.GoalNumber || 0 }}</span>

                <button @click="updateGoals(playerIndex, 1)" class="goal-btn increase">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="12" y1="5" x2="12" y2="19" />
                    <line x1="5" y1="12" x2="19" y2="12" />
                  </svg>
                </button>
              </div>
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
          <h3>Add Players to {{ match.Teams[activeTeam].Colour }} Team</h3>
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
                      <div class="player-avatar-small">
                        {{ getPlayerInitials(player.Name) }}
                      </div>
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
                      <div class="player-avatar-small">
                        {{ getPlayerInitials(player.Name) }}
                      </div>
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
import { getMatchDetailsByID, updateMatch, getPlayers, createPlayer } from '@/services/api';

export default {
  name: 'MatchDetail',
  mounted() {
    document.title = 'Calciotto';
  },
  data() {
    return {
      match: null,
      isLoading: true,
      isSaving: false,
      activeTeam: 0,
      showAddPlayerModal: false,
      message: '',
      messageType: 'success',
      // New properties for player dropdown
      allPlayers: [],
      filteredPlayers: [],
      selectedPlayers: [],
      filteredAvailablePlayers: [],
      isLoadingPlayers: false,
      playerSearchTerm: '',
      isSearchingPlayer: false,
      messageKey: 0,
      showCreatePlayerOption: false,
      isCreatingPlayer: false,
    };
  },
  async created() {
    await this.loadMatch();
  },
  methods: {
    async loadMatch() {
      this.isLoading = true;
      try {
        const matchId = this.$route.params.id;
        this.match = await getMatchDetailsByID(matchId);

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

    onPlayerNameInput() {
      this.playerSearchTerm = this.newPlayerName;
      this.isSearchingPlayer = true;

      // Debounce the search
      clearTimeout(this.searchTimeout);
      this.searchTimeout = setTimeout(() => {
        this.searchPlayers();
      }, 300);
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
          Name: playerNameLowerCase // Send lowercase to backend
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
        this.showMessage('Error creating player. Please try again.', 'error');
      } finally {
        this.isCreatingPlayer = false;
      }
    },

    async reloadAllPlayers() {
      try {
        const players = await getPlayers();
        this.allPlayers = Array.isArray(players) ? players : [];
        this.filterAvailablePlayers();
      } catch (error) {
        console.error('Error reloading players:', error);
        this.showMessage('Error reloading players list', 'error');
        // Don't reset allPlayers on error, keep the current data
      }
    },

    async searchPlayers() {
      if (!this.newPlayerName.trim()) {
        this.playerSuggestions = [];
        this.validationMessage = '';
        this.playerNotFound = false;
        this.playerAlreadyInTeam = false;
        this.isSearchingPlayer = false;
        return;
      }

      try {
        // Load all players if not loaded
        if (!this.allPlayers || this.allPlayers.length === 0) {
          await this.loadAllPlayers();
        }

        // Filter players based on search term
        const searchTerm = this.newPlayerName.toLowerCase().trim();
        const matchingPlayers = this.allPlayers.filter(player =>
          player && player.Name && player.Name.toLowerCase().includes(searchTerm)
        );

        this.playerSuggestions = matchingPlayers;

        // Set validation states
        if (matchingPlayers.length === 0) {
          this.playerNotFound = true;
          this.playerAlreadyInTeam = false;
          this.validationMessage = 'No players found with that name';
        } else if (matchingPlayers.length === 1) {
          const player = matchingPlayers[0];
          this.playerNotFound = false;
          this.playerAlreadyInTeam = this.isPlayerInCurrentTeam(player.Name);

          if (this.playerAlreadyInTeam) {
            this.validationMessage = `${player.Name} is already in the ${this.match.Teams[this.activeTeam].Colour} team`;
          } else if (this.isPlayerInAnyTeam(player.Name)) {
            this.validationMessage = `${player.Name} is already in another team`;
          } else {
            this.validationMessage = `${player.Name} - Ready to add`;
          }
        } else {
          this.playerNotFound = false;
          this.playerAlreadyInTeam = false;
          this.validationMessage = `Found ${matchingPlayers.length} matching players`;
        }
      } catch (error) {
        console.error('Error searching players:', error);
        this.playerSuggestions = [];
        this.validationMessage = 'Error searching players';
        this.playerNotFound = true;
      } finally {
        this.isSearchingPlayer = false;
      }
    },

    selectPlayerSuggestion(player) {
      this.selectedPlayer = player;
      this.newPlayerName = player.Name;
      this.playerSuggestions = [player]; // Show only selected player
      this.playerNotFound = false;
      this.playerAlreadyInTeam = this.isPlayerInCurrentTeam(player.Name);

      if (this.playerAlreadyInTeam) {
        this.validationMessage = `${player.Name} is already in the ${this.match.Teams[this.activeTeam].Colour} team`;
      } else if (this.isPlayerInAnyTeam(player.Name)) {
        this.validationMessage = `${player.Name} is already in another team`;
      } else {
        this.validationMessage = `${player.Name} - Ready to add`;
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

    // Select a player from the list
    selectPlayer(player) {
      this.selectedPlayer = player;
      this.newPlayerName = player.Name; // For compatibility with existing code
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

    // Add selected player to team
    async addPlayer() {
      if (!this.newPlayerName.trim()) {
        this.showMessage('Please enter a player name', 'error');
        return;
      }

      // If we have suggestions, use the first one or the selected one
      let playerToAdd = this.selectedPlayer;

      if (!playerToAdd && this.playerSuggestions.length === 1) {
        playerToAdd = this.playerSuggestions[0];
      }

      if (!playerToAdd) {
        // Try to find player by name
        playerToAdd = this.allPlayers.find(p =>
          p.Name.toLowerCase() === this.newPlayerName.toLowerCase().trim()
        );
      }

      if (!this.match.Teams || !this.match.Teams[this.activeTeam]) {
        console.error('Invalid team data');
        return;
      }

      if (!this.match.Teams[this.activeTeam].Players) {
        this.match.Teams[this.activeTeam].Players = [];
      }

      const newPlayer = {
        ID: playerToAdd.ID,
        Name: playerToAdd.Name,
        GoalNumber: this.newPlayerGoals || 0
      };

      this.match.Teams[this.activeTeam].Players.push(newPlayer);
      this.updateTeamScore();
      this.closeModal();
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
      this.isSearchingPlayer = false;
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
        this.showMessage('Match updated successfully!', 'success');
      } catch (error) {
        console.error('Error saving match:', error);
        this.showMessage('Error saving changes', 'error');
      } finally {
        this.isSaving = false;
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

    getPlayerInitials(name) {
      return name.split(' ')
        .map(word => word.charAt(0).toUpperCase())
        .join('')
        .slice(0, 2);
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

/* Header Section */
.match-header {
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--secondary-color) 100%);
  color: white;
  padding: 1rem 0;
  position: relative;
}

.header-content {
  text-align: center;
  position: relative;
}

.back-button {
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: rgba(255, 255, 255, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.3);
  color: white;
  padding: 0.5rem 1rem;
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.back-button:hover {
  background: rgba(255, 255, 255, 0.3);
}

.back-button svg {
  width: 18px;
  height: 18px;
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
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.match-title h2 {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.match-status-badge {
  padding: 0.5rem 1rem;
  border-radius: 20px;
  font-size: 0.875rem;
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

.team-score {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
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
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  text-align: center;
}

.team-name {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-primary);
  text-transform: capitalize;
  margin: 0;
}

.score {
  font-size: 2.5rem;
  font-weight: 900;
  color: var(--primary-color);
  text-shadow: 0 2px 4px rgba(16, 185, 129, 0.2);
  line-height: 1;
}

.vs-divider {
  display: flex;
  align-items: center;
  justify-content: center;
}

.vs-circle {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 800;
  font-size: 1rem;
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
  gap: 1rem;
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

/* Players Grid */
.players-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 1.5rem;
}

.player-card {
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  padding: 1.5rem;
  border: 1px solid var(--border-color);
  transition: all var(--transition-fast);
}

.player-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.player-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1rem;
}

.player-info {
  flex: 1;
}

.player-goals {
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.btn-danger-icon {
  background-color: var(--danger-color);
  color: white;
  border: none;
  border-radius: 50%;
  width: 32px;
  height: 32px;
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
  gap: 1rem;
}

.goal-btn {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 2px solid var(--primary-color);
  background-color: var(--bg-primary);
  color: var(--primary-color);
  cursor: pointer;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
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
  width: 18px;
  height: 18px;
}

.goal-count {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--primary-color);
  min-width: 40px;
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
  .header-content {
    padding: 0 3rem;
  }

  .back-button {
    position: relative;
    margin-bottom: 1rem;
    transform: none;
  }

  .teams-score {
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  .vs-divider {
    order: 2;
  }

  .team-score:first-child {
    order: 1;
  }

  .team-score:last-child {
    order: 3;
  }

  .management-header {
    flex-direction: column;
    align-items: stretch;
    gap: 1rem;
  }

  .tabs-buttons {
    justify-content: center;
  }

  .action-buttons {
    justify-content: center;
  }

  .players-grid {
    grid-template-columns: 1fr;
  }

  .enhanced-multi-player-modal {
    max-width: 95%;
    max-height: 95vh;
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
    gap: 1rem;
  }

  .footer-buttons {
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .match-content {
    padding: 1rem 0;
  }

  .tab-button {
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
  }

  .team-color-small {
    width: 12px;
    height: 12px;
  }

  .player-card {
    padding: 1rem;
  }

  .goal-btn {
    width: 32px;
    height: 32px;
  }

  .goal-count {
    font-size: 1.25rem;
  }
}
</style>