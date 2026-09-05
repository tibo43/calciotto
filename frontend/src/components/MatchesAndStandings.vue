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
        :season="selectedSeason" :deep-link-match-id="deepLinkMatchId" />

      <section v-else class="standings-section">
        <div class="container">
          <PointsStandingsTable v-if="activeSubTab === 'points'" :rows="pointsStandings"
            :is-loading="isLoadingStandings" />
          <ScorersTable v-else-if="activeSubTab === 'scorers'" :rows="topScorers" :is-loading="isLoadingStandings" />
          <MotmStandingsTable v-else :rows="motmStandings" :is-loading="isLoadingStandings" />
        </div>
      </section>
    </template>
  </div>
</template>

<script>
import { getPointsStandings, getScorers, getMotmStandings, getSeasons } from '@/services/api';
import { resolveActiveGroup, setActiveGroupId } from '@/services/activeGroup';
import { seasonOf, parseCalendarDay } from '@/services/datetime';
import { decodeMatchId } from '@/services/shortLink';
import { findGroupForMatch } from '@/router/index';
import MatchesPanel from '@/components/MatchesPanel.vue';
import PointsStandingsTable from '@/components/PointsStandingsTable.vue';
import ScorersTable from '@/components/ScorersTable.vue';
import MotmStandingsTable from '@/components/MotmStandingsTable.vue';

