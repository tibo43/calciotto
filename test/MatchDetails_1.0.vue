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
        <div class="score-overview">
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
        <div v-if="match.Teams && match.Teams.length > 0" class="team-management">
          <div class="management-header">
            <div class="tabs-buttons">
              <button v-for="(team, index) in match.Teams" :key="team.ID" @click="activeTeam = index"
                :class="['tab-button', { active: activeTeam === index }]">
                <div class="team-color-small" :style="{ backgroundColor: getTeamColor(team.Colour) }"></div>
                {{ team.Colour }} Team ({{ team.Players ? team.Players.length : 0 }})
              </button>
            </div>

            <div class="action-buttons">
              <button @click="showModal" class="add-player-btn">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="12" y1="8" x2="12" y2="16" />
                  <line x1="8" y1="12" x2="16" y2="12" />
                </svg>
                Add Player
              </button>
              <button @click="saveChanges" class="save-button">
                <svg v-if="!isSaving" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
                  <polyline points="17,21 17,13 7,13 7,21" />
                  <polyline points="7,3 7,8 15,8" />
                </svg>
                <div v-else class="save-spinner"></div>
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
                <button @click="removePlayer(playerIndex)" class="remove-player">
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
      <div class="error-content">
        <svg class="error-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <line x1="15" y1="9" x2="9" y2="15" />
          <line x1="9" y1="9" x2="15" y2="15" />
        </svg>
        <h3 class="error-title">Match not found</h3>
        <p class="error-description">The match you're looking for doesn't exist or failed to load.</p>
        <button @click="goBack" class="error-back-btn">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="15,18 9,12 15,6" />
          </svg>
          Go Back
        </button>
      </div>
    </div>

    <div v-if="showAddPlayerModal" class="modal-overlay" @click="closeModal">
      <div class="modal-content enhanced-multi-player-modal" @click.stop>
        <div class="modal-header">
          <h3>Add Players to {{ match.Teams[activeTeam].Colour }} Team</h3>
          <button @click="closeModal" class="modal-close">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div class="modal-body enhanced-modal-body">
          <!-- Search Section - Fixed at top -->
          <div class="search-section">
            <div class="form-group">
              <label for="playerSearch">Search Players</label>
              <div class="input-wrapper">
                <input v-model="playerSearchTerm" type="text" id="playerSearch"
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

          <!-- Two-column layout for selected and available players -->
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

              <div class="column-content">
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

                    <button @click="removeSelectedPlayer(index)" class="remove-selected-btn" title="Remove player">
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

              <div class="column-content">
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
            <button @click="closeModal" class="cancel-button">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
              Cancel
            </button>
            <button @click="addSelectedPlayersToTeam" :disabled="selectedPlayers.length === 0"
              class="confirm-button enhanced-confirm">
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
    <!-- Toast Container -->
    <div v-if="message" class="toast-container">
      <div :class="['message', messageType]" @click="dismissMessage" :key="messageKey">
        {{ message }}
      </div>
    </div>
  </div>
</template>

<style scoped>
/* CSS Variables */
.match-detail-container {
  background-color: var(--bg-secondary);
}

.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1.5rem;
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

/* Loading State */
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 0;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-color);
  border-top: 3px solid var(--primary-color);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 1rem;
}

@keyframes spin {
  0% {
    transform: rotate(0deg);
  }

  100% {
    transform: rotate(360deg);
  }
}

.loading-text {
  color: var(--text-secondary);
  font-size: 1.1rem;
}

/* Match Content */
.match-content {
  padding: 2rem 0;
}

