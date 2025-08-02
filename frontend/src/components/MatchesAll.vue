<template>
  <div class="matches-container">
    <!-- Header Section -->
    <section class="matches-header">
      <div class="container">
        <div class="header-content">
          <h1 class="page-title">Football Matches</h1>
          <p class="page-subtitle">Track live scores and match details</p>
        </div>
      </div>
    </section>

    <!-- Matches Section -->
    <section class="matches-section">
      <div class="container">
        <!-- Loading State -->
        <div v-if="isLoading" class="loading-container">
          <div class="loading-spinner"></div>
          <p class="loading-text">Loading matches...</p>
        </div>

        <!-- Matches Layout -->
        <div v-else-if="matches.length > 0" class="matches-layout">
          <!-- Horizontal Matches Bar -->
          <div class="matches-bar-container">
            <div class="matches-bar" ref="matchesBar">
              <div 
                v-for="match in matches" 
                :key="match.ID" 
                class="match-card-horizontal"
                :class="{ 'active': selectedMatch?.ID === match.ID }"
                @click="selectMatch(match)"
              >
                <!-- Match Date -->
                <div class="match-date-horizontal">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
                    <line x1="16" y1="2" x2="16" y2="6"/>
                    <line x1="8" y1="2" x2="8" y2="6"/>
                    <line x1="3" y1="10" x2="21" y2="10"/>
                  </svg>
                  <span>{{ formatDateShort(match.Date) }}</span>
                </div>

                <!-- Teams and Scores -->
                <div class="teams-horizontal">
                  <div v-for="(team, index) in match.Teams" :key="team.ID" class="team-horizontal">
                    <div class="team-info-horizontal">
                      <div class="team-color-horizontal" :style="{ backgroundColor: getTeamColor(team.Colour) }"></div>
                      <span class="team-name-horizontal">{{ team.Colour }}</span>
                    </div>
                    <div class="team-score-horizontal">{{ team.Score }}</div>
                    <div v-if="index < match.Teams.length - 1" class="vs-separator">vs</div>
                  </div>
                </div>

                <!-- Match Status -->
                <div class="match-status-horizontal">
                  <div class="status-indicator-horizontal" :class="getMatchStatus(match)"></div>
                </div>
              </div>
            </div>
            
            <!-- Scroll indicators -->
            <button 
              class="scroll-btn scroll-left" 
              @click="scrollLeft"
              :disabled="!canScrollLeft"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="15,18 9,12 15,6"/>
              </svg>
            </button>
            <button 
              class="scroll-btn scroll-right" 
              @click="scrollRight"
              :disabled="!canScrollRight"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="9,18 15,12 9,6"/>
              </svg>
            </button>
          </div>

          <!-- Selected Match Details -->
          <transition name="match-details">
            <div v-if="selectedMatch" class="match-details-container">
              <div class="details-header">
                <div class="details-title-section">
                  <h3>{{ formatDate(selectedMatch.Date) }} - Match Details</h3>
                  <router-link 
                    :to="`/matches/${selectedMatch.ID}/edit`" 
                    class="edit-match-btn"
                  >
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                    </svg>
                    Edit Match
                  </router-link>
                </div>
                <div class="details-divider"></div>
              </div>
              
              <div class="players-section">
                <div class="players-table-container">
                  <table class="players-table">
                    <thead>
                      <tr>
                        <th class="goal-number-col">Goal #</th>
                        <th v-for="team in selectedMatch.Teams" :key="team.ID" class="team-col">
                          <div class="team-header">
                            <div class="team-color-small" :style="{ backgroundColor: getTeamColor(team.Colour) }"></div>
                            {{ team.Colour }}
                          </div>
                        </th>
                        <th class="goal-number-col">Goal #</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="rowIndex in getMaxPlayers(selectedMatch.Teams)" :key="rowIndex" class="player-row">
                        <td class="goal-cell">
                          {{ selectedMatch.Teams[0].Players[rowIndex - 1]?.GoalNumber || '-' }}
                        </td>
                        <td v-for="team in selectedMatch.Teams" :key="team.ID" class="player-cell">
                          <div v-if="team.Players[rowIndex - 1]" class="player-info">
                            <div class="player-avatar">
                              {{ getPlayerInitials(team.Players[rowIndex - 1].Name) }}
                            </div>
                            <span class="player-name">{{ team.Players[rowIndex - 1].Name }}</span>
                          </div>
                          <span v-else class="empty-slot">-</span>
                        </td>
                        <td class="goal-cell">
                          {{ selectedMatch.Teams[1]?.Players[rowIndex - 1]?.GoalNumber || '-' }}
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </transition>
        </div>

        <!-- Empty State -->
        <div v-else class="empty-state">
          <div class="empty-content">
            <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <path d="M16 16s-1.5-2-4-2-4 2-4 2"/>
              <line x1="9" y1="9" x2="9.01" y2="9"/>
              <line x1="15" y1="9" x2="15.01" y2="9"/>
            </svg>
            <h3 class="empty-title">No matches available</h3>
            <p class="empty-description">Check back later for upcoming matches</p>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
