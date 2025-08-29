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

    <!-- Toast Container -->
    <div v-if="message" class="toast-container">
      <div :class="['toast-message', messageType]" @click="dismissMessage" :key="messageKey">
        {{ message }}
      </div>
    </div>
  </div>
</template>

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