<template>
  <div class="matches-panel predictions-panel">
    <section class="matches-section">
      <div class="container">
        <!-- Loading State -->
        <div v-if="isLoadingMatches" class="loading-container">
          <div class="loading-spinner"></div>
          <p class="loading-text">Loading matches...</p>
        </div>

        <!-- Matches Layout — same carousel + "Selected Match Details" pattern
             as MatchesPanel.vue, scoped to matches a prediction can be made
             on (a scheduled match whose roster is fully composed, see
             rosterFullyComposed in matchPrediction.js) rather than every
             match in the group. -->
        <div v-else-if="eligibleMatches.length > 0" class="matches-layout">
          <div class="matches-bar-container card-base">
            <div class="matches-bar hide-scrollbar" ref="matchesBar">
              <div v-for="match in eligibleMatches" :key="match.ID" class="match-card-horizontal scheduled"
                :class="{ active: selectedMatchId === match.ID }" @click="selectMatch(match.ID)">
                <div class="match-date-horizontal">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
                    <line x1="16" y1="2" x2="16" y2="6" />
                    <line x1="8" y1="2" x2="8" y2="6" />
                    <line x1="3" y1="10" x2="21" y2="10" />
                  </svg>
                  <span>{{ formatDateTimeShort(match.ScheduledAt) }}</span>
                </div>

                <div class="match-prediction-state-horizontal">
                  <span class="prediction-state-badge" :class="displayState(match)">{{ displayLabel(match) }}</span>
                </div>

                <div class="teams-horizontal">
                  <div v-for="(team, index) in match.Teams" :key="team.ID" class="team-horizontal">
                    <div class="team-info-horizontal">
                      <div class="team-color-horizontal" :style="{ backgroundColor: getTeamColor(team.Colour) }"></div>
                      <span class="team-name-horizontal">{{ team.Name }}</span>
                    </div>
                    <div v-if="isSettled(match)" class="team-score-horizontal">{{ team.Score }}</div>
                    <div v-if="index < match.Teams.length - 1" class="vs-separator">vs</div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Scroll indicators -->
            <button class="scroll-btn scroll-left" @click="scrollLeft" :disabled="!canScrollLeft">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="15,18 9,12 15,6" />
              </svg>
            </button>
            <button class="scroll-btn scroll-right" @click="scrollRight" :disabled="!canScrollRight">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="9,18 15,12 9,6" />
              </svg>
            </button>
          </div>

          <!-- Selected Match Details -->
          <transition name="match-details">
            <div v-if="selectedMatch" class="match-details-container card-base">
              <div class="details-header">
                <div class="details-title-section">
                  <h3>{{ formatDateTimeShort(selectedMatch.ScheduledAt) }} - Match Prediction</h3>
                </div>
                <div class="details-divider"></div>
              </div>

              <!-- The betting widget — state badge + 3-way odds buttons, same
                   visual language as the sign-up panel's own state badge
                   elsewhere in this app, duplicated rather than shared. -->
              <div class="prediction-inline">
                <div class="prediction-inline-top">
                  <span class="prediction-state-badge" :class="displayState(selectedMatch)">{{ displayLabel(selectedMatch) }}</span>
                  <span v-if="isSettled(selectedMatch)" class="prediction-final-score">
                    {{ selectedMatch.Teams[0].Name }} {{ selectedMatch.Teams[0].Score }} – {{ selectedMatch.Teams[1].Score }} {{ selectedMatch.Teams[1].Name }}
                  </span>
                </div>
                <p v-if="predictionClosedReason" class="prediction-inline-detail">{{ predictionClosedReason }}</p>

                <div v-if="isLoadingPredictions" class="signup-loading">
                  <div class="loading-spinner-small"></div>
                  <span>Loading predictions...</span>
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
                  <p v-if="predictionMessage" class="prediction-inline-message" :class="predictionMessageType">{{ predictionMessage }}</p>
                </template>
              </div>

              <!-- The composed roster, same two-column layout as the Matches
                   tab's own team columns — just names, no goals or Man of the
                   Match star, since neither applies before a match that
                   hasn't been played. -->
              <div class="players-section">
                <div class="section-header">
                  <h4 class="signup-list-title">Teams</h4>
                  <button type="button" class="signup-toggle-btn" :aria-expanded="showTeams.toString()"
                    @click="showTeams = !showTeams">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
                      class="signup-toggle-chevron" :class="{ 'is-expanded': showTeams }">
                      <polyline points="6 9 12 15 18 9" />
                    </svg>
                    {{ showTeams ? 'Hide teams' : 'Show teams' }}
                  </button>
                </div>
                <div v-if="showTeams" class="teams-columns">
                  <div v-for="team in selectedMatch.Teams" :key="team.ID" class="team-column">
                    <div class="team-column-header">
                      <div class="team-color-small" :style="{ backgroundColor: getTeamColor(team.Colour) }"></div>
                      <span class="team-column-name">{{ team.Name }}</span>
                    </div>
                    <ul class="team-players-list">
                      <li v-for="player in team.Players" :key="player.ID || player.Name" class="team-player-row">
                        <div class="player-info">
                          <span class="player-name">{{ formatPlayerNameForDisplay(player.Name) }}</span>
                        </div>
                      </li>
                      <li v-if="!team.Players.length" class="empty-slot">No players yet</li>
                    </ul>
                  </div>
                </div>
              </div>

              <!-- Everyone else's pick for this match, reusing the sign-up
                   list's own visual language (title + count badge + list of
                   rows), the same convention the sign-up panel already uses
                   for its own confirmed/waiting lists. -->
              <div v-if="predictionCount > 0" class="signup-lists prediction-entries-section">
                <div class="signup-list">
                  <div class="section-header">
                    <h4 class="signup-list-title">
                      Predictions
                      <span class="count-badge">{{ predictionCount }}</span>
                    </h4>
                    <button type="button" class="signup-toggle-btn" :aria-expanded="showEntries.toString()"
                      @click="showEntries = !showEntries">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
                        class="signup-toggle-chevron" :class="{ 'is-expanded': showEntries }">
                        <polyline points="6 9 12 15 18 9" />
                      </svg>
                      {{ showEntries ? 'Hide predictions' : 'Show predictions' }}
                    </button>
                  </div>
                  <!-- Before kick-off the backend returns only the caller's own
                       pick (see MatchPredictionSummary.Revealed): nobody can
                       follow the crowd while the choice is still open. -->
                  <p v-if="showEntries && !revealed" class="prediction-hidden-hint">
                    Everyone's picks are revealed at kick-off.
                  </p>
                  <ul v-if="showEntries && entries.length > 0" class="signup-entries">
                    <li v-for="entry in entries" :key="entry.PlayerID" class="signup-entry"
                      :class="{ 'is-me': entry.PlayerID === currentPlayerId }">
                      <span class="signup-name">{{ formatPlayerNameForDisplay(entry.Name) }}</span>
                      <span v-if="entry.PlayerID === currentPlayerId" class="signup-you">you</span>
                      <span class="prediction-entry-pick">{{ pickLabel(entry.PredictedTeamID) }}</span>
                      <span v-if="entryOutcome(entry)" class="prediction-entry-points"
                        :class="entryOutcome(entry).correct ? 'won' : 'lost'">
                        {{ formatPoints(entryOutcome(entry).points) }}
                      </span>
                    </li>
                  </ul>
                </div>
              </div>
            </div>
          </transition>
        </div>

        <!-- Empty State -->
        <div v-else class="empty-state">
          <div class="empty-content">
            <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="9" />
              <circle cx="12" cy="12" r="5" />
              <circle cx="12" cy="12" r="1" fill="currentColor" stroke="none" />
            </svg>
            <h3 class="empty-title">Nothing to predict yet</h3>
            <p class="empty-description">Once a scheduled match's teams are composed, it shows up here to predict before kick-off.</p>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
