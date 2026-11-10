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
          <th class="form-col">Last 5</th>
          <th class="points-col">Points</th>
        </tr>
      </thead>
      <tbody>
        <template v-for="(row, index) in rows" :key="row.PlayerID">
          <tr class="player-row" :class="{ expanded: expandedPlayerId === row.PlayerID }"
            :aria-expanded="(expandedPlayerId === row.PlayerID).toString()" @click="toggle(row.PlayerID)">
            <td class="rank-col">{{ index + 1 }}</td>
            <td class="player-col">
              <div class="player-info">
                <span class="player-name">{{ formatPlayerNameForDisplay(row.Name) }}</span>
                <span v-if="row.IsMember === false" class="left-group-tag">(left the group)</span>
              </div>
            </td>
            <td class="form-col">
              <div class="form-squares">
                <span v-for="(slot, i) in lastFive(row)" :key="i" class="form-square"
                  :class="slot ? (slot.Correct ? 'won' : 'lost') : 'empty'"
                  :title="slot ? historyTitle(slot) : ''"></span>
              </div>
            </td>
            <td class="points-col">{{ formatPoints(row.Points) }}</td>
          </tr>
          <tr v-if="expandedPlayerId === row.PlayerID" class="history-row">
            <td colspan="4">
              <ul class="history-list">
                <li v-for="entry in newestFirst(row)" :key="entry.MatchID" class="history-entry">
                  <span class="form-square" :class="entry.Correct ? 'won' : 'lost'"></span>
                  <div class="history-match">
                    <span class="history-date">{{ formatCalendarDayShort(entry.Date) }}</span>
                    <span class="history-score">{{ entry.TeamAName }} {{ entry.ScoreA }} – {{ entry.ScoreB }} {{ entry.TeamBName }}</span>
                  </div>
                  <span class="history-pick">{{ pickName(entry) }} · {{ entry.Odds.toFixed(2) }}x</span>
                  <span class="history-points" :class="entry.Correct ? 'won' : 'lost'">{{ formatEntryPoints(entry.Points) }}</span>
                </li>
              </ul>
            </td>
          </tr>
        </template>
      </tbody>
    </table>
  </div>
</template>

<script>
import { formatCalendarDayShort } from '@/services/datetime';

// The Pronostic page's Leaderboard tab (see Predictions.vue). Each row shows
// the player's last five settled predictions as squares — oldest left,
// newest right, like a football form guide — and clicking the row unfolds
// the whole season's history below it, newest first.
//
// Points is a float (odds-1 for a correct prediction, -1 for a wrong one,
// at one implicit token staked per prediction — see
// app/internal/models/database.go's MatchPrediction comment on the backend
// for why there is no actual stake/balance anywhere), so it is formatted to
// one decimal place rather than printed as a raw float.
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
  data() {
    return {
      expandedPlayerId: ''
    };
  },
  methods: {
    formatCalendarDayShort,
    formatPlayerNameForDisplay(name) {
      return name.split(' ')
        .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
        .join(' ');
    },
    formatPoints(points) {
      const rounded = Math.round(points * 10) / 10;
      return rounded > 0 ? `+${rounded.toFixed(1)}` : rounded.toFixed(1);
    },
    formatEntryPoints(points) {
      return `${points > 0 ? '+' : ''}${points.toFixed(2)}`;
    },
    toggle(playerId) {
      this.expandedPlayerId = this.expandedPlayerId === playerId ? '' : playerId;
    },
    // Always five slots, padded with empty ones on the left, so the squares
    // line up across rows whatever each player's count.
    lastFive(row) {
      const recent = (row.History || []).slice(-5);
      return [...Array(5 - recent.length).fill(null), ...recent];
    },
    newestFirst(row) {
      return [...(row.History || [])].reverse();
    },
    pickName(entry) {
      if (!entry.PredictedTeamID) {
        return 'Draw';
      }
      return entry.PredictedTeamID === entry.TeamAID ? entry.TeamAName : entry.TeamBName;
    },
    historyTitle(entry) {
      return `${entry.TeamAName} ${entry.ScoreA} – ${entry.ScoreB} ${entry.TeamBName}: ${this.pickName(entry)} (${this.formatEntryPoints(entry.Points)})`;
    }
  }
};
</script>

<style scoped>
/* Same family as PointsStandingsTable.vue/ScorersTable.vue/
   MotmStandingsTable.vue. */
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

.player-row {
  cursor: pointer;
}

.standings-table tbody tr.player-row:hover,
.player-row.expanded {
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

/* Form guide: one square per settled prediction. */
.form-squares {
  display: inline-flex;
  gap: 4px;
}

.form-square {
  display: inline-block;
  width: 14px;
  height: 14px;
  flex: 0 0 14px;
  border-radius: 3px;
}

.form-square.won {
  background-color: #10b981;
}

.form-square.lost {
  background-color: #ef4444;
}

.form-square.empty {
  background-color: var(--bg-tertiary);
  border: 1px solid var(--border-color);
}

/* The unfolded history of one player. */
.history-row td {
  padding: 0.25rem 0.75rem 0.75rem;
  background-color: var(--bg-secondary);
}

.history-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.history-entry {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.45rem 0.6rem;
  background-color: var(--bg-primary);
  border-radius: var(--border-radius);
  font-size: 0.85rem;
  text-align: left;
}

.history-match {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.history-date {
  font-size: 0.75rem;
  color: var(--text-secondary);
}

.history-score {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.history-pick {
  color: var(--text-secondary);
  white-space: nowrap;
}

.history-points {
  min-width: 3.25rem;
  text-align: right;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.history-points.won {
  color: #059669;
}

.history-points.lost {
  color: var(--danger-color);
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

  .form-squares {
    gap: 3px;
  }

  .form-square {
    width: 12px;
    height: 12px;
    flex-basis: 12px;
  }

  .history-row td {
    padding: 0.25rem 0.3rem 0.5rem;
  }

  .history-entry {
    gap: 0.45rem;
    padding: 0.4rem 0.45rem;
    font-size: 0.75rem;
  }
}
</style>