// The app's home page: the group's matches and its three standings tables,
// under one season selector. It owns everything the four sub-tabs share — the
// active group, whether the caller is an admin of it, the list of seasons and
// the selected one — and composes the four presentational children.
export default {
  name: 'MatchesAndStandings',
  components: { MatchesPanel, PointsStandingsTable, ScorersTable, MotmStandingsTable },
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
      motmStandings: [],
      // A match id decoded from a shared `/m/:code` link's `?match=` query
      // param (see router/index.js), once resolveDeepLinkedMatch() below has
      // confirmed which of the caller's groups it belongs to. Passed down to
      // MatchesPanel, which auto-selects it once its own matches list has
      // loaded. Empty is the ordinary case — no deep link at all.
      deepLinkMatchId: '',
      // The deep-linked match's own season (see resolveDeepLinkedMatch()),
      // read by loadSeasons() before it defaults selectedSeason. Empty means
      // "no override" — the ordinary seasonOf(now) default applies.
      deepLinkSeason: '',
      activeSubTab: 'matches',
      subTabs: [
        { key: 'matches', label: 'Matches' },
        { key: 'points', label: 'Points' },
        { key: 'scorers', label: 'Scorers' },
        { key: 'motm', label: 'MOTM' }
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

    // A shared match link (router/index.js's `/m/:code` redirect) lands here
    // rather than on MatchDetails.vue directly — see that route's own
    // comment for why. If resolveDeepLinkedMatch() switches the active group,
    // it triggers a full reload and returns true; there is nothing more to
    // do on this instance of the page, since the reloaded one re-runs
    // created() from scratch against the new group_id.
    const reloading = await this.resolveDeepLinkedMatch();
    if (reloading) {
      return;
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

    // Decodes a `?match=<code>` deep link (see router/index.js) and, on
    // success, forces the Matches tab active and preselects the match's own
    // season before loadSeasons() runs — otherwise the current-season default
    // could filter the deep-linked match straight out of the list it's meant
    // to land in. Returns true if it triggered a full-page reload to switch
    // into the match's own group (the caller must stop here and let the
    // reloaded page start over), false otherwise.
    //
    // Every failure mode — no `?match=` param, an undecodable code, the
    // match belonging to no group the caller is in, a network error along
    // the way — degrades silently to the ordinary page with no selection,
    // mirroring the `/m/:code` redirect's own "malformed code falls back
    // home" philosophy: a shared link that doesn't resolve is not worth an
    // error toast.
    async resolveDeepLinkedMatch() {
      const code = this.$route?.query?.match;
      if (!code) {
        return false;
      }
      try {
        const matchId = decodeMatchId(code);
        const found = await findGroupForMatch(matchId, this.groups);
        if (!found) {
          return false;
        }
        this.activeSubTab = 'matches';
        this.deepLinkMatchId = matchId;
        this.deepLinkSeason = seasonOf(parseCalendarDay(found.details.Date));
        if (found.group.id !== this.activeGroupId) {
          setActiveGroupId(found.group.id);
          window.location.reload();
          return true;
        }
      } catch (error) {
        console.error('Error resolving the shared match link:', error);
      }
      return false;
    },

    async loadSeasons() {
      try {
        const seasons = await getSeasons(this.activeGroupId);
        this.seasons = Array.isArray(seasons) ? seasons : [];
      } catch (error) {
        console.error('Error fetching seasons:', error);
        this.seasons = [];
      }
      // Default to whichever season *today* actually falls in, not to "the
      // last one in the list": ComputeSeasons counts a scheduled match even
      // before it's played, so scheduling one far enough out introduces a
      // season later than today's own — and picking "the last entry" would
      // silently jump the default view onto that near-empty future season
      // instead of the history in progress. seasonOf(now) is stable against
      // that; only a season with no matches *at all* falls back to the last
      // one that has some, the same "no filtering" fallback as before.
      //
      // A deep-linked match's own season (resolveDeepLinkedMatch(), above)
      // overrides that default outright — the whole point of the deep link
      // is to land on that match, which the current-season default could
      // otherwise filter straight out of the list.
      const current = this.deepLinkSeason || seasonOf(new Date());
      if (this.seasons.includes(current)) {
        this.selectedSeason = current;
      } else {
        this.selectedSeason = this.seasons.length ? this.seasons[this.seasons.length - 1] : '';
      }
    },
    // All three standings tables are (re)loaded together, as the first two
    // were when they were sub-tabs of their own page: switching between
    // Points, Scorers and MOTM then costs no request. MatchesPanel reloads
    // itself off the season prop instead, since only it knows how to load
    // matches.
    async loadStandings() {
      this.isLoadingStandings = true;
      try {
        const [points, scorers, motm] = await Promise.all([
          getPointsStandings(this.selectedSeason, this.activeGroupId),
          getScorers(this.selectedSeason, this.activeGroupId),
          getMotmStandings(this.selectedSeason, this.activeGroupId)
        ]);
        this.pointsStandings = Array.isArray(points) ? points : [];
        this.topScorers = Array.isArray(scorers) ? scorers : [];
        this.motmStandings = Array.isArray(motm) ? motm : [];
      } catch (error) {
        console.error('Error fetching standings:', error);
        this.pointsStandings = [];
        this.topScorers = [];
        this.motmStandings = [];
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
  .home-controls {
    padding: 1rem 0 0;
  }

  .context-bar {
    /* Group and season sit on one row here too, not stacked: dropping the
       "Group"/"Season" labels below frees up enough width for both
       selects to fit side by side instead of each wrapping onto its own
       line. Centered, same as desktop. */
    justify-content: center;
    flex-wrap: nowrap;
    gap: 0.75rem;
    margin-bottom: 0.75rem;
  }

  /* Visually hidden rather than display:none, so the <select> — still
     associated via its for/id pair — keeps an accessible name for screen
     readers even though sighted mobile users no longer see the label
     text. */
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

  /* Four equal-flex tabs (Matches/Points/Scorers/MOTM) each default to
     min-width: auto, which for a flex item means "at least my content's own
     width" — so on a narrow phone the row simply overflowed the viewport
     instead of shrinking, MOTM's button sticking out past the right edge
     needing a sideways scroll to see. This was already tight with three tabs;
     MOTM's addition was what pushed it over. min-width: 0 is what actually
     lets flex: 1 shrink the buttons below their content size; the smaller
     padding/font just keep "Matches"/"Scorers" legible at that width. */
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
