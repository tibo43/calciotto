<template>
  <div class="standings-table-container card-base card-large">
    <div v-if="isLoading" class="loading-container">
      <div class="loading-spinner"></div>
      <p class="loading-text">Loading standings...</p>
    </div>
    <div v-else-if="rows.length === 0" class="empty-state">
      <div class="empty-content">
        <h3 class="empty-title">No predictions yet</h3>
        <p class="empty-description">Predict a match's result once its teams are set to see the leaderboard.</p>
      </div>
    </div>
    <table v-else class="standings-table">
      <thead>
        <tr>
          <th class="rank-col">#</th>
          <th class="player-col">Player</th>
          <th>Predictions</th>
          <th>Correct</th>
          <th class="points-col">Points</th>
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
          <td>{{ row.Predictions }}</td>
          <td>{{ row.Correct }}</td>
          <td class="points-col">{{ formatPoints(row.Points) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script>
// The Predictions sub-tab's leaderboard, rendered inside PredictionsPanel.vue
// alongside the betting widget itself — unlike Points/Scorers/MOTM, this
// table is not a standalone sub-tab of its own, per explicit product
// direction: predictions are entirely their own feature, sharing no tab,
// data, or UI with Man of the Match.
//
// Points is a float (odds-1 for a correct prediction, -1 for a wrong one,
// at one implicit token staked per prediction — see
// app/internal/models/database.go's MatchPrediction comment on the backend
// for why there is no actual stake/balance anywhere), so it is formatted to
// one decimal place rather than printed as a raw float — "+1.86" reads far
// better than "1.8571428571428572".
export default {
  name: 'PredictionsStandingsTable',
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
    },
    formatPoints(points) {
      const rounded = Math.round(points * 10) / 10;
      return rounded > 0 ? `+${rounded.toFixed(1)}` : rounded.toFixed(1);
    }
  }
};
</script>

<style scoped>
/* Same family as PointsStandingsTable.vue/ScorersTable.vue/
   MotmStandingsTable.vue, just with one more column than Scorers/MOTM. */
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

.points-col {
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
