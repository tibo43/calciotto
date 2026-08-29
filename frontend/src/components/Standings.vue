<template>
  <div class="standings-container">
    <!-- Header Section -->
    <section class="standings-header">
      <div class="container">
        <div class="header-content">
          <div class="title-section">
            <h1 class="page-title">Standings</h1>
            <p class="page-subtitle">Player rankings across all matches</p>
          </div>
        </div>
      </div>
    </section>

    <section class="standings-section">
      <div class="container">
        <!-- Loading State -->
        <div v-if="isLoading" class="loading-container">
          <div class="loading-spinner"></div>
          <p class="loading-text">Loading standings...</p>
        </div>

        <div v-else class="standings-layout">
          <!-- Season selector -->
          <div class="season-bar card-base">
            <label class="season-label" for="season-select">Season</label>
            <select id="season-select" class="season-select" v-model="selectedSeason" @change="loadStandings">
              <option v-for="season in seasons" :key="season" :value="season">{{ season }}</option>
            </select>
          </div>

          <!-- Sub tabs -->
          <div class="sub-tabs-bar card-base">
            <button v-for="tab in subTabs" :key="tab.key" @click="activeSubTab = tab.key"
              :class="['sub-tab-button', { active: activeSubTab === tab.key }]">
              {{ tab.label }}
            </button>
          </div>

          <!-- Points standings -->
          <div v-if="activeSubTab === 'points'" class="standings-table-container card-base card-large">
            <div v-if="pointsStandings.length === 0" class="empty-state">
              <div class="empty-content">
                <h3 class="empty-title">No standings yet</h3>
                <p class="empty-description">Play and save a full match to see the rankings.</p>
              </div>
            </div>
            <table v-else class="standings-table">
              <thead>
                <tr>
                  <th class="rank-col">#</th>
                  <th class="player-col">Player</th>
                  <th>P</th>
                  <th>W</th>
                  <th>D</th>
                  <th>L</th>
                  <th>GF</th>
                  <th class="points-col">Pts</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, index) in pointsStandings" :key="row.PlayerID">
                  <td class="rank-col">{{ index + 1 }}</td>
                  <td class="player-col">
                    <div class="player-info">
                      <div class="player-avatar-small">{{ getPlayerInitials(row.Name) }}</div>
                      <span class="player-name">{{ formatPlayerNameForDisplay(row.Name) }}</span>
                      <span v-if="row.IsMember === false" class="left-group-tag">(left the group)</span>
                    </div>
                  </td>
                  <td>{{ row.Played }}</td>
                  <td>{{ row.Won }}</td>
                  <td>{{ row.Drawn }}</td>
                  <td>{{ row.Lost }}</td>
                  <td>{{ row.GoalsFor }}</td>
                  <td class="points-col">{{ row.Points }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Top scorers -->
          <div v-else class="standings-table-container card-base card-large">
            <div v-if="topScorers.length === 0" class="empty-state">
              <div class="empty-content">
                <h3 class="empty-title">No goals yet</h3>
                <p class="empty-description">Save a match with goals to see the top scorers.</p>
              </div>
            </div>
            <table v-else class="standings-table">
              <thead>
                <tr>
                  <th class="rank-col">#</th>
                  <th class="player-col">Player</th>
                  <th>Matches</th>
                  <th class="points-col">Goals</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, index) in topScorers" :key="row.PlayerID">
                  <td class="rank-col">{{ index + 1 }}</td>
                  <td class="player-col">
                    <div class="player-info">
                      <div class="player-avatar-small">{{ getPlayerInitials(row.Name) }}</div>
                      <span class="player-name">{{ formatPlayerNameForDisplay(row.Name) }}</span>
                      <span v-if="row.IsMember === false" class="left-group-tag">(left the group)</span>
                    </div>
                  </td>
                  <td>{{ row.Played }}</td>
                  <td class="points-col">{{ row.Goals }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
import { getPointsStandings, getScorers, getSeasons } from '@/services/api';
import { resolveActiveGroupId } from '@/services/activeGroup';

export default {
  name: 'PlayerStandings',
  data() {
    return {
      pointsStandings: [],
      topScorers: [],
      seasons: [],
      selectedSeason: '',
      // Seasons are derived from the group's own matches, so this has to be
      // resolved before loadSeasons, not just before loadStandings.
      activeGroupId: '',
      isLoading: true,
      activeSubTab: 'points',
      subTabs: [
        { key: 'points', label: 'Points' },
        { key: 'scorers', label: 'Top Scorers' }
      ]
    };
  },
  async created() {
    this.activeGroupId = await resolveActiveGroupId();
    await this.loadSeasons();
    await this.loadStandings();
  },
  methods: {
    async loadSeasons() {
      try {
        const seasons = await getSeasons(this.activeGroupId);
        this.seasons = Array.isArray(seasons) ? seasons : [];
      } catch (error) {
        console.error('Error fetching seasons:', error);
        this.seasons = [];
      }
      // The backend returns the group's seasons in ascending order, so the
      // last one is the most recent — that's what we open on. With no seasons
      // at all (a group without matches), an empty selection means "no
      // filtering", which is exactly what the backend already does.
      this.selectedSeason = this.seasons.length ? this.seasons[this.seasons.length - 1] : '';
    },
    async loadStandings() {
      this.isLoading = true;
      try {
        const [points, scorers] = await Promise.all([
          getPointsStandings(this.selectedSeason, this.activeGroupId),
          getScorers(this.selectedSeason, this.activeGroupId)
        ]);
        this.pointsStandings = Array.isArray(points) ? points : [];
        this.topScorers = Array.isArray(scorers) ? scorers : [];
      } catch (error) {
        console.error('Error fetching standings:', error);
        this.pointsStandings = [];
        this.topScorers = [];
      } finally {
        this.isLoading = false;
      }
    },
    getPlayerInitials(name) {
      return name.split(' ')
        .map(word => word.charAt(0).toUpperCase())
        .join('')
        .slice(0, 2);
    },
    formatPlayerNameForDisplay(name) {
      return name.split(' ')
        .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
        .join(' ');
    }
  }
};
</script>

<style scoped>
.standings-container {
  background-color: var(--bg-secondary);
}

/* Header Section */
.standings-header {
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--secondary-color) 100%);
  color: white;
  padding: 1rem 0;
}

