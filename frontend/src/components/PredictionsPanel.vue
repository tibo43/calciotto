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
                  <span v-if="isSettled(match)" class="card-score">{{ match.Teams[0].Score }} – {{ match.Teams[1].Score }}</span>
                </div>

                <!-- The bet itself lives on the card: one row per outcome
                     with its odds. Never a disabled <button> — a disabled
                     button swallows the click, and tapping a closed match's
                     rows should still select it — so a closed match's rows
                     are only styled as locked (aria-disabled). Once
                     settled, the winning row gets a check and the caller's
                     own pick turns green or red. -->
                <div class="card-picks">
                  <button v-for="option in pickOptionsFor(match)" :key="option.key" type="button"
                    class="prediction-pick-btn" :class="pickClassesFor(match, option.teamId)"
                    :aria-disabled="(!predictionsAreOpenFor(match)).toString()" :aria-pressed="isMyPickFor(match, option.teamId).toString()"
                    @click.stop="onPick(match, option.teamId)">
                    <span class="pick-label">
                      <svg v-if="isWinningPickFor(match, option.teamId)" class="pick-winner-icon" viewBox="0 0 24 24"
                        fill="none" stroke="currentColor" stroke-width="3" aria-label="Result">
                        <polyline points="20 6 9 17 4 12" />
                      </svg>
                      <span v-else-if="option.colour" class="team-color-horizontal"
                        :style="{ backgroundColor: getTeamColor(option.colour) }"></span>
                      <span class="pick-name">{{ option.label }}</span>
                    </span>
                    <span v-if="oddsFor(match)" class="pick-odds">{{ formatOdds(option.odds) }}</span>
                  </button>
                </div>
              </div>
            </div>

            <!-- Scroll indicators — hidden rather than just disabled when
                 there is nothing to scroll to: they sit on top of the first
                 and last cards, whose rows are now bet buttons. -->
            <button v-show="canScrollLeft" class="scroll-btn scroll-left" @click="scrollLeft">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="15,18 9,12 15,6" />
              </svg>
            </button>
            <button v-show="canScrollRight" class="scroll-btn scroll-right" @click="scrollRight">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="9,18 15,12 9,6" />
              </svg>
            </button>
          </div>

          <!-- Selected Match Details -->
          <transition name="match-details">
            <div v-if="selectedMatch" class="match-details-container card-base">
              <!-- Everything about the bet itself (state, score, the caller's
                   pick and result) is on the match card above; this card
                   only adds the rosters and everyone's picks. -->
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

    <!-- A refused bet (kick-off passed while the page was open, network
         failure...) would otherwise fail silently: the shared toast, at the
         bottom right, closing on its own after a few seconds or from its
         cross. -->
    <transition name="prediction-toast">
      <div v-if="predictionMessage" class="toast-container toast-bottom" role="status" aria-live="polite">
        <div class="toast-message has-close" :class="predictionMessageType">
          <span>{{ predictionMessage }}</span>
          <button type="button" class="toast-close" aria-label="Dismiss" @click="dismissToast">×</button>
        </div>
      </div>
    </transition>
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
  matchWinnerId,
  rosterFullyComposed,
  predictionsAreOpen,
  predictionDisplayState,
  predictionDisplayLabel,
  settlePrediction,
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

