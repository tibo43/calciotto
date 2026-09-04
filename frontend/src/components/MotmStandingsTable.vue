<template>
  <div class="standings-table-container card-base card-large">
    <div v-if="isLoading" class="loading-container">
      <div class="loading-spinner"></div>
      <p class="loading-text">Loading standings...</p>
    </div>
    <div v-else-if="rows.length === 0" class="empty-state">
      <div class="empty-content">
        <h3 class="empty-title">No Man of the Match awards yet</h3>
        <p class="empty-description">Vote on a played match to see the leaderboard.</p>
      </div>
    </div>
    <table v-else class="standings-table">
      <thead>
        <tr>
          <th class="rank-col">#</th>
          <th class="player-col">Player</th>
          <th class="awards-col">Awards</th>
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
          <td class="awards-col">{{ row.Awards }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script>
// The MOTM sub-tab of MatchesAndStandings.vue. Pure presentation, mirroring
// PointsStandingsTable.vue/ScorersTable.vue exactly: the page owns the
// group/season scoping and the fetch, this renders whatever rows the backend
// returned (GET /standings/motm is already sorted, most awards first).
//
// Awards is a count of matches won, not a percentage or anything derived —
// a player tied for MOTM in several matches has each one counted, which is
// why the number can be larger than a naive "matches played / number of
// candidates" estimate would suggest (see ComputeMotmWinners on the backend
// for the tie-inclusive rule this reflects).
export default {
  name: 'MotmStandingsTable',
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
/* Identical layout to PointsStandingsTable.vue/ScorersTable.vue on purpose —
   the three standings tabs should read as one family, not three unrelated
   designs. */
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

.left-group-tag {
  color: var(--text-light);
  font-size: 0.75rem;
  font-weight: 400;
  font-style: italic;
}

.awards-col {
  font-weight: 700;
  color: var(--primary-color);
  background-color: var(--bg-tertiary);
}

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
    max-width: 5.5rem;
    white-space: nowrap;
    vertical-align: bottom;
  }

  .left-group-tag {
    display: block;
    font-size: 0.65rem;
  }
}
</style>
