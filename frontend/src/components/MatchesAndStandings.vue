<template>
  <div class="home-container">
    <!-- Context bar (which group/season we're scoped to) and sub-tab
         navigation, each on their own row — replaces the old gradient hero
         banner, which was purely decorative (a title/subtitle that only
         repeated what the active tab already shows). -->
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
              <span v-else class="no-group-hint">No group yet — join or create one from your Profile.</span>
            </div>

            <div class="context-field">
              <label class="context-label" for="season-select">Season</label>
              <select id="season-select" class="context-select" v-model="selectedSeason" @change="loadStandings">
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
import { resolveActiveGroup, setActiveGroupId } from '@/services/activeGroup';
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
      // The full list backing the group selector, same level as the season
      // one — this is the one and only place the active group is switched
      // from now (it used to be a global navbar selector, but nothing
      // outside Matches/Standings ever consumed it).
      groups: [],
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
        { key: 'matches', label: 'Matches' },
        { key: 'points', label: 'Points' },
        { key: 'scorers', label: 'Scorers' }
      ]
    };
  },
  async created() {
    try {
      const { groups, activeGroupId } = await resolveActiveGroup();
      this.groups = groups;
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
    // Same reasoning as the old navbar selector: there is no reactive store,
    // every scoped view resolves the active group once in created(), so a
    // full reload is what actually re-scopes the page.
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

/* Controls: replaces the old gradient hero — a plain top padding is enough
   since there's no banner to separate from below it. */
.home-controls {
  padding: 1.5rem 0 0;
}

/* Context bar: which group and season the page is scoped to. Its own row,
   separate from the sub-tabs below, so it reads as a data-scoping control
   rather than something bolted onto the tab navigation. */
.context-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.75rem 2.5rem;
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

/* Sub tabs: pure navigation, on its own row now that it no longer shares
   space with the context bar. */
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

/* Standings sub-tabs — MatchesPanel brings its own section wrapper. */
.standings-section {
  padding: 2rem 0;
}

/* Responsive */
@media (max-width: 768px) {
  .context-bar {
    justify-content: space-between;
  }
}
</style>