// Long enough to read a sentence, short enough not to linger.
const TOAST_DURATION_MS = 5000;

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
      // GET /matches/:id/predictions per eligible match, keyed by match id:
      // every card needs its own odds and the caller's pick, not only the
      // selected one.
      summaries: {},
      isLoadingMatches: false,
      submittingMatchId: '',
      predictionMessage: '',
      predictionMessageType: 'error',
      toastTimeout: null,
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
    selectedSummary() {
      return this.summaries[this.selectedMatchId] || {};
    },
    entries() {
      return Array.isArray(this.selectedSummary.Entries) ? this.selectedSummary.Entries : [];
    },
    predictionCount() {
      return this.selectedSummary.Count || this.entries.length;
    },
    revealed() {
      return Boolean(this.selectedSummary.Revealed);
    },
    odds() {
      return this.selectedSummary.Odds || null;
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
    clearTimeout(this.toastTimeout);
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
    oddsFor(match) {
      return this.summaries[match.ID]?.Odds || null;
    },
    myPickFor(match) {
      return this.summaries[match.ID]?.MyPrediction || null;
    },
    predictionsAreOpenFor(match) {
      return predictionsAreOpen(match, this.nowMs);
    },
    // A card's three rows, in display order: team A, draw, team B.
    pickOptionsFor(match) {
      const [teamA, teamB] = match.Teams;
      const odds = this.oddsFor(match) || {};
      return [
        { key: 'a', teamId: teamA.ID, label: teamA.Name, colour: teamA.Colour, odds: odds.TeamAOdds },
        { key: 'draw', teamId: null, label: 'Draw', colour: '', odds: odds.DrawOdds },
        { key: 'b', teamId: teamB.ID, label: teamB.Name, colour: teamB.Colour, odds: odds.TeamBOdds }
      ];
    },
    // null is both "the Draw row" and "no prediction made" — deliberately
    // the same comparison: with no MyPrediction at all, nothing matches (see
    // MatchPredictionSummary.MyPrediction on the backend for why the *outer*
    // pointer, not the inner one, carries "absent").
    isMyPickFor(match, teamId) {
      const mine = this.myPickFor(match);
      return Boolean(mine) && (mine.PredictedTeamID || null) === teamId;
    },
    isWinningPickFor(match, teamId) {
      return this.isSettled(match) && matchWinnerId(match) === teamId;
    },
    outcomeFor(match) {
      const mine = this.myPickFor(match);
      return mine ? settlePrediction(match, this.oddsFor(match), mine.PredictedTeamID) : null;
    },
    pickClassesFor(match, teamId) {
      const mine = this.isMyPickFor(match, teamId);
      const outcome = mine ? this.outcomeFor(match) : null;
      return {
        selected: mine,
        winner: this.isWinningPickFor(match, teamId),
        won: Boolean(outcome) && outcome.correct,
        lost: Boolean(outcome) && !outcome.correct,
        locked: !this.predictionsAreOpenFor(match) || this.submittingMatchId === match.ID
      };
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
        this.summaries = {};
        if (this.eligibleMatches.length > 0) {
          this.selectMatch(this.eligibleMatches[0].ID);
          await Promise.all(this.eligibleMatches.map(match => this.loadSummary(match.ID)));
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
    selectMatch(matchId) {
      if (this.selectedMatchId === matchId) {
        return;
      }
      this.selectedMatchId = matchId;
      this.showTeams = false;
      this.showEntries = true;
    },
    async loadSummary(matchId) {
      try {
        const summary = await getMatchPredictions(matchId);
        this.summaries = { ...this.summaries, [matchId]: summary || {} };
      } catch (error) {
        console.error('Error loading predictions:', error);
        this.summaries = { ...this.summaries, [matchId]: {} };
      }
    },
    // A tap on one of a card's rows always selects that match; it also
    // places (or changes) the bet when predictions are open.
    onPick(match, teamId) {
      this.selectMatch(match.ID);
      if (!this.predictionsAreOpenFor(match) || this.submittingMatchId) {
        return;
      }
      this.predict(match.ID, teamId);
    },
    async predict(matchId, teamId) {
      this.submittingMatchId = matchId;
      this.dismissToast();
      try {
        const summary = await predictForMatch(matchId, teamId);
        this.summaries = { ...this.summaries, [matchId]: summary || {} };
      } catch (error) {
        console.error('Error submitting prediction:', error);
        this.showToast(error?.response?.data?.error || 'Failed to save your prediction. Please try again.', 'error');
        // A rejection here usually means the window just closed or this
        // view is stale — reload either way, the same reasoning the sign-up
        // panel already follows on its own 409s.
        await this.loadSummary(matchId);
      } finally {
        this.submittingMatchId = '';
      }
    },
    showToast(message, type) {
      clearTimeout(this.toastTimeout);
      this.predictionMessage = message;
      this.predictionMessageType = type;
      this.toastTimeout = setTimeout(this.dismissToast, TOAST_DURATION_MS);
    },
    dismissToast() {
      clearTimeout(this.toastTimeout);
      this.toastTimeout = null;
      this.predictionMessage = '';
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
  justify-content: space-between;
  gap: 0.5rem;
  margin-bottom: 0.6rem;
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

/* The betting widget — same pill/row conventions as the sign-up panel's own
   state badge/detail/message, duplicated under prediction-specific class
   names since this isn't a sign-up. */

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

/* The three outcome rows inside a carousel card: name on the left, odds
   on the right, the same row rhythm as the Matches tab's team rows. */
.card-picks {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.prediction-pick-btn {
  width: 100%;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.35rem 0.55rem;
  border-radius: var(--border-radius);
  border: 1.5px solid var(--border-color);
  background-color: var(--bg-primary);
  color: var(--text-primary);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.prediction-pick-btn:not(.locked):hover {
  border-color: var(--primary-color);
}

.prediction-pick-btn.selected {
  border-color: var(--primary-color);
  background-color: var(--bg-tertiary);
}

.prediction-pick-btn.locked {
  cursor: default;
}

/* Once closed, only what matters keeps full strength: the winning row and
   the caller's own pick. */
.prediction-pick-btn.locked:not(.selected):not(.winner) {
  opacity: 0.6;
}

.prediction-pick-btn.winner {
  border-color: #10b981;
}

/* Fixed text colours on fixed light fills, like the state badges, so they
   stay readable in dark mode too. */
.prediction-pick-btn.won {
  border-color: #10b981;
  background-color: #d1fae5;
  color: #065f46;
}

.prediction-pick-btn.lost {
  border-color: var(--danger-color);
  background-color: #fee2e2;
  color: #991b1b;
}

.prediction-pick-btn.won .pick-odds,
.prediction-pick-btn.lost .pick-odds {
  color: inherit;
}

.pick-winner-icon {
  width: 12px;
  height: 12px;
  flex: 0 0 12px;
  color: #059669;
}

.prediction-pick-btn.won .pick-winner-icon {
  color: inherit;
}

.pick-label {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  min-width: 0;
  font-weight: 600;
}

.pick-label .team-color-horizontal {
  flex: 0 0 12px;
}

/* Draw has no swatch: indent it to line up with the team names. */
.pick-label:not(:has(.team-color-horizontal, .pick-winner-icon)) {
  padding-left: calc(12px + 0.4rem);
}

.pick-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pick-odds {
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--primary-color);
  font-variant-numeric: tabular-nums;
}

.card-score {
  font-weight: 700;
  color: var(--text-primary);
  font-size: 0.95rem;
  font-variant-numeric: tabular-nums;
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
  margin-bottom: 0.75rem;
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

.prediction-toast-leave-active {
  animation: toastSlideOut 0.3s cubic-bezier(0.4, 0, 1, 1) forwards;
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
