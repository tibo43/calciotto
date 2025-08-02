<template>
  <div class="match-detail-container">
    <!-- Header Section -->
    <section class="match-header">
      <div class="container">
        <div class="header-content">
          <button @click="goBack" class="back-button">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15,18 9,12 15,6"/>
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

        <!-- Team Management Tabs -->
        <div v-if="match.Teams && match.Teams.length > 0" class="team-tabs">
          <div class="tabs-buttons">
            <button 
              v-for="(team, index) in match.Teams" 
              :key="team.ID"
              @click="activeTeam = index"
              :class="['tab-button', { active: activeTeam === index }]"
            >
              <div class="team-color-small" :style="{ backgroundColor: getTeamColor(team.Colour) }"></div>
              {{ team.Colour }} Team ({{ team.Players ? team.Players.length : 0 }})
            </button>
          </div>
          
          <!-- Save Changes Button -->
          <button @click="saveChanges" :disabled="isSaving" class="save-button">
            <svg v-if="!isSaving" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/>
              <polyline points="17,21 17,13 7,13 7,21"/>
              <polyline points="7,3 7,8 15,8"/>
            </svg>
            <div v-else class="save-spinner"></div>
            {{ isSaving ? 'Saving...' : 'Save Changes' }}
          </button>
        </div>

        <!-- Active Team Management -->
        <div v-if="match.Teams && match.Teams[activeTeam]" class="team-management">
          <div class="management-header">
            <h3>{{ match.Teams[activeTeam].Colour }} Team Management</h3>
            <button @click="showAddPlayerModal = true" class="add-player-btn">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <line x1="12" y1="8" x2="12" y2="16"/>
                <line x1="8" y1="12" x2="16" y2="12"/>
              </svg>
              Add Player
            </button>
          </div>

          <!-- Players List -->
          <div v-if="match.Teams[activeTeam].Players" class="players-grid">
            <div 
              v-for="(player, playerIndex) in match.Teams[activeTeam].Players" 
              :key="playerIndex"
              class="player-card"
            >
              <div class="player-header">
                <div class="player-avatar">
                  {{ getPlayerInitials(player.Name) }}
                </div>
                <div class="player-info">
                  <h4 class="player-name">{{ player.Name }}</h4>
                  <span class="player-goals">Goals: {{ player.GoalNumber || 0 }}</span>
                </div>
                <button @click="removePlayer(playerIndex)" class="remove-player">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="18" y1="6" x2="6" y2="18"/>
                    <line x1="6" y1="6" x2="18" y2="18"/>
                  </svg>
                </button>
              </div>
              
              <!-- Goal Management -->
              <div class="goal-management">
                <button 
                  @click="updateGoals(playerIndex, -1)" 
                  :disabled="!player.GoalNumber || player.GoalNumber <= 0"
                  class="goal-btn decrease"
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="5" y1="12" x2="19" y2="12"/>
                  </svg>
                </button>
                
                <span class="goal-count">{{ player.GoalNumber || 0 }}</span>
                
                <button @click="updateGoals(playerIndex, 1)" class="goal-btn increase">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="12" y1="5" x2="12" y2="19"/>
                    <line x1="5" y1="12" x2="19" y2="12"/>
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Save Changes Button moved to team-tabs section -->
      </div>
    </div>

    <!-- Error State -->
    <div v-else class="error-state">
      <div class="error-content">
        <svg class="error-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="15" y1="9" x2="9" y2="15"/>
          <line x1="9" y1="9" x2="15" y2="15"/>
        </svg>
        <h3 class="error-title">Match not found</h3>
        <p class="error-description">The match you're looking for doesn't exist or failed to load.</p>
        <button @click="goBack" class="error-back-btn">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="15,18 9,12 15,6"/>
          </svg>
          Go Back
        </button>
      </div>
    </div>

    <!-- Add Player Modal -->
    <div v-if="showAddPlayerModal" class="modal-overlay" @click="closeModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>Add New Player</h3>
          <button @click="closeModal" class="modal-close">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>
        
        <div class="modal-body">
          <div class="form-group">
            <label for="playerName">Player Name</label>
            <input 
              v-model="newPlayerName" 
              type="text" 
              id="playerName"
              placeholder="Enter player name"
              @keyup.enter="addPlayer"
            >
          </div>
          
          <div class="form-group">
            <label for="playerGoals">Initial Goals</label>
            <input 
              v-model.number="newPlayerGoals" 
              type="number" 
              id="playerGoals"
              min="0"
              placeholder="0"
            >
          </div>
        </div>
        
        <div class="modal-footer">
          <button @click="closeModal" class="cancel-button">Cancel</button>
          <button @click="addPlayer" :disabled="!newPlayerName.trim()" class="confirm-button">
            Add Player
          </button>
        </div>
      </div>
    </div>

    <!-- Error/Success Messages -->
    <div v-if="message" :class="['message', messageType]">
      {{ message }}
    </div>
  </div>
</template>

<script>
import { getMatchDetailsByID, updateMatch } from '@/services/api';