/* Score Overview */
.score-overview {
  background-color: var(--bg-primary);
  border-radius: var(--border-radius-lg);
  padding: 1.5rem;
  margin-bottom: 2rem;
  box-shadow: var(--shadow-md);
  border: 1px solid var(--border-color);
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

.team-color {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 3px solid white;
  box-shadow: var(--shadow-md);
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
  background-color: var(--bg-primary);
  border-radius: var(--border-radius-lg);
  padding: 2rem;
  box-shadow: var(--shadow-md);
  border: 1px solid var(--border-color);
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

.team-color-small {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 1px solid currentColor;
}

.action-buttons {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.add-player-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background-color: var(--secondary-color);
  color: white;
  border: none;
  padding: 0.75rem 1.5rem;
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-weight: 500;
}

.add-player-btn:hover {
  background-color: #2563eb;
  transform: translateY(-1px);
}

.add-player-btn svg {
  width: 18px;
  height: 18px;
}

.save-button {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  background-color: var(--primary-color);
  color: white;
  border: none;
  padding: 0.75rem 1.5rem;
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-weight: 500;
  white-space: nowrap;
}

.save-button:hover:not(:disabled) {
  background-color: var(--primary-hover);
  transform: translateY(-1px);
}

.save-button svg {
  width: 18px;
  height: 18px;
}

.save-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top: 2px solid white;
  border-radius: 50%;
  animation: spin 1s linear infinite;
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

.player-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background-color: var(--primary-color);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 1.1rem;
}

.player-info {
  flex: 1;
}

.player-name {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 0.25rem;
}

.player-goals {
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.remove-player {
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

.remove-player:hover {
  background-color: var(--danger-hover);
}

.remove-player svg {
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

/* Modal */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background-color: var(--bg-primary);
  border-radius: var(--border-radius-lg);
  padding: 0;
  max-width: 500px;
  width: 90%;
  max-height: 90vh;
  overflow: hidden;
  box-shadow: var(--shadow-lg);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h3 {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
}

.modal-close {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-secondary);
  padding: 0.25rem;
  border-radius: 50%;
  transition: all var(--transition-fast);
}

.modal-close:hover {
  background-color: var(--bg-tertiary);
}

.modal-close svg {
  width: 20px;
  height: 20px;
}

.modal-body {
  padding: 1.5rem;
}

.form-group {
  margin-bottom: 1.5rem;
  position: relative;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  color: var(--text-primary);
}

.form-group input {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
  font-size: 1rem;
  transition: border-color var(--transition-fast);
}

.form-group input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.modal-footer {
  display: flex;
  gap: 1rem;
  padding: 1.5rem;
  border-top: 1px solid var(--border-color);
  justify-content: flex-end;
}

.cancel-button {
  padding: 0.75rem 1.5rem;
  border: 1px solid var(--border-color);
  background-color: var(--bg-primary);
  color: var(--text-secondary);
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.cancel-button:hover {
  background-color: var(--bg-tertiary);
}

.confirm-button {
  padding: 0.75rem 1.5rem;
  background-color: var(--primary-color);
  color: white;
  border: none;
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.confirm-button:hover:not(:disabled) {
  background-color: var(--primary-hover);
}

.confirm-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  background-color: var(--text-light);
}

.input-wrapper {
  position: relative;
}

.input-wrapper input.error {
  border-color: var(--danger-color);
  background-color: #fef2f2;
}

.input-wrapper input.success {
  border-color: var(--primary-color);
  background-color: #f0fdf4;
}

.search-loading {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
}

.spinner-small {
  width: 16px;
  height: 16px;
  border: 2px solid var(--border-color);
  border-top: 2px solid var(--primary-color);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

/* Validation messages */
.validation-message {
  margin-top: 0.5rem;
  padding: 0.5rem;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
}

.validation-message.error {
  background-color: #fef2f2;
  color: #dc2626;
  border: 1px solid #fecaca;
}

.validation-message.success {
  background-color: #f0fdf4;
  color: #16a34a;
  border: 1px solid #bbf7d0;
}

.validation-message.info {
  background-color: #eff6ff;
  color: #2563eb;
  border: 1px solid #bfdbfe;
}

/* Suggestions dropdown */
.suggestions-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  background: white;
  border: 1px solid var(--border-color);
  border-top: none;
  border-radius: 0 0 var(--border-radius) var(--border-radius);
  box-shadow: var(--shadow-lg);
  z-index: 10;
  max-height: 200px;
  overflow-y: auto;
}

.suggestion-item {
  width: 100%;
  padding: 0.75rem;
  border: none;
  background: white;
  text-align: left;
  cursor: pointer;
  transition: background-color var(--transition-fast);
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.suggestion-item:hover {
  background-color: var(--bg-secondary);
}

.suggestion-item:last-child {
  border-bottom: none;
}

.suggestion-name {
  font-weight: 500;
  color: var(--text-tertiary);
}

.suggestion-item:hover .suggestion-name {
  color: var(--text-primary);
}

.suggestion-status {
  font-size: 0.75rem;
  color: var(--text-light);
  background-color: var(--bg-tertiary);
  padding: 0.25rem 0.5rem;
  border-radius: 12px;
}

.enhanced-multi-player-modal {
  max-width: 900px;
  width: 95%;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
}

.enhanced-multi-player-modal .modal-header {
  flex-shrink: 0;
  background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
  color: white;
  border-radius: var(--border-radius-lg) var(--border-radius-lg) 0 0;
  border-bottom: none;
}

.enhanced-multi-player-modal .modal-header h3 {
  color: white;
  font-size: 1.25rem;
  font-weight: 600;
}

.enhanced-multi-player-modal .modal-close {
  color: rgba(255, 255, 255, 0.8);
}

.enhanced-multi-player-modal .modal-close:hover {
  background-color: rgba(255, 255, 255, 0.1);
  color: white;
}

.enhanced-modal-body {
  flex: 1;
  padding: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
  /* Important for proper scrolling */
}

/* Search Section - Fixed at top */
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

.search-section input {
  font-size: 1rem;
  padding: 0.875rem 1rem;
  border: 2px solid var(--border-color);
  border-radius: var(--border-radius-lg);
  transition: all var(--transition-fast);
}

.search-section input:focus {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
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

.column-header {
  flex-shrink: 0;
  padding: 1rem 1.5rem;
  background: linear-gradient(135deg, var(--bg-tertiary), var(--bg-secondary));
  border-bottom: 1px solid var(--border-color);
}

.column-header h4 {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 0.5rem;
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
  font-weight: 700;
  padding: 0.25rem 0.5rem;
  border-radius: 12px;
  min-width: 20px;
  text-align: center;
}

.column-content {
  flex: 1;
  padding: 1rem 1.5rem;
  overflow-y: auto;
  min-height: 0;
}

/* Selected Players Column */
.selected-column {
  background-color: var(--bg-primary);
  border-right: 1px solid var(--border-color);
}

.selected-column .column-header {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.1), rgba(16, 185, 129, 0.05));
  border-bottom-color: rgba(16, 185, 129, 0.2);
}

/* Available Players Column */
.available-column {
  background-color: var(--bg-primary);
}

.available-column .column-header {
  background: linear-gradient(135deg, var(--bg-tertiary), var(--bg-secondary));
}

/* Empty States */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem 1rem;
  text-align: center;
  color: var(--text-secondary);
  height: 200px;
}

.empty-icon {
  width: 48px;
  height: 48px;
  margin-bottom: 1rem;
  color: var(--text-light);
}

.empty-icon svg {
  width: 100%;
  height: 100%;
}

.empty-state p {
  margin: 0 0 0.5rem 0;
  font-weight: 500;
  color: var(--text-secondary);
  font-size: 1rem;
}

.empty-state span {
  font-size: 0.875rem;
  color: var(--text-light);
  font-style: italic;
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
  padding: 0.875rem;
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.1), rgba(16, 185, 129, 0.05));
  border: 2px solid rgba(16, 185, 129, 0.2);
  border-radius: var(--border-radius-lg);
  transition: all var(--transition-fast);
  animation: slideInFromRight 0.3s ease-out;
}

.selected-player-item:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.15);
  border-color: rgba(16, 185, 129, 0.3);
}