.standings-section {
  padding: 2rem 0;
}

.standings-layout {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

/* Season selector */
.season-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
}

.season-label {
  color: var(--text-secondary);
  font-weight: 600;
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.season-select {
  flex: 1;
  max-width: 220px;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
  background-color: var(--bg-primary);
  color: var(--text-primary);
  font-size: 0.95rem;
  font-weight: 500;
  cursor: pointer;
  transition: border-color var(--transition-fast);
}

.season-select:hover,
.season-select:focus {
  border-color: var(--primary-color);
  outline: none;
}

/* Sub tabs */
.sub-tabs-bar {
  display: flex;
  gap: 0.5rem;
  padding: 0.5rem;
}

.sub-tab-button {
  flex: 1;
  padding: 0.75rem 1.5rem;
  background: none;
  border: none;
  border-radius: var(--border-radius);
  color: var(--text-secondary);
  font-weight: 500;
  font-size: 0.95rem;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.sub-tab-button:hover {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
}

.sub-tab-button.active {
  background-color: var(--primary-color);
  color: white;
  box-shadow: var(--shadow-sm);
}

/* Table */
.standings-table-container {
  overflow-x: auto;
}

.standings-table {
  width: 100%;
  border-collapse: collapse;
}

.standings-table th {
  background-color: var(--bg-tertiary);
  color: var(--text-secondary);
  font-weight: 600;
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  padding: 0.75rem;
  text-align: center;
  border-bottom: 2px solid var(--border-color);
}

.standings-table td {
  padding: 0.75rem;
  text-align: center;
  border-bottom: 1px solid var(--border-color);
  color: var(--text-primary);
}

.standings-table tbody tr:hover {
  background-color: var(--bg-secondary);
}

.rank-col {
  width: 48px;
  font-weight: 700;
  color: var(--text-secondary);
}

.player-col {
  text-align: left;
  min-width: 180px;
}

.player-col .player-info {
  justify-content: flex-start;
}

/* Subtle marker for a row whose player is no longer a group member — their
   historical stats stay fully visible, this just labels the row. */
.left-group-tag {
  color: var(--text-light);
  font-size: 0.75rem;
  font-weight: 400;
  font-style: italic;
}

.points-col {
  font-weight: 700;
  color: var(--primary-color);
  background-color: var(--bg-tertiary);
}

/* Responsive */
@media (max-width: 768px) {
  .standings-header {
    padding: 2rem 0;
  }

  .season-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .season-select {
    max-width: none;
  }

  .sub-tabs-bar {
    flex-direction: column;
  }

  .standings-table th,
  .standings-table td {
    padding: 0.5rem;
    font-size: 0.875rem;
  }
}
</style>