// The Pronostic page's actual content (see Predictions.vue for the
// context-bar shell around it): the match carousel, the per-match betting
// widget, and the season leaderboard. Deliberately mirrors MatchesPanel.vue's
// own carousel + "Selected Match Details" structure (same card-base/
// match-card-horizontal/match-details-container classes), so the two pages
// read as the same app rather than two different ones — the content inside
// the selected-match card is what actually differs (a betting widget instead
// of goal editing/Man of the Match voting).
import {
  getMatchesDetails,
  getMatchPredictions,
  predictForMatch,
  getToken
} from '@/services/api';
import { formatDateTimeShort } from '@/services/datetime';
import { isScheduledMatch } from '@/services/matchRegistration';
import {
  rosterFullyComposed,
  predictionsAreOpen,
  predictionDisplayState,
  predictionDisplayLabel,
  settlePrediction,
  PREDICTIONS_AWAITING_RESULT,
  PREDICTIONS_SETTLED
} from '@/services/matchPrediction';

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
      selectedMatchId: '',
      entries: [],
      predictionCount: 0,
      revealed: false,
      odds: null,
      myPrediction: null,
      isLoadingMatches: false,
      isLoadingPredictions: false,
      isSubmitting: false,
      predictionMessage: '',
      predictionMessageType: 'info',
      currentPlayerId: '',
      canScrollLeft: false,
      canScrollRight: false,
      // Collapsible sections of the selected match, reset on every
      // selection like MatchesPanel.vue's isSelectedMatchExpanded: the
      // picks come first, the teams are one tap away, and everyone's
      // predictions are shown since that is what people come back to see.
      showTeams: false,
      showEntries: true,
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
      if (!this.selectedMatch || this.predictionsOpen) {
        return '';
      }
      switch (this.displayState(this.selectedMatch)) {
        case PREDICTIONS_AWAITING_RESULT:
          return 'Kick-off has passed — points are counted once an admin saves the score.';
        case PREDICTIONS_SETTLED:
          return '';
        default:
          return "Predictions aren't open for this match yet.";
      }
    }
  },
  watch: {
    // The season selector live-updates without a page reload (unlike the
    // active group, which only ever changes through one — see
    // activeGroup.js), so this is the one prop this panel actually needs to
    // react to, the same convention MatchesPanel.vue's own season watch
    // follows.
    season() {
      this.loadMatches();
    }
  },
  created() {
    this.currentPlayerId = currentPlayerIdFromToken();
    this.loadMatches();
  },
  mounted() {
    // loadMatches() (started in created()) is still pending here — see
    // MatchesPanel.vue's identical comment on its own mounted() hook for why
    // this call alone can't do anything useful yet, and why
    // attachScrollListener() is called again once loadMatches() actually
    // populates the list.
    this.attachScrollListener();
  },
  beforeUnmount() {
    if (this.$refs.matchesBar) {
      this.$refs.matchesBar.removeEventListener('scroll', this.updateScrollButtons);
    }
  },
  methods: {
    formatDateTimeShort,
    displayState(match) {
      return predictionDisplayState(match, this.nowMs);
    },
    displayLabel(match) {
      return predictionDisplayLabel(this.displayState(match));
    },
    isSettled(match) {
      return this.displayState(match) === PREDICTIONS_SETTLED;
    },
    entryOutcome(entry) {
      return settlePrediction(this.selectedMatch, this.odds, entry.PredictedTeamID);
    },
    formatPoints(points) {
      return `${points > 0 ? '+' : ''}${points.toFixed(2)}`;
    },
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
      this.selectedMatchId = '';
      try {
        const data = await getMatchesDetails(this.activeGroupId, this.season);
        this.matches = Array.isArray(data) ? data : [];
        if (this.eligibleMatches.length > 0) {
          await this.selectMatch(this.eligibleMatches[0].ID);
        }
      } catch (error) {
        console.error('Error fetching matches:', error);
        this.matches = [];
      } finally {
        this.isLoadingMatches = false;
      }
      // matchesBar only exists in the DOM once eligibleMatches.length > 0 —
      // (re)attaching here, after the list actually renders, is what makes
      // the scroll buttons work on first load and after a season change, the
      // same reasoning MatchesPanel.vue's own loadMatches() follows.
      this.attachScrollListener();
    },
    async selectMatch(matchId) {
      this.selectedMatchId = matchId;
      this.predictionMessage = '';
      this.showTeams = false;
      this.showEntries = true;
      await this.loadSelectedPredictions();
    },
    applySummary(summary) {
      this.entries = Array.isArray(summary.Entries) ? summary.Entries : [];
      this.predictionCount = summary.Count || this.entries.length;
      this.revealed = Boolean(summary.Revealed);
      this.odds = summary.Odds || null;
      this.myPrediction = summary.MyPrediction || null;
    },
    async loadSelectedPredictions() {
      if (!this.selectedMatchId) {
        return;
      }
      this.isLoadingPredictions = true;
      try {
        const summary = await getMatchPredictions(this.selectedMatchId);
        this.applySummary(summary);
      } catch (error) {
        console.error('Error loading predictions:', error);
        this.applySummary({});
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
        this.applySummary(summary);
      } catch (error) {
        console.error('Error submitting prediction:', error);
        this.predictionMessage = error?.response?.data?.error || 'Failed to save your prediction. Please try again.';
        this.predictionMessageType = 'error';
        // A rejection here usually means the window just closed or this
        // view is stale — reload either way, the same reasoning the sign-up
        // panel already follows on its own 409s.
        await this.loadSelectedPredictions();
      } finally {
        this.isSubmitting = false;
      }
    },
    scrollLeft() {
      if (this.$refs.matchesBar) {
        this.$refs.matchesBar.scrollBy({ left: -300, behavior: 'smooth' });
      }
    },
    scrollRight() {
      if (this.$refs.matchesBar) {
        this.$refs.matchesBar.scrollBy({ left: 300, behavior: 'smooth' });
      }
    },
    updateScrollButtons() {
      if (this.$refs.matchesBar) {
        const element = this.$refs.matchesBar;
        this.canScrollLeft = element.scrollLeft > 0;
        this.canScrollRight = element.scrollLeft < (element.scrollWidth - element.clientWidth);
      }
    },
    attachScrollListener() {
      this.$nextTick(() => {
        this.updateScrollButtons();
        if (this.$refs.matchesBar) {
          this.$refs.matchesBar.addEventListener('scroll', this.updateScrollButtons);
        }
      });
    }
  }
};
</script>