@keyframes slideInFromRight {
  from {
    opacity: 0;
    transform: translateX(20px);
  }

  to {
    opacity: 1;
    transform: translateX(0);
  }
}

.selected-player-item .player-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex: 1;
}

.selected-player-item .player-name {
  font-weight: 600;
  color: var(--text-primary);
}

.selected-player-item .player-avatar-small {
  background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
  box-shadow: 0 2px 4px rgba(16, 185, 129, 0.2);
}

.remove-selected-btn {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: none;
  background-color: var(--danger-color);
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
  box-shadow: 0 2px 4px rgba(239, 68, 68, 0.2);
}

.remove-selected-btn:hover {
  background-color: var(--danger-hover);
  transform: scale(1.1);
  box-shadow: 0 4px 8px rgba(239, 68, 68, 0.3);
}

.remove-selected-btn svg {
  width: 14px;
  height: 14px;
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
  padding: 0.875rem;
  background-color: var(--bg-secondary);
  border: 2px solid transparent;
  border-radius: var(--border-radius-lg);
  cursor: pointer;
  transition: all var(--transition-fast);
  text-align: left;
  width: 100%;
}

.available-player-item:hover:not(:disabled) {
  background-color: var(--bg-tertiary);
  border-color: var(--primary-color);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.1);
}

