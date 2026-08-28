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

          <!-- Per-group breakdown -->
          <div class="standings-table-container card-base card-large">
            <div v-if="perGroup.length === 0" class="empty-state">
              <div class="empty-content">
                <h3 class="empty-title">No group yet</h3>
                <p class="empty-description">Join a group to start building your record.</p>
              </div>
            </div>
            <table v-else class="standings-table">
              <thead>
                <tr>
                  <th class="player-col">Group</th>
                  <th>P</th>
                  <th>W</th>
                  <th>D</th>
                  <th>L</th>
                  <th>GF</th>
                  <th class="points-col">Pts</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in perGroup" :key="row.GroupID">
                  <td class="player-col">
                    <span class="group-name">{{ row.GroupName }}</span>
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
        </div>
      </div>
    </section>
  </div>
</template>

<script>
import { getPlayerProfile } from '@/services/api';

export default {
  name: 'PlayerProfile',
  data() {
    return {
      overall: { Name: '', Played: 0, Won: 0, Drawn: 0, Lost: 0, GoalsFor: 0, Points: 0 },
      perGroup: [],
      isLoading: true,
      loadFailed: false
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
    }
  },
  async created() {
    await this.loadProfile();
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
/* Layout mirrors Standings.vue — same container/header/table treatment, so
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

/* Table — same rules as Standings.vue's per-group ranking */
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

.player-col {
  text-align: left;
  min-width: 180px;
}

.group-name {
  font-weight: 600;
}

.points-col {
  font-weight: 700;
  color: var(--primary-color);
  background-color: var(--bg-tertiary);
}

/* Responsive */
@media (max-width: 768px) {
  .profile-header {
    padding: 2rem 0;
  }

  .overall-stats {
    grid-template-columns: repeat(3, 1fr);
  }

  .standings-table th,
  .standings-table td {
    padding: 0.5rem;
    font-size: 0.875rem;
  }
}
</style>
