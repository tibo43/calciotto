<template>
  <PredictionsV2 v-if="designV2" />
  <div v-else class="home-container">
    <!-- Same context-bar + sub-tabs pattern as MatchesAndStandings.vue: which
         group and season this page is scoped to, then which view of it. -->
    <section class="home-controls">
      <div class="container">
        <div v-if="isInitializing" class="loading-container">
          <div class="loading-spinner"></div>
          <p class="loading-text">Loading...</p>
        </div>

        <template v-else>
          <div class="context-bar card-base">
            <div class="context-field">
              <template v-if="groups.length > 0">
                <label class="context-label" for="group-select">Group</label>
                <select id="group-select" class="context-select" v-model="activeGroupId" @change="switchGroup">
                  <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
                </select>
              </template>
              <span v-else class="no-group-hint">No group yet — ask an admin to invite you (creating or joining a group yourself isn't available yet).</span>
            </div>

            <div class="context-field">
              <label class="context-label" for="season-select">Season</label>
              <select id="season-select" class="context-select" v-model="selectedSeason">
                <option v-for="season in seasons" :key="season" :value="season">{{ season }}</option>
              </select>
            </div>
          </div>

          <div class="sub-tabs-bar">
            <button v-for="tab in subTabs" :key="tab.key" @click="activeSubTab = tab.key"
              :class="['sub-tab-button', { active: activeSubTab === tab.key }]">
              {{ tab.label }}
            </button>
          </div>
        </template>
      </div>
    </section>

    <template v-if="!isInitializing">
      <PredictionsPanel v-if="activeSubTab === 'matches'" :active-group-id="activeGroupId" :season="selectedSeason" />
      <section v-else class="standings-section">
        <div class="container">
          <PredictionsStandingsTable :rows="standings" :is-loading="isLoadingStandings" />
        </div>
      </section>
    </template>
  </div>
</template>

<script>
import { getSeasons, getPredictionStandings } from '@/services/api';
import { resolveActiveGroup, setActiveGroupId } from '@/services/activeGroup';
import { seasonOf } from '@/services/datetime';
import PredictionsPanel from '@/components/PredictionsPanel.vue';
import PredictionsStandingsTable from '@/components/PredictionsStandingsTable.vue';
import PredictionsV2 from '@/components/v2/PredictionsV2.vue';

// Its own top-level page now (see router/index.js), not a sub-tab of
// MatchesAndStandings.vue any more — it owns the same active-group/season
// resolution that page already established, since it can no longer read
// those off a parent. Like MatchesAndStandings.vue it has two sub-tabs:
// PredictionsPanel.vue (match carousel and betting widget) and the season
// leaderboard, whose data this shell loads itself, as that page does for its
// own standings tables.
export default {
  name: 'PredictionsPage',
  components: { PredictionsPanel, PredictionsStandingsTable, PredictionsV2 },
  inject: {
    designV2On: { default: false },
  },
  computed: {
    designV2() {
      return Boolean(this.designV2On);
    },
  },
  data() {
    return {
      activeGroupId: '',
      groups: [],
      seasons: [],
      selectedSeason: '',
      isInitializing: true,
      standings: [],
      isLoadingStandings: false,
      activeSubTab: 'matches',
      subTabs: [
        { key: 'matches', label: 'Matches' },
        { key: 'leaderboard', label: 'Leaderboard' }
      ]
    };
  },
  watch: {
    selectedSeason() {
      this.loadStandings();
    }
  },
  async created() {
    if (this.designV2) return;
    try {
      const { groups, activeGroupId } = await resolveActiveGroup();
      this.groups = groups;
      this.activeGroupId = activeGroupId;
    } catch (error) {
      // Same degrade-instead-of-break contract as resolveActiveGroupId()'s
      // other callers: fall through with no group_id (the backend's own
      // first-group fallback).
      console.error('Error resolving the active group:', error);
    }
    await this.loadSeasons();
    this.isInitializing = false;
  },
  methods: {
    // Same full-reload contract as MatchesAndStandings.vue's own
    // switchGroup() — there is no reactive store, so a scoped page only ever
    // re-reads the active group on created().
    switchGroup() {
      setActiveGroupId(this.activeGroupId);
      window.location.reload();
    },
    async loadStandings() {
      if (!this.activeGroupId) {
        this.standings = [];
        return;
      }
      this.isLoadingStandings = true;
      try {
        const rows = await getPredictionStandings(this.selectedSeason, this.activeGroupId);
        this.standings = Array.isArray(rows) ? rows : [];
      } catch (error) {
        console.error('Error fetching prediction standings:', error);
        this.standings = [];
      } finally {
        this.isLoadingStandings = false;
      }
    },
    async loadSeasons() {
      try {
        const seasons = await getSeasons(this.activeGroupId);
        this.seasons = Array.isArray(seasons) ? seasons : [];
      } catch (error) {
        console.error('Error fetching seasons:', error);
        this.seasons = [];
      }
      // Same "default to whichever season today actually falls in" rule as
      // MatchesAndStandings.vue's own loadSeasons() — see its comment for why
      // "the last season in the list" is the wrong default.
      const current = seasonOf(new Date());
      if (this.seasons.includes(current)) {
        this.selectedSeason = current;
      } else {
        this.selectedSeason = this.seasons.length ? this.seasons[this.seasons.length - 1] : '';
      }
    }
  }
};
</script>

<style scoped>
/* Identical to MatchesAndStandings.vue's own — duplicated rather than
   shared, the same "small pieces duplicated per component" convention as
   getTeamColor()/formatPlayerNameForDisplay() elsewhere in this codebase. */
.home-container {
  background-color: var(--bg-secondary);
}

.home-controls {
  padding: 1.5rem 0 0;
}

.context-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 0.75rem 6rem;
  padding: 0.75rem 1rem;
  margin-bottom: 1rem;
}

.context-field {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.context-label {
  color: var(--text-secondary);
  font-weight: 600;
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.context-select {
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

.context-select:hover,
.context-select:focus {
  border-color: var(--primary-color);
  outline: none;
}

.no-group-hint {
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.sub-tabs-bar {
  display: flex;
  gap: 0.5rem;
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

.standings-section {
  padding: 2rem 0;
}

@media (max-width: 768px) {
  .home-controls {
    padding: 1rem 0 0;
  }

  .context-bar {
    justify-content: center;
    flex-wrap: nowrap;
    gap: 0.75rem;
    margin-bottom: 0.75rem;
  }

  .context-label {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }

  /* min-width: 0 lets flex: 1 shrink below the label's width — see
     MatchesAndStandings.vue for the overflow this prevents. */
  .sub-tabs-bar {
    gap: 0.35rem;
  }

  .sub-tab-button {
    min-width: 0;
    padding: 0.6rem 0.5rem;
    font-size: 0.8rem;
  }
}
</style>