.available-player-item:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  background-color: var(--bg-tertiary);
  border-color: var(--border-color);
}

.available-player-item .player-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.available-player-item .player-name {
  font-weight: 500;
  color: var(--text-primary);
}

.available-player-item:disabled .player-name {
  color: var(--text-secondary);
}

.player-avatar-small {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background-color: var(--primary-color);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 0.875rem;
  transition: all var(--transition-fast);
}

/* Player Status Icons */
.player-status {
  display: flex;
  align-items: center;
}

.status-selected,
.status-in-match,
.status-available {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  transition: all var(--transition-fast);
}

.status-selected {
  color: var(--primary-color);
  background-color: rgba(16, 185, 129, 0.1);
}

.status-selected svg {
  width: 16px;
  height: 16px;
}

.status-in-match {
  color: var(--text-secondary);
  background-color: var(--bg-tertiary);
}

.status-in-match svg {
  width: 14px;
  height: 14px;
}

.status-available {
  color: var(--primary-color);
  background-color: rgba(16, 185, 129, 0.1);
}

.status-available svg {
  width: 16px;
  height: 16px;
}

/* Enhanced Footer */
.enhanced-footer {
  flex-shrink: 0;
  background: linear-gradient(135deg, var(--bg-secondary), var(--bg-tertiary));
  border-top: 2px solid var(--border-color);
  padding: 1.5rem;
}

.selection-summary {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.summary-icon {
  width: 20px;
  height: 20px;
  color: var(--primary-color);
}

.selection-count {
  font-weight: 600;
  color: var(--text-primary);
}

.selection-count.empty {
  color: var(--text-secondary);
}

.footer-buttons {
  gap: 1rem;
}

.enhanced-footer .cancel-button,
.enhanced-footer .confirm-button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.875rem 1.5rem;
  font-weight: 600;
  border-radius: var(--border-radius-lg);
  transition: all var(--transition-fast);
}

.enhanced-footer .cancel-button {
  border: 2px solid var(--border-color);
  background-color: var(--bg-primary);
  color: var(--text-secondary);
}

.enhanced-footer .cancel-button:hover {
  background-color: var(--bg-tertiary);
  border-color: var(--text-secondary);
  color: var(--text-primary);
  transform: translateY(-1px);
}

.enhanced-footer .cancel-button svg {
  width: 16px;
  height: 16px;
}

.enhanced-confirm {
  background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
  border: none;
  color: white;
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.3);
}

.enhanced-confirm:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(16, 185, 129, 0.4);
}

.enhanced-confirm:disabled {
  background: var(--text-light);
  box-shadow: none;
  cursor: not-allowed;
  opacity: 0.6;
}

.enhanced-confirm svg {
  width: 18px;
  height: 18px;
}

/* Custom Scrollbars */
.column-content::-webkit-scrollbar {
  width: 8px;
}

.column-content::-webkit-scrollbar-track {
  background: var(--bg-tertiary);
  border-radius: 4px;
}

.column-content::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 4px;
  transition: background var(--transition-fast);
}

.column-content::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}