export default {
  name: 'MatchDetail',
  data() {
    return {
      match: null,
      isLoading: true,
      isSaving: false,
      activeTeam: 0,
      showAddPlayerModal: false,
      newPlayerName: '',
      newPlayerGoals: 0,
      message: '',
      messageType: 'success' // 'success' or 'error'
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
        // You'll need to implement getMatchDetailsByID in your API service
        this.match = await getMatchDetailsByID(matchId);
        
        // Ensure each player has GoalNumber property
        this.match.Teams.forEach(team => {
          team.Players.forEach(player => {
            if (!player.GoalNumber) {
              player.GoalNumber = 0;
            }
          });
        });
        
      } catch (error) {
        console.error('Error fetching match:', error);
        this.showMessage('Error loading match details', 'error');
      } finally {
        this.isLoading = false;
      }
    },
    
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
      
      // Remove player without confirmation popup
      this.match.Teams[this.activeTeam].Players.splice(playerIndex, 1);
      this.updateTeamScore();
    },
    
    addPlayer() {
      if (!this.newPlayerName.trim()) return;
      
      if (!this.match.Teams || !this.match.Teams[this.activeTeam]) {
        console.error('Invalid team data');
        return;
      }
      
      if (!this.match.Teams[this.activeTeam].Players) {
        this.match.Teams[this.activeTeam].Players = [];
      }
      
      const newPlayer = {
        Name: this.newPlayerName.trim(),
        GoalNumber: this.newPlayerGoals || 0
      };
      
      this.match.Teams[this.activeTeam].Players.push(newPlayer);
      this.updateTeamScore();
      this.closeModal();
    },
    
    closeModal() {
      this.showAddPlayerModal = false;
      this.newPlayerName = '';
      this.newPlayerGoals = 0;
    },
    
    async saveChanges() {
      this.isSaving = true;
      try {
        // You'll need to implement updateMatch in your API service
        await updateMatch(this.match.ID, this.match);
        this.showMessage('Match updated successfully!', 'success');
      } catch (error) {
        console.error('Error saving match:', error);
        this.showMessage('Error saving changes', 'error');
      } finally {
        this.isSaving = false;
      }
    },
    
    showMessage(text, type) {
      this.message = text;
      this.messageType = type;
      setTimeout(() => {
        this.message = '';
      }, 3000);
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
      return name.split(' ').map(word => word.charAt(0)).join('').toUpperCase().slice(0, 2);
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
/* CSS Variables (same as MatchesAll.vue) */
:root {
  --primary-color: #10b981;
  --primary-hover: #059669;
  --primary-light: #6ee7b7;
  --secondary-color: #3b82f6;
  --accent-color: #f59e0b;
  --danger-color: #ef4444;
  --danger-hover: #dc2626;
  --text-primary: #1f2937;
  --text-secondary: #6b7280;
  --text-light: #9ca3af;
  --bg-primary: #ffffff;
  --bg-secondary: #f8fafc;
  --bg-tertiary: #f1f5f9;
  --border-color: #e2e8f0;
  --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.05);
  --shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1);
  --shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.1);
  --transition-fast: 0.2s ease;
  --transition-smooth: 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  --border-radius: 12px;
  --border-radius-lg: 16px;
}

.match-detail-container {
  min-height: 100vh;
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
  padding: 2rem 0;
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

.page-title {
  font-size: 2.5rem;
  font-weight: 800;
  margin-bottom: 0.5rem;
}

.page-subtitle {
  font-size: 1.1rem;
  opacity: 0.9;
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
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
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

/* Team Tabs */
.team-tabs {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  margin-bottom: 2rem;
  padding: 1rem;
  background-color: var(--bg-primary);
  border-radius: var(--border-radius-lg);
  box-shadow: var(--shadow-sm);
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
  background-color: var(--bg-tertiary);
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
  margin-bottom: 2rem;
}

.management-header h3 {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-primary);
}

.add-player-btn {
  display: flex;
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

.add-player-btn:hover {
  background-color: var(--primary-hover);
  transform: translateY(-1px);
}

.add-player-btn svg {
  width: 18px;
  height: 18px;
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

/* Save Section - now in team-tabs */
.save-button {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  background-color: var(--primary-color);
  color: white;
  border: none;
  padding: 1rem 2rem;
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-weight: 600;
  font-size: 1rem;
  white-space: nowrap;
}

.save-button:hover:not(:disabled) {
  background-color: var(--primary-hover);
  transform: translateY(-1px);
}

.save-button:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.save-button svg {
  width: 20px;
  height: 20px;
}

.save-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top: 2px solid white;
  border-radius: 50%;
  animation: spin 1s linear infinite;
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
  box-shadow: var(--shadow-xl);
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
}

/* Messages */
.message {
  position: fixed;
  top: 20px;
  right: 20px;
  padding: 1rem 1.5rem;
  border-radius: var(--border-radius);
  color: white;
  font-weight: 500;
  z-index: 1001;
  animation: slideIn 0.3s ease-out;
}

.message.success {
  background-color: var(--primary-color);
}

.message.error {
  background-color: var(--danger-color);
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

  .team-tabs {
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

  .save-button {
    justify-content: center;
    padding: 1rem;
  }

  .management-header {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
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

  .save-button {
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
  }
}
</style>