<style scoped>
/* Carousel + selected-match-details layout, duplicated from MatchesPanel.vue
   rather than shared (this codebase's own convention — see getTeamColor()) —
   same class names so the two pages genuinely look like the same app. */
.matches-panel {
  background-color: var(--bg-secondary);
}

.matches-section {
  padding: 2rem 0;
}

.matches-layout {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.matches-bar-container {
  position: relative;
  overflow: hidden;
  padding: 0.5rem;
}

.matches-bar {
  display: flex;
  gap: 0.75rem;
  padding: 0.5rem;
  overflow-x: auto;
  scroll-behavior: smooth;
}

.match-card-horizontal {
  flex: 0 0 220px;
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  padding: 0.65rem;
  cursor: pointer;
  transition: all var(--transition-smooth);
  border: 2px solid transparent;
}

.match-card-horizontal:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
  background-color: var(--bg-primary);
}

.match-card-horizontal.active {
  border-color: var(--primary-color);
  background-color: var(--bg-primary);
  box-shadow: var(--shadow-lg);
}

.match-card-horizontal.scheduled {
  border-left-color: var(--primary-color);
}

.match-card-horizontal.scheduled .match-date-horizontal {
  flex-wrap: wrap;
  margin-bottom: 0.4rem;
}

.match-date-horizontal {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--text-secondary);
  font-weight: 500;
  margin-bottom: 0.75rem;
  font-size: 0.875rem;
}