/* Create Player Section */
.create-player-section {
  margin-top: 1rem;
  padding: 1rem;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.05), rgba(37, 99, 235, 0.05));
  border: 2px dashed rgba(59, 130, 246, 0.3);
  border-radius: var(--border-radius-lg);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.create-player-prompt {
  display: flex;
  align-items: center;
  gap: 0.5rem;
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
  background: linear-gradient(135deg, var(--secondary-color), #1d4ed8);
  color: white;
  border: none;
  padding: 0.75rem 1rem;
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-weight: 500;
  align-self: flex-start;
}

.create-player-btn:hover:not(:disabled) {
  background: linear-gradient(135deg, #1d4ed8, #1e40af);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.create-player-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.create-player-btn svg {
  width: 16px;
  height: 16px;
}

/* Responsive Design */
@media (max-width: 768px) {
  .enhanced-multi-player-modal {
    width: 98%;
    max-height: 90vh;
  }

  .players-columns {
    grid-template-columns: 1fr;
    grid-template-rows: 300px 1fr;
  }

  .selected-column {
    border-right: none;
    border-bottom: 2px solid var(--border-color);
  }

  .column-content {
    padding: 1rem;
  }

  .enhanced-footer {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }

  .footer-buttons {
    justify-content: center;
  }

  .create-player-section {
    padding: 0.75rem;
  }

  .create-player-btn {
    align-self: stretch;
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .players-columns {
    grid-template-rows: 250px 1fr;
  }

  .column-header {
    padding: 0.75rem 1rem;
  }

  .column-content {
    padding: 0.75rem;
  }

  .selected-player-item,
  .available-player-item {
    padding: 0.75rem;
  }

  .player-avatar-small {
    width: 32px;
    height: 32px;
    font-size: 0.75rem;
  }
}

/* Selected Players Section */
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
  background-color: var(--bg-primary);
  border-radius: var(--border-radius);
  border: 1px solid var(--border-color);
}

.selected-player-item .player-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex: 1;
}

.player-avatar-small {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background-color: var(--primary-color);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 0.75rem;
}

.selected-player-item .player-name {
  font-weight: 500;
  color: var(--text-primary);
}

.player-controls {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.remove-selected-btn {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: none;
  background-color: var(--danger-color);
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
}

.remove-selected-btn:hover {
  background-color: var(--danger-hover);
  transform: scale(1.1);
}

.remove-selected-btn svg {
  width: 12px;
  height: 12px;
}

/* Available Players Section */
.available-players-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-height: 300px;
  overflow-y: auto;
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
}

.available-player-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem;
  background-color: var(--bg-primary);
  border: none;
  border-bottom: 1px solid var(--border-color);
  cursor: pointer;
  transition: all var(--transition-fast);
  text-align: left;
  width: 100%;
}

.available-player-item:hover:not(:disabled) {
  background-color: var(--bg-secondary);
}

.available-player-item:last-child {
  border-bottom: none;
}

.available-player-item:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  background-color: var(--bg-tertiary);
}

.available-player-item .player-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.available-player-item .player-name {
  font-weight: 500;
  color: var(--text-primary);
}

.available-player-item:disabled .player-name {
  color: var(--text-secondary);
}

.player-status {
  display: flex;
  align-items: center;
}

.status-selected {
  font-size: 0.75rem;
  color: var(--primary-color);
  font-weight: 600;
  background-color: rgba(16, 185, 129, 0.1);
  padding: 0.25rem 0.5rem;
  border-radius: 12px;
}

.status-in-match {
  font-size: 0.75rem;
  color: var(--text-secondary);
  background-color: var(--bg-tertiary);
  padding: 0.25rem 0.5rem;
  border-radius: 12px;
}

.status-available {
  color: var(--primary-color);
}

.status-available svg {
  width: 16px;
  height: 16px;
}

/* Updated Modal Footer */
.modal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.5rem;
  border-top: 1px solid var(--border-color);
  background-color: var(--bg-secondary);
}

.footer-info {
  flex: 1;
}

.selection-count {
  font-size: 0.875rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.footer-buttons {
  display: flex;
  gap: 1rem;
}

/* Responsive adjustments for multi-player modal */
@media (max-width: 768px) {
  .selected-player-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.75rem;
  }

  .player-controls {
    width: 100%;
    justify-content: space-between;
  }

  .available-players-list {
    max-height: 250px;
  }

  .modal-footer {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }

  .footer-buttons {
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .available-player-item {
    padding: 0.5rem;
  }
}

/* Scrollbar styling for lists */
.available-players-list::-webkit-scrollbar {
  width: 6px;
}

.available-players-list::-webkit-scrollbar-track {
  background: var(--bg-tertiary);
  border-radius: 3px;
}

.available-players-list::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.available-players-list::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}

/* Toast Container */
.toast-container {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 1001;
  pointer-events: none;
}

/* Modern Toast Message */
.message {
  position: relative;
  display: flex;
  align-items: center;
  gap: 20px;
  min-width: 300px;
  max-width: 420px;
  padding: 16px 20px;
  margin-bottom: 12px;
  border-radius: 12px;
  font-weight: 500;
  font-size: 0.95rem;
  line-height: 1.4;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.2);
  pointer-events: auto;

  /* Animation */
  animation: toastSlideIn 0.4s cubic-bezier(0.16, 1, 0.3, 1);
  transform-origin: top right;
}

/* Success Toast */
.message.success {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.95), rgba(5, 150, 105, 0.95));
  color: white;
  border-color: rgba(255, 255, 255, 0.3);
}

/* Error Toast */
.message.error {
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.95), rgba(220, 38, 38, 0.95));
  color: white;
  border-color: rgba(255, 255, 255, 0.3);
}