import { getMatchesDetails } from '@/services/api';

export default {
  name: 'MatchesAll',
  data() {
    return {
      matches: [],
      selectedMatch: null,
      isLoading: true,
      canScrollLeft: false,
      canScrollRight: false
    };
  },
  async created() {
    await this.loadMatches();
  },
  mounted() {
    this.$nextTick(() => {
      this.updateScrollButtons();
      if (this.$refs.matchesBar) {
        this.$refs.matchesBar.addEventListener('scroll', this.updateScrollButtons);
      }
    });
  },
  beforeUnmount() {
    if (this.$refs.matchesBar) {
      this.$refs.matchesBar.removeEventListener('scroll', this.updateScrollButtons);
    }
  },
  methods: {
    async loadMatches() {
      this.isLoading = true;
      try {
        const matches = await getMatchesDetails();
        console.log("matches", matches);
        
        // Validate matches data
        if (!Array.isArray(matches)) {
          throw new Error('Invalid matches data');
        }
        
        matches.forEach(match => {
          if (!match.ID || !match.Date || !Array.isArray(match.Teams)) {
            throw new Error('Invalid match structure');
          }
          match.Teams.forEach(team => {
            if (!team.ID || !team.Colour || !Array.isArray(team.Players)) {
              throw new Error('Invalid team structure');
            }
          });
        });

        this.matches = matches;
        
        // Auto-select the newest match
        if (this.matches.length > 0) {
          this.selectedMatch = this.matches[0];
        }
      } catch (error) {
        console.error('Error fetching matches:', error);
      } finally {
        this.isLoading = false;
      }
    },
    
    selectMatch(match) {
      this.selectedMatch = match;
    },
    
    scrollLeft() {
      if (this.$refs.matchesBar) {
        this.$refs.matchesBar.scrollBy({ left: -300, behavior: 'smooth' });
      }
    },
    
    scrollRight() {
      if (this.$refs.matchesBar) {
        this.$refs.matchesBar.scrollBy({ left: 300, behavior: 'smooth' });
      }
    },
    
    updateScrollButtons() {
      if (this.$refs.matchesBar) {
        const element = this.$refs.matchesBar;
        this.canScrollLeft = element.scrollLeft > 0;
        this.canScrollRight = element.scrollLeft < (element.scrollWidth - element.clientWidth);
      }
    },
    
    formatDate(dateString) {
      try {
        const date = new Date(dateString);
        return date.toLocaleDateString('en-US', {
          weekday: 'short',
          year: 'numeric',
          month: 'short',
          day: 'numeric'
        });
      } catch (error) {
        return dateString;
      }
    },
    
    formatDateShort(dateString) {
      try {
        const date = new Date(dateString);
        return date.toLocaleDateString('en-US', {
          year: 'numeric',
          month: 'short',
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
    
    getMaxPlayers(teams) {
      return Math.max(...teams.map(team => team.Players.length));
    },
    
    getPlayerInitials(name) {
      return name.split(' ').map(word => word.charAt(0)).join('').toUpperCase().slice(0, 2);
    },
    
    getTotalGoals(match) {
      return match.Teams.reduce((total, team) => total + (team.Score || 0), 0);
    },
    
    getTotalPlayers(match) {
      return match.Teams.reduce((total, team) => total + team.Players.length, 0);
    },
    
    getMatchStatus(match) {
      const totalGoals = this.getTotalGoals(match);
      if (totalGoals === 0) return 'upcoming';
      return 'completed';
    },
    
    getMatchStatusText(match) {
      const status = this.getMatchStatus(match);
      switch (status) {
        case 'upcoming': return 'Scheduled';
        case 'completed': return 'Completed';
        default: return 'Unknown';
      }
    }
  }
};
</script>

<style scoped>
/* CSS Variables */
:root {
  --primary-color: #10b981;
  --primary-hover: #059669;
  --primary-light: #6ee7b7;
  --secondary-color: #3b82f6;
  --accent-color: #f59e0b;
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
  --shadow-xl: 0 20px 25px -5px rgb(0 0 0 / 0.1);
  --transition-fast: 0.2s ease;
  --transition-smooth: 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  --border-radius: 12px;
  --border-radius-lg: 16px;
}

.dark-mode {
  --text-primary: #f9fafb;
  --text-secondary: #d1d5db;
  --text-light: #9ca3af;
  --bg-primary: #0f172a;
  --bg-secondary: #1e293b;
  --bg-tertiary: #334155;
  --border-color: #475569;
}

/* Container */
.matches-container {
  /* min-height: 100vh; */
  background-color: var(--bg-secondary);
}

.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1.5rem;
}

/* Header Section */
.matches-header {
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--secondary-color) 100%);
  color: white;
  padding: 1rem 0;
  position: relative;
  overflow: hidden;
}

.matches-header::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="2" fill="white" opacity="0.1"/></svg>') repeat;
  animation: float 20s ease-in-out infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0px); }
  50% { transform: translateY(-10px); }
}

