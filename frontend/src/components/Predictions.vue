<template>
  <div class="home-container">
    <!-- Same context-bar pattern as MatchesAndStandings.vue — which group and
         season this page is scoped to — just with no sub-tabs row below it,
         since this page has only the one view. -->
    <section class="home-controls">
      <div class="container">
        <div v-if="isInitializing" class="loading-container">
          <div class="loading-spinner"></div>
          <p class="loading-text">Loading...</p>
        </div>

        <div v-else class="context-bar card-base">
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
      </div>
    </section>

    <PredictionsPanel v-if="!isInitializing" :active-group-id="activeGroupId" :season="selectedSeason" />
  </div>
</template>

<script>
import { getSeasons } from '@/services/api';
import { resolveActiveGroup, setActiveGroupId } from '@/services/activeGroup';
import { seasonOf } from '@/services/datetime';
import PredictionsPanel from '@/components/PredictionsPanel.vue';

// Its own top-level page now (see router/index.js), not a sub-tab of
// MatchesAndStandings.vue any more — it owns the same active-group/season
// resolution that page already established, since it can no longer read
// those off a parent. PredictionsPanel.vue itself still does all the actual
// work (match carousel, the betting widget, the leaderboard); this is purely
// the context-scoping shell, mirroring MatchesAndStandings.vue's own split
// with MatchesPanel.vue.
export default {
  name: 'PredictionsPage',
  components: { PredictionsPanel },
  data() {
    return {
      activeGroupId: '',
      groups: [],
      seasons: [],
      selectedSeason: '',
      isInitializing: true
    };
  },
  async created() {
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
}
</style>
