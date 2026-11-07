<template>
  <section class="predictions-section">
    <div class="container">
      <div class="predictions-matches card-base card-large">
        <h2 class="section-title">Predict a match</h2>

        <div v-if="isLoadingMatches" class="loading-container">
          <div class="loading-spinner"></div>
          <p class="loading-text">Loading matches...</p>
        </div>
        <div v-else-if="eligibleMatches.length === 0" class="empty-state">
          <div class="empty-content">
            <h3 class="empty-title">Nothing to predict yet</h3>
            <p class="empty-description">Once a scheduled match's teams are composed, it shows up here to predict before kick-off.</p>
          </div>
        </div>
        <ul v-else class="match-pick-list">
          <li v-for="match in eligibleMatches" :key="match.ID" class="match-pick-item"
            :class="{ active: selectedMatchId === match.ID }" @click="selectMatch(match.ID)">
            <span class="match-pick-date">{{ formatDateTimeShort(match.ScheduledAt) }}</span>
            <span class="match-pick-teams">
              <span class="team-color-small" :style="{ backgroundColor: getTeamColor(match.Teams[0].Colour) }"></span>
              {{ match.Teams[0].Name }}
              <span class="match-pick-vs">vs</span>
              {{ match.Teams[1].Name }}
              <span class="team-color-small" :style="{ backgroundColor: getTeamColor(match.Teams[1].Colour) }"></span>
            </span>
          </li>
        </ul>
      </div>

      <div v-if="selectedMatch" class="prediction-widget card-base card-large">
        <h3 class="widget-title">{{ selectedMatch.Teams[0].Name }} vs {{ selectedMatch.Teams[1].Name }}</h3>
        <p class="widget-kickoff">{{ formatDateTimeShort(selectedMatch.ScheduledAt) }}</p>
        <p v-if="!predictionsOpen" class="prediction-closed-hint">{{ predictionClosedReason }}</p>

        <div v-if="isLoadingPredictions" class="loading-container">
          <div class="loading-spinner"></div>
        </div>
        <template v-else>
          <div class="prediction-picks">
            <button type="button" class="prediction-pick-btn" :class="{ selected: isMyPick(selectedMatch.Teams[0].ID) }"
              :disabled="!predictionsOpen || isSubmitting" @click="predict(selectedMatch.Teams[0].ID)">
              <span class="pick-label">{{ selectedMatch.Teams[0].Name }}</span>
              <span v-if="odds" class="pick-odds">{{ formatOdds(odds.TeamAOdds) }}</span>
            </button>
            <button type="button" class="prediction-pick-btn" :class="{ selected: isMyPick(null) }"
              :disabled="!predictionsOpen || isSubmitting" @click="predict(null)">
              <span class="pick-label">Draw</span>
              <span v-if="odds" class="pick-odds">{{ formatOdds(odds.DrawOdds) }}</span>
            </button>
            <button type="button" class="prediction-pick-btn" :class="{ selected: isMyPick(selectedMatch.Teams[1].ID) }"
              :disabled="!predictionsOpen || isSubmitting" @click="predict(selectedMatch.Teams[1].ID)">
              <span class="pick-label">{{ selectedMatch.Teams[1].Name }}</span>
              <span v-if="odds" class="pick-odds">{{ formatOdds(odds.TeamBOdds) }}</span>
            </button>
          </div>

          <p v-if="predictionMessage" class="prediction-message" :class="predictionMessageType">{{ predictionMessage }}</p>

          <ul v-if="entries.length > 0" class="prediction-entries">
            <li v-for="entry in entries" :key="entry.PlayerID" class="prediction-entry">
              <span class="prediction-entry-name">{{ formatPlayerNameForDisplay(entry.Name) }}</span>
              <span class="prediction-entry-pick">{{ pickLabel(entry.PredictedTeamID) }}</span>
            </li>
          </ul>
        </template>
      </div>

      <PredictionsStandingsTable :rows="standings" :is-loading="isLoadingStandings" />
    </div>
  </section>
</template>

<script>
// The Predictions sub-tab of MatchesAndStandings.vue: pick a scheduled,
// fully-composed match, back a win/draw/loss outcome at odds derived from
// both teams' Elo rating (see ComputeMatchOdds on the backend), and see the
// season leaderboard. Deliberately its own, self-contained panel rather than
// another row of the generic standings section the other three sub-tabs
// share — it owns its own fetching the same way MatchesPanel.vue does,
// rather than depend on anything the page's own Promise.all standings loader
// fetches — and it shares no data, component, or UI with Man of the Match,
// by explicit product direction.
import {
  getMatchesDetails,
  getMatchPredictions,
  predictForMatch,
  getPredictionStandings,
  getToken
} from '@/services/api';
import { formatDateTimeShort } from '@/services/datetime';
import { isScheduledMatch } from '@/services/matchRegistration';
import { rosterFullyComposed, predictionsAreOpen, predictionState, PREDICTIONS_CLOSED } from '@/services/matchPrediction';
import PredictionsStandingsTable from '@/components/PredictionsStandingsTable.vue';