.header-content {
  text-align: center;
  position: relative;
  z-index: 1;
}

.page-title {
  font-size: 2.5rem;
  font-weight: 800;
  margin-bottom: 0.5rem;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.page-subtitle {
  font-size: 1.1rem;
  opacity: 0.9;
}

/* Matches Section */
.matches-section {
  padding: 2rem 0;
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

/* Matches Layout */
.matches-layout {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  /* height: calc(100vh - 200px); */
}

/* Horizontal Matches Bar */
.matches-bar-container {
  position: relative;
  background-color: var(--bg-primary);
  border-radius: var(--border-radius-lg);
  box-shadow: var(--shadow-md);
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.matches-bar {
  display: flex;
  gap: 1rem;
  padding: 1rem;
  overflow-x: auto;
  scroll-behavior: smooth;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.matches-bar::-webkit-scrollbar {
  display: none;
}

.match-card-horizontal {
  flex: 0 0 280px;
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  padding: 1rem;
  cursor: pointer;
  transition: all var(--transition-smooth);
  border: 2px solid transparent;
}

.match-card-horizontal:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
  background-color: var(--bg-primary);
}

.match-card-horizontal.active {
  border-color: var(--primary-color);
  background-color: var(--bg-primary);
  box-shadow: var(--shadow-lg);
}

.match-date-horizontal {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--text-secondary);
  font-weight: 500;
  margin-bottom: 0.75rem;
  font-size: 0.875rem;
}

.match-date-horizontal svg {
  width: 14px;
  height: 14px;
}

.teams-horizontal {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.team-horizontal {
  display: flex;
  align-items: center;
  justify-content: space-between;
  position: relative;
}

.team-info-horizontal {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.team-color-horizontal {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 1px solid white;
  box-shadow: var(--shadow-sm);
}

.team-name-horizontal {
  font-weight: 600;
  color: var(--text-primary);
  text-transform: capitalize;
  font-size: 0.875rem;
}

.team-score-horizontal {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--primary-color);
  background-color: var(--bg-primary);
  padding: 0.125rem 0.5rem;
  border-radius: 6px;
  min-width: 30px;
  text-align: center;
}

.vs-separator {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  top: -0.25rem;
  font-size: 0.75rem;
  color: var(--text-light);
  font-weight: 500;
}

.match-status-horizontal {
  display: flex;
  justify-content: center;
}

.status-indicator-horizontal {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  animation: pulse-status 2s infinite;
}

.status-indicator-horizontal.upcoming {
  background-color: #f59e0b;
}

.status-indicator-horizontal.completed {
  background-color: var(--primary-color);
}

@keyframes pulse-status {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* Scroll Buttons */
.scroll-btn {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  background-color: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 50%;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
  color: var(--text-secondary);
  box-shadow: var(--shadow-md);
  z-index: 10;
}

.scroll-btn:hover:not(:disabled) {
  background-color: var(--primary-color);
  color: white;
  transform: translateY(-50%) scale(1.1);
}

.scroll-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.scroll-left {
  left: 10px;
}

.scroll-right {
  right: 10px;
}

.scroll-btn svg {
  width: 20px;
  height: 20px;
}

/* Match Details Container */
.match-details-container {
  background-color: var(--bg-primary);
  border-radius: var(--border-radius-lg);
  padding: 1.5rem;
  box-shadow: var(--shadow-md);
  border: 1px solid var(--border-color);
  overflow: hidden;
}

.details-header {
  margin-bottom: 1rem;
  flex-shrink: 0;
}

.details-header h3 {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 0.5rem;
}

.details-divider {
  height: 3px;
  background: linear-gradient(90deg, var(--primary-color), var(--secondary-color));
  border-radius: 2px;
  width: 80px;
}

.details-title-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.edit-match-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  background-color: var(--primary-color);
  color: white;
  text-decoration: none;
  padding: 0.75rem 1.5rem;
  cursor: pointer;
  border-radius: var(--border-radius);
  font-weight: 500;
  font-size: 0.875rem;
  transition: all var(--transition-fast);
  box-shadow: var(--shadow-sm);
}

.edit-match-btn:hover {
  background-color: var(--primary-hover);
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
  color: white;
  text-decoration: none;
}

.edit-match-btn svg {
  width: 16px;
  height: 16px;
}

/* Responsive adjustments */
@media (max-width: 768px) {
  .details-title-section {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }
  
  .edit-match-btn {
    justify-content: center;
  }
}

/* Players Table */
.players-section {
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.players-table-container {
  overflow: auto;
}

.players-table {
  width: 100%;
  border-collapse: collapse;
  background-color: var(--bg-primary);
  border-radius: var(--border-radius);
  overflow: hidden;
  box-shadow: var(--shadow-sm);
  height: fit-content;
}

.players-table th {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
  font-weight: 600;
  padding: 0.75rem;
  text-align: center;
  border-bottom: 2px solid var(--border-color);
  position: sticky;
  top: 0;
  z-index: 5;
}

.team-header {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

.team-color-small {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 1px solid white;
}

.goal-number-col {
  width: 80px;
  background-color: var(--primary-color);
  color: white;
}

.team-col {
  min-width: 200px;
}

.players-table td {
  padding: 0.5rem;
  text-align: center;
  border-bottom: 1px solid var(--border-color);
}

.player-row:hover {
  background-color: var(--bg-secondary);
}

.player-info {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
}

.player-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background-color: var(--primary-color);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  font-weight: 600;
}

.player-name {
  font-weight: 500;
  color: var(--text-primary);
}

.goal-cell {
  background-color: var(--bg-tertiary);
  font-weight: 600;
  color: var(--primary-color);
}

.empty-slot {
  color: var(--text-light);
  font-style: italic;
}



/* Empty State */
.empty-state {
  padding: 4rem 0;
}

.empty-content {
  text-align: center;
  max-width: 400px;
  margin: 0 auto;
}

.empty-icon {
  width: 4rem;
  height: 4rem;
  color: var(--text-light);
  margin-bottom: 1rem;
}

.empty-title {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 0.5rem;
}

.empty-description {
  color: var(--text-secondary);
}

/* Transitions */
.match-details-enter-active,
.match-details-leave-active {
  transition: all var(--transition-smooth);
}

.match-details-enter-from,
.match-details-leave-to {
  opacity: 0;
  transform: translateY(20px);
}

/* Responsive Design */
@media (max-width: 768px) {
  .page-title {
    font-size: 2rem;
  }

  .container {
    padding: 0 1rem;
  }

  .matches-section {
    padding: 2rem 0;
  }

  .matches-header {
    padding: 2rem 0;
  }

  .match-card-horizontal {
    flex: 0 0 240px;
    padding: 0.75rem;
  }

  .match-details-container {
    padding: 1rem;
    height: calc(100vh - 320px);
  }

  .players-table {
    font-size: 0.875rem;
  }

  .players-table th,
  .players-table td {
    padding: 0.4rem;
  }

  .scroll-btn {
    width: 36px;
    height: 36px;
  }

  .scroll-btn svg {
    width: 18px;
    height: 18px;
  }
}

@media (max-width: 480px) {
  .page-title {
    font-size: 1.75rem;
  }

  .match-card-horizontal {
    flex: 0 0 200px;
    padding: 0.5rem;
  }

  .match-details-container {
    padding: 0.75rem;
    height: calc(100vh - 280px);
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }

  .scroll-left {
    left: 5px;
  }

  .scroll-right {
    right: 5px;
  }
}
</style>
