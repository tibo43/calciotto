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

    <!-- Matches Grid -->
    <section class="matches-section">
      <div class="container">
        <!-- Loading State -->
        <div v-if="isLoading" class="loading-container">
          <div class="loading-spinner"></div>
          <p class="loading-text">Loading matches...</p>
        </div>

        <!-- Matches List -->
        <div v-else-if="matches.length > 0" class="matches-grid">
          <transition-group name="match-item" tag="div" class="matches-wrapper">
            <div 
              v-for="match in matches" 
              :key="match.ID" 
              class="match-item"
            >
              <!-- Match Card -->
              <div class="match-card" @click="toggleMatch(match.ID)">
                <div class="match-header">
                  <div class="match-date">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
                      <line x1="16" y1="2" x2="16" y2="6"/>
                      <line x1="8" y1="2" x2="8" y2="6"/>
                      <line x1="3" y1="10" x2="21" y2="10"/>
                    </svg>
                    <span>{{ formatDate(match.Date) }}</span>
                  </div>
                  
                  <button class="expand-btn" :class="{ 'active': expandedMatch === match.ID }">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="6,9 12,15 18,9"/>
                    </svg>
                  </button>
                </div>

                <!-- Teams and Scores -->
                <div class="teams-section">
                  <div v-for="(team, index) in match.Teams" :key="team.ID" class="team-container">
                    <div class="team-info">
                      <div class="team-color" :style="{ backgroundColor: getTeamColor(team.Colour) }"></div>
                      <span class="team-name">{{ team.Colour }}</span>
                    </div>
                    <div class="team-score">{{ team.Score }}</div>
                  </div>
                </div>

                <!-- Match Status Indicator -->
                <div class="match-status">
                  <div class="status-indicator" :class="getMatchStatus(match)"></div>
                  <span class="status-text">{{ getMatchStatusText(match) }}</span>
                </div>
              </div>

              <!-- Expanded Details -->
              <transition name="match-details">
                <div v-if="expandedMatch === match.ID" class="match-details">
                  <div class="details-header">
                    <h3>Match Details</h3>
                    <div class="details-divider"></div>
                  </div>
                  
                  <div class="players-section">
                    <div class="players-table-container">
                      <table class="players-table">
                        <thead>
                          <tr>
                            <th class="goal-number-col">Goal #</th>
                            <th v-for="team in match.Teams" :key="team.ID" class="team-col">
                              <div class="team-header">
                                <div class="team-color-small" :style="{ backgroundColor: getTeamColor(team.Colour) }"></div>
                                {{ team.Colour }}
                              </div>
                            </th>
                            <th class="goal-number-col">Goal #</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr v-for="rowIndex in getMaxPlayers(match.Teams)" :key="rowIndex" class="player-row">
                            <td class="goal-cell">
                              {{ match.Teams[0].Players[rowIndex - 1]?.GoalNumber || '-' }}
                            </td>
                            <td v-for="team in match.Teams" :key="team.ID" class="player-cell">
                              <div v-if="team.Players[rowIndex - 1]" class="player-info">
                                <div class="player-avatar">
                                  {{ getPlayerInitials(team.Players[rowIndex - 1].Name) }}
                                </div>
                                <span class="player-name">{{ team.Players[rowIndex - 1].Name }}</span>
                              </div>
                              <span v-else class="empty-slot">-</span>
                            </td>
                            <td class="goal-cell">
                              {{ match.Teams[1]?.Players[rowIndex - 1]?.GoalNumber || '-' }}
                            </td>
                          </tr>
                        </tbody>
                      </table>
                    </div>
                  </div>

                  <!-- Match Statistics -->
                  <div class="match-stats">
                    <div class="stats-grid">
                      <div class="stat-item">
                        <span class="stat-label">Total Goals</span>
                        <span class="stat-value">{{ getTotalGoals(match) }}</span>
                      </div>
                      <div class="stat-item">
                        <span class="stat-label">Players</span>
                        <span class="stat-value">{{ getTotalPlayers(match) }}</span>
                      </div>
                      <div class="stat-item">
                        <span class="stat-label">Teams</span>
                        <span class="stat-value">{{ match.Teams.length }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </transition>
            </div>
          </transition-group>
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
      expandedMatch: null,
      isLoading: true
    };
  },
  async created() {
    await this.loadMatches();
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
      } catch (error) {
        console.error('Error fetching matches:', error);
      } finally {
        this.isLoading = false;
      }
    },
    
    toggleMatch(matchID) {
      this.expandedMatch = this.expandedMatch === matchID ? null : matchID;
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
      if (totalGoals > 5) return 'high-scoring';
      return 'completed';
    },
    
    getMatchStatusText(match) {
      const status = this.getMatchStatus(match);
      switch (status) {
        case 'upcoming': return 'Scheduled';
        case 'high-scoring': return 'High Scoring';
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
  --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.3);
  --shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.4);
  --shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.5);
  --shadow-xl: 0 20px 25px -5px rgb(0 0 0 / 0.6);
}