.match-date-horizontal svg {
  width: 14px;
  height: 14px;
}

.match-prediction-state-horizontal {
  display: flex;
  align-items: center;
  margin-bottom: 0.6rem;
}

.teams-horizontal {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.team-horizontal {
  display: flex;
  align-items: center;
  justify-content: space-between;
  position: relative;
}

.team-info-horizontal {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

/* Outlined in the text colour, like the team columns' own .team-color-small
   (global-styles.css): a white team's swatch is otherwise invisible on a
   light card. */
.team-color-horizontal {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 1px solid currentColor;
  color: var(--text-secondary);
  box-shadow: var(--shadow-sm);
}

.team-score-horizontal {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--primary-color);
  background-color: var(--bg-primary);
  padding: 0.125rem 0.5rem;
  border-radius: 6px;
  min-width: 30px;
  text-align: center;
}

.team-name-horizontal {
  font-weight: 600;
  color: var(--text-primary);
  text-transform: capitalize;
  font-size: 0.875rem;
}

.vs-separator {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  top: -0.25rem;
  font-size: 0.75rem;
  color: var(--text-light);
  font-weight: 500;
}

.scroll-btn {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  background-color: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 50%;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
  color: var(--text-secondary);
  box-shadow: var(--shadow-md);
  z-index: 10;
}

.scroll-btn:hover:not(:disabled) {
  background-color: var(--primary-color);
  color: white;
  transform: translateY(-50%) scale(1.1);
}

.scroll-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.scroll-left {
  left: 10px;
}

.scroll-right {
  right: 10px;
}

.scroll-btn svg {
  width: 20px;
  height: 20px;
}

.match-details-container {
  overflow: hidden;
}

.details-header {
  margin-bottom: 1rem;
  flex-shrink: 0;
}

.details-header h3 {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 0.5rem;
}

.details-divider {
  height: 3px;
  background: linear-gradient(90deg, var(--primary-color), var(--secondary-color));
  border-radius: 2px;
  width: 80px;
}

.details-title-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

/* The betting widget — same pill/row conventions as the sign-up panel's own
   state badge/detail/message, duplicated under prediction-specific class
   names since this isn't a sign-up. */
.prediction-inline {
  margin-bottom: 1.5rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--border-color);
}

