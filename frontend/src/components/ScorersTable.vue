<template>
  <div class="standings-table-container card-base card-large">
    <div v-if="isLoading" class="loading-container">
      <div class="loading-spinner"></div>
      <p class="loading-text">Loading scorers...</p>
    </div>
    <div v-else-if="rows.length === 0" class="empty-state">
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
        <tr v-for="(row, index) in rows" :key="row.PlayerID">
          <td class="rank-col">{{ index + 1 }}</td>
          <td class="player-col">
            <div class="player-info">
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
</template>

<script>
// The Scorers sub-tab of MatchesAndStandings.vue. Same contract as
// PointsStandingsTable: the page fetches (GET /standings/scorers, already
// sorted), this only renders.
export default {
  name: 'ScorersTable',
  props: {
    rows: {
      type: Array,
      default: () => []
    },
    isLoading: {
      type: Boolean,
      default: false
    }
  },
  methods: {
    formatPlayerNameForDisplay(name) {
      return name.split(' ')
        .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
        .join(' ');
    }
  }
};
</script>

<style scoped>
/* Capped to whatever's left below the nav/context-bar/tabs above it, with
   its own scrollbar — a long list then scrolls inside this card instead
   of growing the whole page, so the tabs/season selector above never
   scroll out of reach. The offset is an estimate of that chrome's height;
   it doesn't need to be exact, just enough that the container's bottom
   edge lands near the viewport's rather than past it. */
.standings-table-container {
  max-height: calc(100vh - 260px);
  overflow: auto;
}

.standings-table {
  width: 100%;
  border-collapse: collapse;
}

.standings-table th {
  position: sticky;
  top: 0;
  z-index: 1;
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
  min-width: 0;
}

.player-info .player-name {
  overflow: hidden;
  text-overflow: ellipsis;
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

/* Responsive — shrunk enough (no avatar, tighter padding, a capped name
   width) that the whole table fits without any horizontal scroll, on top
   of .standings-table-container's own overflow-x:auto safety net for an
   unusually long name. */
@media (max-width: 768px) {
  .standings-table th,
  .standings-table td {
    padding: 0.4rem 0.3rem;
    font-size: 0.75rem;
  }

  .rank-col {
    width: 20px;
  }

  .player-col {
    min-width: 0;
  }

  .player-info .player-name {
    display: inline-block;
    max-width: 8rem;
    white-space: nowrap;
    vertical-align: bottom;
  }

  .left-group-tag {
    display: block;
    font-size: 0.65rem;
  }
}
</style>