/* Container */
.matches-container {
  min-height: 100vh;
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
  padding: 3rem 0;
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
  padding: 3rem 0;
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

/* Matches Grid */
.matches-grid {
  display: grid;
  gap: 1.5rem;
}

.matches-wrapper {
  display: contents;
}

.match-item {
  background-color: var(--bg-primary);
  border-radius: var(--border-radius-lg);
  box-shadow: var(--shadow-md);
  overflow: hidden;
  transition: all var(--transition-smooth);
  border: 1px solid var(--border-color);
}

.match-item:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
}

/* Match Card */
.match-card {
  padding: 1.5rem;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.match-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.match-date {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.match-date svg {
  width: 18px;
  height: 18px;
}

.expand-btn {
  background: none;
  border: none;
  padding: 0.5rem;
  border-radius: 50%;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
}

.expand-btn:hover {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
}

.expand-btn.active {
  background-color: var(--primary-color);
  color: white;
  transform: rotate(180deg);
}

.expand-btn svg {
  width: 20px;
  height: 20px;
  transition: transform var(--transition-fast);
}

/* Teams Section */
.teams-section {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
}

.team-container {
  flex: 1;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  transition: all var(--transition-fast);
}

.team-container:hover {
  background-color: var(--border-color);
}

.team-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.team-color {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 2px solid white;
  box-shadow: var(--shadow-sm);
}

.team-name {
  font-weight: 600;
  color: var(--text-primary);
  text-transform: capitalize;
}

.team-score {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--primary-color);
  background-color: var(--bg-primary);
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  min-width: 40px;
  text-align: center;
}

/* Match Status */
.match-status {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  justify-content: center;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  animation: pulse-status 2s infinite;
}

.status-indicator.upcoming {
  background-color: #f59e0b;
}

.status-indicator.completed {
  background-color: var(--primary-color);
}

.status-indicator.high-scoring {
  background-color: #ef4444;
}

@keyframes pulse-status {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.status-text {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* Match Details */
.match-details {
  padding: 1.5rem;
  background-color: var(--bg-tertiary);
  border-top: 1px solid var(--border-color);
}

.details-header {
  margin-bottom: 1.5rem;
}

.details-header h3 {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 0.5rem;
}

.details-divider {
  height: 2px;
  background: linear-gradient(90deg, var(--primary-color), var(--secondary-color));
  border-radius: 1px;
  width: 60px;
}

/* Players Table */
.players-table-container {
  overflow-x: auto;
  margin-bottom: 1.5rem;
}

.players-table {
  width: 100%;
  border-collapse: collapse;
  background-color: var(--bg-primary);
  border-radius: var(--border-radius);
  overflow: hidden;
  box-shadow: var(--shadow-sm);
}

.players-table th {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
  font-weight: 600;
  padding: 1rem;
  text-align: center;
  border-bottom: 2px solid var(--border-color);
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
  padding: 0.75rem;
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

/* Match Statistics */
.match-stats {
  background-color: var(--bg-primary);
  border-radius: var(--border-radius);
  padding: 1.5rem;
  box-shadow: var(--shadow-sm);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 1rem;
}

.stat-item {
  text-align: center;
  padding: 1rem;
  border-radius: var(--border-radius);
  background-color: var(--bg-tertiary);
  transition: all var(--transition-fast);
}

.stat-item:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.stat-label {
  display: block;
  font-size: 0.875rem;
  color: var(--text-secondary);
  margin-bottom: 0.25rem;
  font-weight: 500;
}

.stat-value {
  display: block;
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--primary-color);
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
.match-item-enter-active,
.match-item-leave-active {
  transition: all var(--transition-smooth);
}

.match-item-enter-from {
  opacity: 0;
  transform: translateY(20px);
}

.match-item-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

.match-details-enter-active,
.match-details-leave-active {
  transition: all var(--transition-smooth);
  overflow: hidden;
}

.match-details-enter-from,
.match-details-leave-to {
  opacity: 0;
  max-height: 0;
  padding-top: 0;
  padding-bottom: 0;
}

.match-details-enter-to {
  opacity: 1;
  max-height: 1000px;
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

  .teams-section {
    flex-direction: column;
    gap: 0.75rem;
  }

  .team-container {
    padding: 0.75rem;
  }

  .players-table {
    font-size: 0.875rem;
  }

  .players-table th,
  .players-table td {
    padding: 0.5rem;
  }

  .stat-item {
    padding: 0.75rem;
  }

  .goal-number-col {
    width: 60px;
  }

  .team-col {
    min-width: 150px;
  }
}

@media (max-width: 480px) {
  .page-title {
    font-size: 1.75rem;
  }

  .match-card {
    padding: 1rem;
  }

  .match-details {
    padding: 1rem;
  }

  .players-table-container {
    margin: 0 -1rem;
    padding: 0 1rem;
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }
}
</style>