// Same shape as MatchesPanel.vue's/MatchDetails.vue's/Profile.vue's own
// helper — the app has no auth store, so each page decodes the JWT payload
// locally rather than a third place holding state.
function currentPlayerIdFromToken() {
  const token = getToken();
  if (!token) {
    return '';
  }
  try {
    const payload = token.split('.')[1];
    const decoded = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')));
    return decoded.player_id || '';
  } catch (error) {
    console.error('Error decoding token:', error);
    return '';
  }
}

export default {
  name: 'PredictionsPanel',
  components: { PredictionsStandingsTable },
  props: {
    activeGroupId: {
      type: String,
      default: ''
    },
    season: {
      type: String,
      default: ''
    }
  },
  data() {
    return {
      matches: [],
      standings: [],
      selectedMatchId: '',
      entries: [],
      odds: null,
      myPrediction: null,
      isLoadingMatches: false,
      isLoadingStandings: false,
      isLoadingPredictions: false,
      isSubmitting: false,
      predictionMessage: '',
      predictionMessageType: 'info',
      currentPlayerId: '',
      // Sampled once, like every other clock-dependent view in this app
      // (MatchesPanel.vue's own registrationState/motmVotingOpen) — there is
      // no reactive clock anywhere, a player who had the tab open across
      // kick-off sees the window close on their next reload.
      nowMs: Date.now()
    };
  },
  computed: {
    eligibleMatches() {
      return this.matches.filter(match => isScheduledMatch(match) && rosterFullyComposed(match));
    },
    selectedMatch() {
      return this.eligibleMatches.find(match => match.ID === this.selectedMatchId) || null;
    },
    predictionsOpen() {
      return this.selectedMatch ? predictionsAreOpen(this.selectedMatch, this.nowMs) : false;
    },
    predictionClosedReason() {
      if (!this.selectedMatch) {
        return '';
      }
      return predictionState(this.selectedMatch, this.nowMs) === PREDICTIONS_CLOSED
        ? 'Predictions are closed — kick-off has passed.'
        : "Predictions aren't open for this match yet.";
    }
  },
  watch: {
    // The season selector live-updates without a page reload (unlike the
    // active group, which only ever changes through one — see
    // activeGroup.js), so this is the one prop this panel actually needs to
    // react to, the same convention MatchesPanel.vue's own season watch
    // follows.
    season() {
      this.selectedMatchId = '';
      this.loadMatches();
      this.loadStandings();
    }
  },
  created() {
    this.currentPlayerId = currentPlayerIdFromToken();
    this.loadMatches();
    this.loadStandings();
  },
  methods: {
    formatDateTimeShort,
    getTeamColor(colour) {
      if (colour && colour.startsWith('#')) {
        return colour;
      }
      const colorMap = {
        red: '#ef4444',
        blue: '#3b82f6',
        green: '#10b981',
        yellow: '#f59e0b',
        purple: '#8b5cf6',
        orange: '#f97316',
        pink: '#ec4899',
        cyan: '#06b6d4',
        white: '#f8fafc',
        black: '#1f2937'
      };
      return colorMap[(colour || '').toLowerCase()] || '#6b7280';
    },
    formatPlayerNameForDisplay(name) {
      return name.split(' ')
        .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
        .join(' ');
    },
    formatOdds(value) {
      return `${value.toFixed(2)}x`;
    },
    pickLabel(predictedTeamId) {
      if (!this.selectedMatch) {
        return '';
      }
      if (!predictedTeamId) {
        return 'Draw';
      }
      if (predictedTeamId === this.selectedMatch.Teams[0].ID) {
        return this.selectedMatch.Teams[0].Name;
      }
      if (predictedTeamId === this.selectedMatch.Teams[1].ID) {
        return this.selectedMatch.Teams[1].Name;
      }
      return '';
    },
    // null both represents "the Draw button" and "no prediction made" —
    // deliberately the same comparison either way: a caller who hasn't
    // predicted sees no button highlighted, since myPrediction itself is
    // null (see MatchPredictionSummary.MyPrediction's own comment on the
    // backend for why the *outer* pointer, not the inner one, carries
    // "absent").
    isMyPick(teamId) {
      if (!this.myPrediction) {
        return false;
      }
      return (this.myPrediction.PredictedTeamID || null) === teamId;
    },
    async loadMatches() {
      if (!this.activeGroupId) {
        this.matches = [];
        return;
      }
      this.isLoadingMatches = true;
      try {
        const data = await getMatchesDetails(this.activeGroupId, this.season);
        this.matches = Array.isArray(data) ? data : [];
        if (!this.selectedMatchId && this.eligibleMatches.length > 0) {
          await this.selectMatch(this.eligibleMatches[0].ID);
        }
      } catch (error) {
        this.matches = [];
      } finally {
        this.isLoadingMatches = false;
      }
    },
    async loadStandings() {
      if (!this.activeGroupId) {
        this.standings = [];
        return;
      }
      this.isLoadingStandings = true;
      try {
        const rows = await getPredictionStandings(this.season, this.activeGroupId);
        this.standings = Array.isArray(rows) ? rows : [];
      } catch (error) {
        this.standings = [];
      } finally {
        this.isLoadingStandings = false;
      }
    },
    async selectMatch(matchId) {
      this.selectedMatchId = matchId;
      this.predictionMessage = '';
      await this.loadSelectedPredictions();
    },
    async loadSelectedPredictions() {
      if (!this.selectedMatchId) {
        return;
      }
      this.isLoadingPredictions = true;
      try {
        const summary = await getMatchPredictions(this.selectedMatchId);
        this.entries = Array.isArray(summary.Entries) ? summary.Entries : [];
        this.odds = summary.Odds || null;
        this.myPrediction = summary.MyPrediction || null;
      } catch (error) {
        this.entries = [];
        this.odds = null;
        this.myPrediction = null;
      } finally {
        this.isLoadingPredictions = false;
      }
    },
    async predict(teamId) {
      if (!this.selectedMatchId || this.isSubmitting) {
        return;
      }
      this.isSubmitting = true;
      this.predictionMessage = '';
      try {
        const summary = await predictForMatch(this.selectedMatchId, teamId);
        this.entries = Array.isArray(summary.Entries) ? summary.Entries : [];
        this.odds = summary.Odds || null;
        this.myPrediction = summary.MyPrediction || null;
      } catch (error) {
        this.predictionMessage = error?.response?.data?.error || 'Failed to save your prediction. Please try again.';
        this.predictionMessageType = 'error';
        // A rejection here usually means the window just closed or this
        // view is stale — reload either way, the same reasoning the sign-up
        // panel already follows on its own 409s.
        await this.loadSelectedPredictions();
      } finally {
        this.isSubmitting = false;
      }
    }
  }
};
</script>