.prediction-inline-top {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.6rem;
}

.prediction-state-badge {
  padding: 0.3rem 0.65rem;
  border-radius: 20px;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  background-color: var(--bg-tertiary);
  color: var(--text-secondary);
  /* One line whatever the card width — a wrapped badge made one carousel
     card taller than its neighbours. */
  white-space: nowrap;
}

.prediction-state-badge.open {
  background-color: #d1fae5;
  color: #065f46;
}

/* Same amber as MatchesPanel.vue's "Upcoming": something is still to come. */
.prediction-state-badge.awaiting-result {
  background-color: #fef3c7;
  color: #92400e;
}

.prediction-state-badge.settled {
  background-color: #e5e7eb;
  color: #374151;
}

.prediction-final-score {
  font-weight: 700;
  color: var(--text-primary);
  font-size: 0.95rem;
}

.prediction-inline-detail {
  margin: 0.5rem 0 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.prediction-inline-message {
  margin: 0.6rem 0 0;
  font-size: 0.85rem;
}

.prediction-inline-message.success {
  color: #065f46;
}

.prediction-inline-message.error {
  color: var(--danger-color);
}

.prediction-picks {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
  margin-top: 1rem;
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
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pick-odds {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--primary-color);
}

.signup-loading {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.75rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

/* The composed roster — identical to MatchesPanel.vue's own team columns,
   minus the goal/own-goal/MOTM columns that don't apply before a match has
   been played. */
.players-section {
  overflow: hidden;
  display: flex;
  flex-direction: column;
  margin-bottom: 1.5rem;
}

.teams-columns {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
  align-items: start;
}

.team-column {
  background-color: var(--bg-primary);
  border-radius: var(--border-radius);
  box-shadow: var(--shadow-sm);
  overflow: hidden;
  min-width: 0;
}

.team-column-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem;
  background-color: var(--bg-tertiary);
  border-bottom: 2px solid var(--border-color);
  font-weight: 600;
  color: var(--text-primary);
}

.team-column-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.team-players-list {
  list-style: none;
  margin: 0;
  padding: 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.team-player-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 0.5rem;
  border-radius: var(--border-radius);
  transition: background-color var(--transition-fast);
}

.team-player-row:hover {
  background-color: var(--bg-secondary);
}

.team-player-row .player-info {
  flex: 1 1 auto;
  min-width: 0;
}

.team-player-row .player-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.team-players-list .empty-slot {
  padding: 0.5rem;
  text-align: center;
  color: var(--text-light);
  font-style: italic;
}

/* Other members' picks — reusing the sign-up list's own
   title/count-badge/entries visual language (see MatchesPanel.vue), just one
   list instead of a confirmed/waiting pair. */
.prediction-entries-section {
  grid-template-columns: 1fr;
}

.signup-list-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  font-size: 0.9rem;
}

