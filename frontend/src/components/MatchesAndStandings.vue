<template>
  <div class="home-container">
    <!-- Header Section -->
    <section class="home-header">
      <div class="container">
        <div class="header-content">
          <div class="title-section">
            <h1 class="page-title">{{ activeTab.title }}</h1>
            <p class="page-subtitle">{{ activeTab.subtitle }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- Season selector + sub tabs, shared by the three sub-tabs below -->
    <section class="home-controls">
      <div class="container">
        <div v-if="isInitializing" class="loading-container">
          <div class="loading-spinner"></div>
          <p class="loading-text">Loading...</p>
        </div>

        <div v-else class="controls-layout">
          <div class="season-bar card-base">
            <label class="season-label" for="season-select">Season</label>
            <select id="season-select" class="season-select" v-model="selectedSeason" @change="loadStandings">
              <option v-for="season in seasons" :key="season" :value="season">{{ season }}</option>
            </select>
          </div>

          <div class="sub-tabs-bar card-base">
            <button v-for="tab in subTabs" :key="tab.key" @click="activeSubTab = tab.key"
              :class="['sub-tab-button', { active: activeSubTab === tab.key }]">
              {{ tab.label }}
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- Sub-tab content. Everything the children are scoped by (group, admin
         role, season) is resolved here once and passed down. -->
    <template v-if="!isInitializing">
      <MatchesPanel v-if="activeSubTab === 'matches'" :active-group-id="activeGroupId" :is-admin="isAdmin"
        :season="selectedSeason" />

      <section v-else class="standings-section">
        <div class="container">
          <PointsStandingsTable v-if="activeSubTab === 'points'" :rows="pointsStandings"
            :is-loading="isLoadingStandings" />
          <ScorersTable v-else :rows="topScorers" :is-loading="isLoadingStandings" />
        </div>
      </section>
    </template>
  </div>
</template>

<script>
import { getPointsStandings, getScorers, getSeasons } from '@/services/api';
import { resolveActiveGroup } from '@/services/activeGroup';
import MatchesPanel from '@/components/MatchesPanel.vue';
import PointsStandingsTable from '@/components/PointsStandingsTable.vue';
import ScorersTable from '@/components/ScorersTable.vue';

// The app's home page: the group's matches and its two standings tables, under
// one season selector. It owns everything the three sub-tabs share — the
// active group, whether the caller is an admin of it, the list of seasons and
// the selected one — and composes the three presentational children.
export default {
  name: 'MatchesAndStandings',
  components: { MatchesPanel, PointsStandingsTable, ScorersTable },
  data() {
    return {
      // Resolved once, before any child mounts: MatchesPanel creates matches
      // in this group and the standings are scoped to it.
      activeGroupId: '',
      // Caller's role on the active group — only gates UI (the backend's
      // requireGroupAdmin is the real boundary), see MatchesPanel.
      isAdmin: false,
      seasons: [],
      selectedSeason: '',
      isInitializing: true,
      isLoadingStandings: false,
      pointsStandings: [],
      topScorers: [],
      activeSubTab: 'matches',
      subTabs: [
        { key: 'matches', label: 'Matches', title: 'Football Matches', subtitle: 'Track live scores and match details' },
        { key: 'points', label: 'Points', title: 'Standings', subtitle: 'Player rankings across all matches' },
        { key: 'scorers', label: 'Scorers', title: 'Top Scorers', subtitle: 'Goals scored across all matches' }
      ]
    };
  },
  computed: {
    activeTab() {
      return this.subTabs.find(tab => tab.key === this.activeSubTab) || this.subTabs[0];
    }
  },
  async created() {
    try {
      const { groups, activeGroupId } = await resolveActiveGroup();
      this.activeGroupId = activeGroupId;
      this.isAdmin = groups.find(g => g.id === activeGroupId)?.role === 'admin';
    } catch (error) {
      // Same degrade-instead-of-break contract as resolveActiveGroupId():
      // fall through with no group_id (the backend's own first-group
      // fallback) and isAdmin left false, which just hides the admin-only
      // controls.
      console.error('Error resolving the active group:', error);
    }
    await this.loadSeasons();
    this.isInitializing = false;
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
    // Both standings tables are (re)loaded together, as they were when they
    // were two sub-tabs of their own page: switching between Points and
    // Scorers then costs no request. MatchesPanel reloads itself off the
    // season prop instead, since only it knows how to load matches.
    async loadStandings() {
      this.isLoadingStandings = true;
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
        this.isLoadingStandings = false;
      }
    }
  }
};
</script>

<style scoped>
.home-container {
  background-color: var(--bg-secondary);
}

/* Header Section */
.home-header {
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--secondary-color) 100%);
  color: white;
  padding: 1rem 0;
  position: relative;
  overflow: hidden;
}

.home-header::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="2" fill="white" opacity="0.1"/></svg>') repeat;
  animation: float 20s ease-in-out infinite;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  position: relative;
  z-index: 1;
}

.title-section {
  text-align: left;
}

/* Controls: the sub-tab content below brings its own top padding. */
.home-controls {
  padding: 2rem 0 0;
}

.controls-layout {
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

/* Standings sub-tabs — MatchesPanel brings its own section wrapper. */
.standings-section {
  padding: 2rem 0;
}

/* Responsive */
@media (max-width: 768px) {
  .home-header {
    padding: 2rem 0;
  }

  .header-content {
    flex-direction: column;
    gap: 1rem;
    text-align: center;
  }

  .title-section {
    text-align: center;
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
}
</style>