<style scoped>
.predictions-section {
  padding: 1.5rem 0;
}

.predictions-section .container {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.section-title {
  margin: 0 0 1rem;
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--text-primary);
}

.match-pick-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.match-pick-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.6rem 0.85rem;
  border-radius: var(--border-radius);
  background-color: var(--bg-tertiary);
  border: 2px solid transparent;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.match-pick-item:hover {
  background-color: var(--bg-secondary);
}

.match-pick-item.active {
  border-color: var(--primary-color);
}

.match-pick-date {
  flex-shrink: 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
  min-width: 110px;
}

.match-pick-teams {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-weight: 600;
  color: var(--text-primary);
}

.match-pick-vs {
  color: var(--text-light);
  font-weight: 400;
  font-size: 0.8rem;
}

.widget-title {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--text-primary);
}

.widget-kickoff {
  margin: 0.2rem 0 1rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.prediction-closed-hint {
  margin: 0 0 1rem;
  font-size: 0.85rem;
  font-style: italic;
  color: var(--text-light);
}

.prediction-picks {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.prediction-pick-btn {
  flex: 1 1 150px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.3rem;
  padding: 0.85rem 0.5rem;
  border-radius: var(--border-radius);
  border: 2px solid var(--border-color);
  background-color: var(--bg-primary);
  color: var(--text-primary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.prediction-pick-btn:hover:not(:disabled) {
  border-color: var(--primary-color);
}

.prediction-pick-btn.selected {
  border-color: var(--primary-color);
  background-color: var(--bg-tertiary);
}

.prediction-pick-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.pick-label {
  font-weight: 600;
}

.pick-odds {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--primary-color);
}

.prediction-message {
  margin: 1rem 0 0;
  font-size: 0.85rem;
}

.prediction-message.error {
  color: var(--danger-color);
}

.prediction-entries {
  list-style: none;
  margin: 1rem 0 0;
  padding: 0.75rem 0 0;
  border-top: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.prediction-entry {
  display: flex;
  justify-content: space-between;
  font-size: 0.85rem;
}

.prediction-entry-name {
  color: var(--text-primary);
}

.prediction-entry-pick {
  color: var(--text-secondary);
  font-weight: 600;
}

@media (max-width: 768px) {
  .match-pick-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.3rem;
  }

  .prediction-picks {
    flex-direction: column;
  }
}
</style>