/* A section title with its collapse toggle on the right. */
.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

/* Same text-and-chevron toggle as MatchesPanel.vue's sign-up details,
   duplicated per this codebase's convention. */
.signup-toggle-btn {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  padding: 0.4rem 0.5rem;
  border-radius: var(--border-radius);
  transition: all var(--transition-fast);
}

.signup-toggle-btn:hover {
  color: var(--primary-color);
  background-color: var(--bg-tertiary);
}

.signup-toggle-chevron {
  width: 14px;
  height: 14px;
  transition: transform var(--transition-fast);
}

.signup-toggle-chevron.is-expanded {
  transform: rotate(180deg);
}

.count-badge {
  background-color: var(--primary-color);
  color: white;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.25rem 0.5rem;
  border-radius: 12px;
  min-width: 20px;
  text-align: center;
}

.signup-entries {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.signup-entry {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  padding: 0.5rem 0.65rem;
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  font-size: 0.875rem;
}

.signup-entry.is-me {
  border: 1px solid var(--primary-color);
}

.signup-name {
  flex: 1;
  color: var(--text-primary);
  font-weight: 500;
}

.prediction-entry-pick {
  color: var(--text-secondary);
  font-weight: 600;
}

.prediction-hidden-hint {
  margin: 0 0 0.5rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
  font-style: italic;
}

.prediction-entry-points {
  min-width: 3.5rem;
  text-align: right;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.prediction-entry-points.won {
  color: #059669;
}

.prediction-entry-points.lost {
  color: var(--danger-color);
}

.signup-you {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--primary-color);
}

.match-details-enter-active,
.match-details-leave-active {
  transition: all var(--transition-smooth);
}

.match-details-enter-from,
.match-details-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

@media (max-width: 768px) {
  .matches-section {
    padding: 1rem 0;
  }

  .matches-layout {
    gap: 0.75rem;
  }

  .matches-bar-container {
    padding: 0.35rem;
  }

  .matches-bar {
    padding: 0.35rem;
    gap: 0.5rem;
  }

  .match-card-horizontal {
    flex: 0 0 200px;
    padding: 0.5rem;
  }

  .match-details-container {
    padding: 1rem;
  }

  .details-header h3 {
    font-size: 1.1rem;
  }

  .teams-columns {
    grid-template-columns: 1fr;
    gap: 0.5rem;
  }

  .team-column-header {
    padding: 0.5rem;
    gap: 0.4rem;
    font-size: 0.8rem;
  }

  .team-players-list {
    padding: 0.35rem;
    gap: 0.25rem;
  }

  .team-player-row {
    padding: 0.3rem 0.35rem;
    gap: 0.35rem;
  }

  .team-player-row .player-name {
    font-size: 0.85rem;
  }

  /* Three across on one line, compact: stacked, flex-basis became each
     button's height and the three of them filled a whole phone screen. */
  .prediction-picks {
    flex-wrap: nowrap;
    gap: 0.5rem;
  }

  .prediction-pick-btn {
    flex: 1 1 0;
    min-width: 0;
    padding: 0.6rem 0.35rem;
    gap: 0.15rem;
  }

  .pick-label {
    font-size: 0.85rem;
  }

  .scroll-btn {
    display: none;
  }
}

@media (max-width: 480px) {
  .match-card-horizontal {
    flex: 0 0 170px;
    padding: 0.5rem;
  }

  .match-details-container {
    padding: 0.75rem;
  }
}
</style>