/* Info Toast */
.message.info {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.95), rgba(37, 99, 235, 0.95));
  color: white;
  border-color: rgba(255, 255, 255, 0.3);
}

/* Warning Toast */
.message.warning {
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.95), rgba(217, 119, 6, 0.95));
  color: white;
  border-color: rgba(255, 255, 255, 0.3);
}

/* Close Button */
.message::after {
  content: '×';
  position: absolute;
  top: 8px;
  right: 10px;
  width: 20px;
  height: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.7);
  font-size: 18px;
  font-weight: bold;
  cursor: pointer;
  border-radius: 50%;
  transition: all 0.2s ease;
}

.message::after:hover {
  background: rgba(255, 255, 255, 0.2);
  color: white;
}

/* Animations */
@keyframes toastSlideIn {
  from {
    opacity: 0;
    transform: translateX(100%) scale(0.8);
  }

  to {
    opacity: 1;
    transform: translateX(0) scale(1);
  }
}

@keyframes toastSlideOut {
  from {
    opacity: 1;
    transform: translateX(0) scale(1);
  }

  to {
    opacity: 0;
    transform: translateX(100%) scale(0.8);
  }
}

@keyframes toastProgress {
  from {
    width: 100%;
  }

  to {
    width: 0%;
  }
}

/* Exit Animation */
.message.toast-exit {
  animation: toastSlideOut 0.3s cubic-bezier(0.4, 0, 1, 1) forwards;
}

/* Responsive */
@media (max-width: 480px) {
  .toast-container {
    top: 10px;
    right: 10px;
    left: 10px;
  }

  .message {
    min-width: unset;
    max-width: unset;
    width: 100%;
    margin-bottom: 8px;
    font-size: 0.9rem;
  }
}

/* Dark mode support */
@media (prefers-color-scheme: dark) {
  .message {
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  }
}

/* Error State */
.error-state {
  padding: 4rem 0;
  min-height: 60vh;
  display: flex;
  align-items: center;
  justify-content: center;
}

.error-content {
  text-align: center;
  max-width: 400px;
  margin: 0 auto;
}

.error-icon {
  width: 4rem;
  height: 4rem;
  color: var(--danger-color);
  margin-bottom: 1rem;
}

.error-title {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 0.5rem;
}

.error-description {
  color: var(--text-secondary);
  margin-bottom: 2rem;
}

.error-back-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  background-color: var(--primary-color);
  color: white;
  border: none;
  padding: 0.75rem 1.5rem;
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-weight: 500;
}

.error-back-btn:hover {
  background-color: var(--primary-hover);
  transform: translateY(-1px);
}

.error-back-btn svg {
  width: 16px;
  height: 16px;
}

@keyframes slideIn {
  from {
    transform: translateX(100%);
    opacity: 0;
  }

  to {
    transform: translateX(0);
    opacity: 1;
  }
}

/* Responsive Design */
@media (max-width: 768px) {
  .container {
    padding: 0 1rem;
  }

  .back-button {
    position: static;
    transform: none;
    margin-bottom: 1rem;
  }

  .header-content {
    text-align: left;
  }

  .page-title {
    font-size: 2rem;
  }

  .match-title {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
    text-align: center;
  }

  .teams-score {
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  .vs-divider {
    transform: rotate(90deg);
  }

  .vs-circle {
    width: 40px;
    height: 40px;
    font-size: 0.875rem;
  }

  .score {
    font-size: 2rem;
  }

  .management-header {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }

  .tabs-buttons {
    flex-direction: column;
    gap: 0.5rem;
  }

  .tab-button {
    justify-content: center;
  }

  .action-buttons {
    justify-content: center;
    flex-wrap: wrap;
  }

  .players-grid {
    grid-template-columns: 1fr;
  }

  .modal-content {
    width: 95%;
  }
}

@media (max-width: 480px) {
  .score-overview {
    padding: 1rem;
  }

  .team-score {
    padding: 0.75rem;
  }

  .score {
    font-size: 1.75rem;
  }

  .team-color {
    width: 24px;
    height: 24px;
  }

  .team-name {
    font-size: 1rem;
  }

  .vs-circle {
    width: 32px;
    height: 32px;
    font-size: 0.75rem;
  }

  .tab-button {
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
  }

  .add-player-btn,
  .save-button {
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
  }
}
</style>
