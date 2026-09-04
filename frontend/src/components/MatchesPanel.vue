<template>
  <div class="matches-panel">
    <!-- Create Match Modal -->
    <transition name="modal">
      <div v-if="showCreateModal" class="modal-overlay" @click="closeModal">
        <div class="modal-container" @click.stop>
          <div class="modal-header modal-header-gradient">
            <h3>Create New Match</h3>
            <button class="modal-close" @click="closeModal">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>

          <div class="modal-body modal-body-large">
            <div class="form-group">
              <label>Select Match Date</label>
              <div class="date-picker-container">
                <div class="date-picker-header">
                  <button type="button" @click="previousMonth" class="nav-btn">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="15,18 9,12 15,6" />
                    </svg>
                  </button>
                  <h4 class="month-year">{{ getCurrentMonthYear() }}</h4>
                  <button type="button" @click="nextMonth" class="nav-btn">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="9,18 15,12 9,6" />
                    </svg>
                  </button>
                </div>

                <div class="date-picker-grid">
                  <div class="day-headers">
                    <span v-for="day in dayHeaders" :key="day" class="day-header">{{ day }}</span>
                  </div>
                  <div class="days-grid">
                    <button v-for="date in calendarDays" :key="`${date.day}-${date.month}`" type="button"
                      class="day-button" :class="{
                        'other-month': date.isOtherMonth,
                        'selected': isSelectedDate(date),
                        'today': isToday(date),
                        'disabled': isPastDate(date)
                      }" :disabled="isPastDate(date)" @click="selectDate(date)">
                      {{ date.day }}
                    </button>
                  </div>
                </div>
              </div>
              <p v-if="dateError" class="error-message">{{ dateError }}</p>

              <div v-if="selectedDate" class="selected-date-display">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="20,6 9,17 4,12" />
                </svg>
                Selected: {{ formatSelectedDate(selectedDate) }}
              </div>
            </div>

            <!-- Optional scheduling. Unchecked (the default) the modal creates
                 an ordinary match exactly as it always has — no scheduling
                 field is sent at all. Checked, all three become required: the
                 backend treats them as all-or-nothing and 400s on a partial
                 set. -->
            <div class="form-group schedule-group">
              <label class="schedule-toggle">
                <input type="checkbox" class="schedule-checkbox" v-model="isScheduled" :disabled="isCreating" />
                <span>Schedule this match and open sign-ups</span>
              </label>

              <div v-if="isScheduled" class="schedule-fields">
                <div class="schedule-field">
                  <label for="schedule-kickoff-time">Kick-off time</label>
                  <input id="schedule-kickoff-time" class="form-input schedule-input" type="time" v-model="kickoffTime"
                    :disabled="isCreating" />
                  <p class="schedule-hint">On the day selected above.</p>
                </div>

                <div class="schedule-field">
                  <label for="schedule-registration-opens">Sign-ups open</label>
                  <input id="schedule-registration-opens" class="form-input schedule-input" type="datetime-local"
                    v-model="registrationOpensAt" :disabled="isCreating" />
                  <p class="schedule-hint">Must be strictly before kick-off.</p>
                </div>

                <div class="schedule-field">
                  <label for="schedule-max-players">Maximum players</label>
                  <input id="schedule-max-players" class="form-input schedule-input" type="number" min="1" step="1"
                    v-model="maxPlayers" :disabled="isCreating" />
                  <p class="schedule-hint">A calciotto is 8v8, so 16 by default.</p>
                </div>
              </div>

              <p v-if="scheduleError" class="error-message schedule-error">{{ scheduleError }}</p>
            </div>
          </div>

          <div class="modal-footer">
            <button class="btn-base btn-cancel" @click="closeModal" :disabled="isCreating">
              Cancel
            </button>
            <button class="btn-base btn-primary" @click="creatingMatch" :disabled="!selectedDate || isCreating"
              :class="{ 'loading': isCreating }">
              <div v-if="isCreating" class="loading-spinner-small"></div>
              <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="20,6 9,17 4,12" />
              </svg>
              {{ isCreating ? 'Creating...' : 'Create Match' }}
            </button>
          </div>
        </div>
      </div>
    </transition>

    <!-- Matches Section -->
    <section class="matches-section">
      <div class="container">
        <!-- Loading State -->
        <div v-if="isLoading" class="loading-container">
          <div class="loading-spinner"></div>
          <p class="loading-text">Loading matches...</p>
        </div>

        <!-- Matches Layout -->
        <div v-else-if="matches.length > 0" class="matches-layout">
          <!-- Horizontal Matches Bar -->
          <div class="matches-bar-container card-base">
            <div class="matches-bar hide-scrollbar" ref="matchesBar">
              <!-- Create Match — admin-only, integrated into the match list
                   itself (as a leading "+" card) rather than a separate
                   toolbar above it. -->
              <button v-if="isAdmin" class="match-card-horizontal add-match-card" @click="showCreateModal = true"
                aria-label="Create match">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19" />
                  <line x1="5" y1="12" x2="19" y2="12" />
                </svg>
              </button>

              <div v-for="match in matches" :key="match.ID" class="match-card-horizontal"
                :class="{ 'active': selectedMatch?.ID === match.ID, 'scheduled': isScheduledMatch(match) }"
                @click="selectMatch(match)">
                <!-- Match Date — a scheduled match shows its kick-off instead
                     (same calendar day, plus the time), so the card says when
                     the match actually starts rather than just which day it
                     is filed under. -->
                <div class="match-date-horizontal">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
                    <line x1="16" y1="2" x2="16" y2="6" />
                    <line x1="8" y1="2" x2="8" y2="6" />
                    <line x1="3" y1="10" x2="21" y2="10" />
                  </svg>
                  <span>{{ isScheduledMatch(match) ? formatKickoff(match) : formatDateShort(match.Date) }}</span>
                </div>

                <!-- Sign-ups line — one compact row (state badge + count),
                     not a panel: these cards are ~200px wide in a horizontal
                     carousel. The badge shows the actual registration state
                     (open/not-open-yet/closed), not just "this is a scheduled
                     match" — that part is already implied by the row existing
                     at all, so it would tell a browsing member nothing they
                     couldn't already see. Everything about the sign-up list
                     itself (names, Participate/Withdraw) still lives below,
                     once this card is selected, or on the match page. -->
                <div v-if="isScheduledMatch(match)" class="match-signups-horizontal">
                  <span class="signup-state-badge" :class="cardRegistrationState(match)">{{ cardRegistrationLabel(match) }}</span>
                  <span class="signup-count">{{ signupCountLabel(match) }}</span>
                </div>

                <!-- Teams and Scores — hidden until composed for a scheduled
                     match (see showTeamRoster): "0 vs 0" is noise for the
                     whole sign-up window, where the badge/count above is what
                     actually matters. -->
                <div v-if="showTeamRoster(match)" class="teams-horizontal">
                  <div v-for="(team, index) in match.Teams" :key="team.ID" class="team-horizontal">
                    <div class="team-info-horizontal">
                      <div class="team-color-horizontal" :style="{ backgroundColor: getTeamColor(team.Colour) }"></div>
                      <span class="team-name-horizontal">{{ team.Name }}</span>
                    </div>
                    <div class="team-score-horizontal">{{ team.Score }}</div>
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
                  <h3>{{ formatDate(selectedMatch.Date) }} - Match Details</h3>
                  <div class="details-header-actions">
                    <!-- Collapse/expand the sign-up chrome below, once the
                         roster is actually composed — see
                         canCollapseSelectedMatchSignup's own comment for why
                         this only ever appears then. Stays visible in both
                         states (it's what re-expands, too), and toggling
                         never touches the team columns themselves — those
                         are unaffected either way. -->
                    <button
                      v-if="canCollapseSelectedMatchSignup"
                      type="button"
                      class="signup-toggle-btn"
                      :aria-expanded="isSelectedMatchExpanded.toString()"
                      @click="isSelectedMatchExpanded = !isSelectedMatchExpanded"
                    >
                      <svg
                        viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
                        class="signup-toggle-chevron" :class="{ 'is-expanded': isSelectedMatchExpanded }"
                      >
                        <polyline points="6 9 12 15 18 9" />
                      </svg>
                      {{ isSelectedMatchExpanded ? 'Hide sign-up details' : 'Show sign-up details' }}
                    </button>
                    <!-- Admin-only now: MatchDetails.vue is gated by a router
                         guard (router/index.js's canEditMatch) to admins of
                         the match's own group, so a plain member following
                         this link would just bounce straight back here.
                         Hidden rather than relabelled "View Match" as it used
                         to be — everything a member needs (the full sign-up
                         list with names, Man of the Match voting) now lives
                         in this panel directly, so there is no reduced "view"
                         version of that page left to send them to. -->
                    <router-link
                      v-if="isAdmin"
                      :to="`/matches/${selectedMatch.ID}/edit`"
                      class="btn-base btn-primary btn-small edit-match-btn"
                    >
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                      </svg>
                      Edit Match
                    </router-link>
                  </div>
                </div>
                <div class="details-divider"></div>
              </div>

              <!-- Sign-ups, without leaving this page. This used to be
                   deliberately light — state, count, and Participate/Withdraw
                   only, with the full confirmed/waiting roster reserved for
                   the match page — but a plain member can no longer reach
                   that page at all (see the router guard above), so the full
                   named lists moved here too, reusing MatchDetails.vue's own
                   .signup-list* markup/classes rather than inventing new
                   ones.

                   Collapsed by default once the roster is composed
                   (showSignupInline / canCollapseSelectedMatchSignup) — once
                   teams exist the sign-up process is effectively finished,
                   so leading with team composition instead of sign-up chrome
                   is the more useful default; the toggle above brings this
                   back for whoever still wants it (a late sign-up change, an
                   admin reopening the list, etc). Before composition, or for
                   an unscheduled match, this always renders — there is
                   nothing to collapse when the sign-up info *is* the main
                   content. -->
              <div v-if="showSignupInline" class="signup-inline">
                <!-- Badge, count and the action button all on one row — moved
                     here from a separate row below after the badge/count
                     alone read as a status display with nothing to act on
                     right next to it. -->
                <div class="signup-inline-top">
                  <span class="signup-state-badge" :class="registrationState">{{ registrationStateLabel }}</span>
                  <span class="signup-count-inline">{{ signupCountLabel(selectedMatch) }}</span>
                  <div class="signup-inline-actions">
                    <button v-if="canParticipate" @click="participate" :disabled="isUpdatingRegistration"
                      class="btn-base btn-primary btn-small">
                      {{ isUpdatingRegistration ? 'Signing up...' : 'Participate' }}
                    </button>
                    <button v-if="canWithdraw" @click="confirmWithdraw" :disabled="isUpdatingRegistration"
                      class="btn-base btn-cancel btn-small">
                      {{ isUpdatingRegistration ? 'Withdrawing...' : 'Withdraw' }}
                    </button>
                  </div>
                </div>
                <p v-if="registrationStateDetail" class="signup-inline-detail">{{ registrationStateDetail }}</p>
                <p v-if="signupMessage" class="signup-inline-message" :class="signupMessageType">{{ signupMessage }}</p>

                <!-- Same confirmed/waiting split as MatchDetails.vue's own
                     .signup-panel: server-derived IsWaiting, never guessed
                     here from Position vs. MaxPlayers. -->
                <div v-if="isLoadingRegistrations" class="signup-loading">
                  <div class="loading-spinner-small"></div>
                  <span>Loading sign-ups...</span>
                </div>
                <div v-else class="signup-lists">
                  <div class="signup-list">
                    <h4 class="signup-list-title">
                      Confirmed
                      <span class="count-badge">{{ confirmedRegistrations.length }} / {{ selectedMatch.MaxPlayers }}</span>
                    </h4>
                    <ul class="signup-entries">
                      <li v-for="entry in confirmedRegistrations" :key="entry.PlayerID" class="signup-entry"
                        :class="{ 'is-me': entry.PlayerID === currentPlayerId }">
                        <span class="signup-position">{{ entry.Position }}</span>
                        <span class="signup-name">{{ formatPlayerNameForDisplay(entry.Name) }}</span>
                        <span v-if="entry.PlayerID === currentPlayerId" class="signup-you">you</span>
                      </li>
                      <li v-if="confirmedRegistrations.length === 0" class="signup-empty">Nobody has signed up yet</li>
                    </ul>
                  </div>

                  <div v-if="waitingRegistrations.length > 0" class="signup-list signup-list-waiting">
                    <h4 class="signup-list-title">
                      Waiting list
                      <span class="count-badge">{{ waitingRegistrations.length }}</span>
                    </h4>
                    <ul class="signup-entries">
                      <li v-for="entry in waitingRegistrations" :key="entry.PlayerID" class="signup-entry"
                        :class="{ 'is-me': entry.PlayerID === currentPlayerId }">
                        <span class="signup-position">{{ entry.Position }}</span>
                        <span class="signup-name">{{ formatPlayerNameForDisplay(entry.Name) }}</span>
                        <span v-if="entry.PlayerID === currentPlayerId" class="signup-you">you</span>
                      </li>
                    </ul>
                  </div>
                </div>
              </div>

              <div v-if="showTeamRoster(selectedMatch)" class="players-section">
                <!-- Each team lists its own scorers — a grid column per team
                     (stacked on mobile, see below) rather than a shared table
                     with one row per player-index. The old table forced both
                     team columns onto one row (200px min-width each), which
                     on a narrow phone meant scrolling right just to see the
                     other team; it also visually paired team A's Nth player
                     with team B's Nth player, two players with no actual
                     relationship to each other. -->
                <div class="teams-columns">
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
                        <!-- Man of the Match voting — moved here from
                             MatchDetails.vue entirely (see CLAUDE.md): a star
                             per candidate rather than a dropdown, right next
                             to the goal count, in its own fixed-width column
                             so the star lines up vertically across every row
                             regardless of whether the vote-count pill is
                             showing. Not admin-gated — voter eligibility is
                             broader than "played in this match" — and offered
                             for any composed roster, scheduled or not
                             (showTeamRoster already implies that). The
                             caller's own row still renders the button (so the
                             column stays aligned) but makes it
                             visibility:hidden via .is-self — the backend
                             rejects a self-vote outright, so there is nothing
                             to gain from offering it, but removing the
                             element entirely would shift every other row's
                             star out of its column.

                             Two separate stars, two separate questions: the
                             button below answers "did *I* vote for this
                             player" (filled amber only for the caller's own
                             choice) — it says nothing about who is actually
                             winning. isCurrentMotmLeader answers that other
                             question directly: a small gold star inside the
                             vote-count pill itself marks whoever currently
                             has the *most* votes (tie-inclusive, mirroring
                             the backend's ComputeMotmWinners), regardless of
                             whether the caller voted for them at all — real
                             feedback after a match where nobody had voted for
                             the only candidate with a vote, yet nothing on
                             screen indicated they were the (derived) match
                             winner. -->
                        <div class="player-motm">
                          <span v-if="motmVoteCount(player.ID) > 0" class="motm-vote-count"
                            :class="{ 'is-leader': isCurrentMotmLeader(player.ID) }">
                            <svg v-if="isCurrentMotmLeader(player.ID)" class="motm-leader-icon" viewBox="0 0 24 24"
                              fill="currentColor" aria-hidden="true">
                              <polygon
                                points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
                            </svg>
                            {{ motmVoteCount(player.ID) }} vote{{ motmVoteCount(player.ID) === 1 ? '' : 's' }}
                          </span>
                          <button v-if="player.ID" type="button"
                            class="motm-star-btn"
                            :class="{ 'is-voted': isMyMotmVote(player.ID), 'is-self': player.ID === currentPlayerId }"
                            :disabled="isUpdatingMotmVote || !motmVotingOpen || player.ID === currentPlayerId"
                            :title="motmStarTitle(player)"
                            :aria-label="motmStarTitle(player)" @click="toggleMotmVote(player.ID)">
                            <svg viewBox="0 0 24 24" :fill="isMyMotmVote(player.ID) ? 'currentColor' : 'none'"
                              stroke="currentColor" stroke-width="2">
                              <polygon
                                points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
                            </svg>
                          </button>
                        </div>
                        <span class="goal-badge">{{ player.GoalNumber || 0 }}</span>
                      </li>
                      <li v-if="!team.Players.length" class="empty-slot">No players yet</li>
                    </ul>
                  </div>
                </div>
                <p v-if="motmMessage" class="motm-inline-message" :class="motmMessageType">{{ motmMessage }}</p>
              </div>
              <!-- Same hidden-until-composed rule as the card above, spelled
                   out here since this preview otherwise has nothing else to
                   explain the gap where the roster would be. -->
              <p v-else-if="isScheduledMatch(selectedMatch)" class="no-roster-hint">
                Teams will appear here once sign-ups close and are composed.
              </p>
            </div>
          </transition>
        </div>

        <!-- Empty State -->
        <div v-else class="empty-state">
          <div class="empty-content">
            <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10" />
              <path d="M16 16s-1.5-2-4-2-4 2-4 2" />
              <line x1="9" y1="9" x2="9.01" y2="9" />
              <line x1="15" y1="9" x2="15.01" y2="9" />
            </svg>
            <h3 class="empty-title">No matches available</h3>
            <p class="empty-description">Create your first match to get started</p>
            <button v-if="isAdmin" class="btn-base btn-primary btn-large" @click="showCreateModal = true">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="12" y1="5" x2="12" y2="19" />
                <line x1="5" y1="12" x2="19" y2="12" />
              </svg>
              Create Your First Match
            </button>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
import {
  getMatchesDetails,
  createMatch,
  getMatchRegistrations,
  registerForMatch,
  unregisterFromMatch,
  voteForMotm,
  removeMotmVote,
  getMatchVotes,
  getToken
} from '@/services/api';
import { toLocalRFC3339, dateTimeLocalToRFC3339, formatDateTimeShort, formatCalendarDay, formatCalendarDayShort } from '@/services/datetime';
import {
  isScheduledMatch,
  deriveRegistrationState,
  registrationsAreOpen,
  registrationStateLabel,
  teamsAreComposed,
  REGISTRATION_NOT_OPEN_YET,
  REGISTRATION_CLOSED_BY_ADMIN,
  REGISTRATION_CLOSED_AT_KICKOFF
} from '@/services/matchRegistration';
import { isMotmVotingOpen } from '@/services/motmVoting';

// Same shape as MatchDetails.vue's and Profile.vue's own helper — the app has
// no auth store, and the player id is only ever needed to answer "which
// registration row is mine", so each page decodes the JWT payload locally
// rather than a third place holding state.
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

// The Matches sub-tab of MatchesAndStandings.vue: the match carousel, the
// selected match's preview, and the admin-only create-match modal. Everything
// it is scoped by — the active group, the caller's admin role, the selected
// season — is resolved once by the page and passed down, so this component
// only ever loads matches.
// A calciotto is 8v8, so 16 is the roster size worth defaulting to.
const DEFAULT_MAX_PLAYERS = 16;

export default {
  name: 'MatchesPanel',
  props: {
    // The group the list (and any match created from here) belongs to. Empty
    // means "let the backend pick the caller's first group", the same degraded
    // behaviour resolveActiveGroupId() falls back to.
    activeGroupId: {
      type: String,
      default: ''
    },
    // Whether the caller is an admin of that group — gates the "Create Match"
    // button, since POST /matches is admin-only on the backend
    // (RequireGroupAdmin in main.go).
    isAdmin: {
      type: Boolean,
      default: false
    },
    // Empty means every season, exactly like the standings endpoints.
    season: {
      type: String,
      default: ''
    },
    // A match id resolved from a shared `/m/:code` link (see
    // MatchesAndStandings.vue's resolveDeepLinkedMatch() and
    // router/index.js's `/m/:code` route). When present and found in this
    // list, loadMatches() auto-selects it instead of the newest match.
    // Empty is the ordinary case — no deep link at all.
    deepLinkMatchId: {
      type: String,
      default: ''
    }
  },
  data() {
    return {
      matches: [],
      selectedMatch: null,
      isLoading: true,
      canScrollLeft: false,
      canScrollRight: false,
      // Create Match Modal
      showCreateModal: false,
      selectedDate: '',
      isCreating: false,
      dateError: '',
      match: {},
      // Optional scheduling (opt-in). All four are reset by closeModal(), so
      // reopening the modal always starts from the unscheduled default.
      isScheduled: false,
      kickoffTime: '',
      registrationOpensAt: '',
      maxPlayers: DEFAULT_MAX_PLAYERS,
      scheduleError: '',
      // Custom Date Picker
      currentMonth: new Date().getMonth(),
      currentYear: new Date().getFullYear(),
      dayHeaders: ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su'],
      // --- Sign-ups, for whichever match is selected below the carousel ---
      // Sampled once, in created(), the same "no polling, re-read on
      // navigation" contract MatchDetails.vue uses for the same field.
      nowMs: 0,
      currentPlayerId: '',
      // The sign-up list of `selectedMatch` alone — never all matches at
      // once, which is why it's reloaded on every selection change rather
      // than kept per-match. Only used to answer "am I registered", not
      // rendered as a roster (that stays on the match page).
      registrations: [],
      isLoadingRegistrations: false,
      isUpdatingRegistration: false,
      signupMessage: '',
      signupMessageType: 'success',
      // Whether the full sign-up chrome (state badge, count,
      // Participate/Withdraw, confirmed/waiting named lists) is expanded for
      // `selectedMatch`, once its roster is composed — see
      // canCollapseSelectedMatchSignup. Reset to the collapsed default
      // whenever the selection changes (selectMatch()/loadMatches()); this
      // value is only ever consulted through showSignupInline, which ignores
      // it entirely for a match that can't be collapsed in the first place.
      isSelectedMatchExpanded: false,
      // --- Man of the Match voting, for whichever match is selected ---
      // Moved here from MatchDetails.vue in full — see CLAUDE.md. Same
      // "always fully re-fetch after a change" contract as the sign-up list
      // above: the tally is server-derived, so patching it locally is how the
      // display drifts from reality.
      motmVotes: { Tally: [], MyVoteFor: null },
      isLoadingMotmVotes: false,
      isUpdatingMotmVote: false,
      motmMessage: '',
      motmMessageType: 'success'
    };
  },
  async created() {
    this.nowMs = Date.now();
    this.currentPlayerId = currentPlayerIdFromToken();
    await this.loadMatches();
  },
  watch: {
    // The page's season selector is shared with the standings tabs, so a
    // change there has to re-scope this list too.
    season() {
      this.loadMatches();
    }
  },
  mounted() {
    // loadMatches() (started in created()) is still pending at this point —
    // matchesBar doesn't exist yet since matches.length is still 0, so this
    // alone can't compute canScrollLeft/canScrollRight or attach the scroll
    // listener. attachScrollListener() runs again once loadMatches()
    // actually populates matches (see there) — this call only covers the
    // rare case where matches was already loaded by the time this mounts.
    this.attachScrollListener();
  },
  beforeUnmount() {
    if (this.$refs.matchesBar) {
      this.$refs.matchesBar.removeEventListener('scroll', this.updateScrollButtons);
    }
  },
  computed: {
    calendarDays() {
      const days = [];
      const firstDay = new Date(this.currentYear, this.currentMonth, 1);
      const startDate = new Date(firstDay);

      // Calculate days to go back to get Monday as first day
      const dayOfWeek = firstDay.getDay(); // 0 = Sunday, 1 = Monday, etc.
      const daysToGoBack = dayOfWeek === 0 ? 6 : dayOfWeek - 1; // If Sunday (0), go back 6 days, else go back (dayOfWeek - 1)

      startDate.setDate(startDate.getDate() - daysToGoBack);

      // Generate 42 days (6 weeks)
      for (let i = 0; i < 42; i++) {
        const date = new Date(startDate);
        date.setDate(startDate.getDate() + i);

        days.push({
          day: date.getDate(),
          month: date.getMonth(),
          year: date.getFullYear(),
          isOtherMonth: date.getMonth() !== this.currentMonth
        });
      }

      return days;
    },

    // The state and copy below mirror MatchDetails.vue's own
    // registrationState/registrationStateLabel/registrationStateDetail
    // exactly — same feature, same words, whichever page a player reaches it
    // from. Unlike that page, there is no separate "registrationsClosedAt"
    // local field to keep in sync with `match`: this panel has no editable
    // state of its own to protect from a false-dirty flag, so
    // RegistrationsClosedAt is read straight off `selectedMatch`.
    registrationState() {
      if (!this.selectedMatch) return '';
      return deriveRegistrationState(this.selectedMatch, this.nowMs);
    },

    registrationsOpen() {
      return registrationsAreOpen(this.registrationState);
    },

    registrationStateLabel() {
      return registrationStateLabel(this.registrationState);
    },

    registrationStateDetail() {
      switch (this.registrationState) {
        case REGISTRATION_NOT_OPEN_YET:
          return `Sign-ups open on ${formatDateTimeShort(this.selectedMatch.RegistrationOpensAt)}.`;
        case REGISTRATION_CLOSED_BY_ADMIN:
          return 'An admin has closed sign-ups for this match.';
        case REGISTRATION_CLOSED_AT_KICKOFF:
          return 'Kick-off has passed, so sign-ups are closed.';
        default:
          return '';
      }
    },

    isRegistered() {
      if (!this.currentPlayerId) return false;
      return this.registrations.some(entry => entry.PlayerID === this.currentPlayerId);
    },

    canParticipate() {
      return this.registrationsOpen && !this.isLoadingRegistrations && !this.isUpdatingRegistration && !this.isRegistered;
    },

    canWithdraw() {
      return this.registrationsOpen && !this.isLoadingRegistrations && !this.isUpdatingRegistration && this.isRegistered;
    },

    // The confirmed/waiting split, same as MatchDetails.vue's own computed
    // pair: server-derived IsWaiting, never re-derived here from Position vs.
    // MaxPlayers (the two would disagree the moment an admin changes the cap).
    confirmedRegistrations() {
      return this.registrations.filter(entry => !entry.IsWaiting);
    },

    waitingRegistrations() {
      return this.registrations.filter(entry => entry.IsWaiting);
    },

    // Whether a Man of the Match vote can still be cast/changed/removed for
    // `selectedMatch` at this page's one sampled `nowMs` — see
    // services/motmVoting.js. The backend re-checks this on every call
    // regardless; this only lets a star grey out with a tooltip before a 409.
    motmVotingOpen() {
      return isMotmVotingOpen(this.selectedMatch, this.nowMs);
    },

    // Whether the toggle button (and therefore the collapse behaviour
    // itself) applies to `selectedMatch` at all: only once it's both
    // scheduled and its roster is composed — before that, the sign-up info
    // *is* the main content, so there's nothing to lead with instead.
    canCollapseSelectedMatchSignup() {
      return Boolean(this.selectedMatch)
        && isScheduledMatch(this.selectedMatch)
        && teamsAreComposed(this.selectedMatch);
    },

    // Gates the sign-up chrome block itself. A match that can't collapse
    // (unscheduled, or scheduled but not yet composed) always shows it, same
    // as before this feature existed; a composed scheduled match shows it
    // only while explicitly expanded.
    showSignupInline() {
      if (!this.selectedMatch || !isScheduledMatch(this.selectedMatch)) return false;
      return !this.canCollapseSelectedMatchSignup || this.isSelectedMatchExpanded;
    }
  },
  methods: {
    async loadMatches() {
      this.isLoading = true;
      this.selectedMatch = null;
      try {
        const matches = await getMatchesDetails(this.activeGroupId, this.season);

        // Validate matches data
        if (!Array.isArray(matches)) {
          throw new Error('Invalid matches data');
        }

        matches.forEach(match => {
          if (!match.ID || !match.Date || !Array.isArray(match.Teams)) {
            console.error('Invalid match structure:', match);
            throw new Error('Invalid match structure');
          }
          match.Teams.forEach(team => {
            if (!team.ID || !team.Colour || !Array.isArray(team.Players)) {
              throw new Error('Invalid team structure');
            }
          });
        });

        this.matches = matches;

        // Auto-select the newest match — unless a deep-linked match id (a
        // shared /m/:code link, resolved one level up in
        // MatchesAndStandings.vue) is present and actually in this list, in
        // which case that one wins instead. Not found (wrong season somehow,
        // already gone) falls back to the same newest-match default.
        if (this.matches.length > 0) {
          const deepLinked = this.deepLinkMatchId
            ? this.matches.find(match => match.ID === this.deepLinkMatchId)
            : null;
          this.selectedMatch = deepLinked || this.matches[0];
          // Same reset as selectMatch() — a season change reloading a
          // different match here shouldn't inherit whatever expand state was
          // left over from before.
          this.isSelectedMatchExpanded = false;
        }
        await this.loadSelectedRegistrations();
        await this.loadSelectedMotmVotes();
      } catch (error) {
        console.error('Error fetching matches:', error);
        // Don't leave the previous season's list on screen after a failed
        // reload — an empty list is at least not misleading.
        this.matches = [];
      } finally {
        this.isLoading = false;
      }
      // matchesBar only exists in the DOM once matches.length > 0 (see the
      // v-else-if in the template) — (re)attaching here, after the list
      // actually renders, is what makes the scroll buttons work at all on
      // first load, and keeps canScrollLeft/canScrollRight correct after a
      // season change reloads a shorter or longer list.
      this.attachScrollListener();
    },

    // Create Match Methods
    async creatingMatch() {
      if (!this.selectedDate) {
        this.dateError = 'Please select a date';
        return;
      }

      // Mirrors the backend's own rules so the reason shows up next to the
      // field instead of coming back as an opaque 400. The backend stays the
      // authority — see the catch below, which surfaces its message verbatim.
      let scheduling = null;
      if (this.isScheduled) {
        scheduling = this.buildScheduling();
        if (!scheduling) {
          return;
        }
      }
      this.scheduleError = '';

      this.isCreating = true;
      this.dateError = '';

      this.match.Date = this.selectedDate;
      try {
        // Call your createMatch API function. scheduling is null for an
        // ordinary match, which makes the request body identical to what it
        // was before scheduling existed.
        const response = await createMatch(this.match, this.activeGroupId, scheduling);

        if (response) {
          // Close modal
          this.closeModal();

          // Route to edit page with the new match ID
          this.$router.push(`/matches/${response}/edit`);
        } else {
          throw new Error('No match ID returned from server');
        }
      } catch (error) {
        console.error('Error creating match:', error);
        // A rejected schedule (the backend re-checks everything validated
        // above, plus whatever this client doesn't know about) says why —
        // don't flatten it into the generic message.
        const backendMessage = error.response && error.response.data && error.response.data.error;
        this.dateError = backendMessage || 'Failed to create match. Please try again.';
      } finally {
        this.isCreating = false;
      }
    },

    // Returns the { scheduledAt, registrationOpensAt, maxPlayers } payload
    // createMatch expects, or null after setting scheduleError.
    //
    // Both timestamps carry the browser's local UTC offset (see
    // services/datetime.js): the backend derives the match's date — and hence
    // its season — from scheduled_at's calendar day in the offset it was sent
    // in, so a UTC 'Z' string would move a late kick-off to the day before.
    buildScheduling() {
      if (!this.kickoffTime) {
        this.scheduleError = 'Please set a kick-off time';
        return null;
      }
      if (!this.registrationOpensAt) {
        this.scheduleError = 'Please set when sign-ups open';
        return null;
      }

      const maxPlayers = Number(this.maxPlayers);
      if (!Number.isInteger(maxPlayers) || maxPlayers < 1) {
        this.scheduleError = 'Maximum players must be a whole number, at least 1';
        return null;
      }

      const scheduledAt = toLocalRFC3339(this.selectedDate, this.kickoffTime);
      const registrationOpensAt = dateTimeLocalToRFC3339(this.registrationOpensAt);
      if (!scheduledAt || !registrationOpensAt) {
        this.scheduleError = 'Please check the kick-off and sign-up times';
        return null;
      }

      if (Date.parse(registrationOpensAt) >= Date.parse(scheduledAt)) {
        this.scheduleError = 'Sign-ups must open before kick-off';
        return null;
      }

      this.scheduleError = '';
      return { scheduledAt, registrationOpensAt, maxPlayers };
    },

    closeModal() {
      this.showCreateModal = false;
      this.selectedDate = '';
      this.dateError = '';
      this.isCreating = false;
      this.match = {};
      this.isScheduled = false;
      this.kickoffTime = '';
      this.registrationOpensAt = '';
      this.maxPlayers = DEFAULT_MAX_PLAYERS;
      this.scheduleError = '';
    },

    formatSelectedDate(dateStr) {
      return formatCalendarDay(dateStr);
    },

    getCurrentMonthYear() {
      const date = new Date(this.currentYear, this.currentMonth);
      return date.toLocaleDateString('en-US', {
        month: 'long',
        year: 'numeric'
      });
    },

    previousMonth() {
      if (this.currentMonth === 0) {
        this.currentMonth = 11;
        this.currentYear--;
      } else {
        this.currentMonth--;
      }
    },

    nextMonth() {
      if (this.currentMonth === 11) {
        this.currentMonth = 0;
        this.currentYear++;
      } else {
        this.currentMonth++;
      }
    },

    selectDate(date) {
      if (this.isPastDate(date)) return;

      // Fix: Create date in local timezone, not UTC
      const year = date.year;
      const month = date.month;
      const day = date.day;

      // Format as YYYY-MM-DD string directly
      const monthStr = (month + 1).toString().padStart(2, '0');
      const dayStr = day.toString().padStart(2, '0');
      this.selectedDate = `${year}-${monthStr}-${dayStr}`;

      this.dateError = '';
    },

    isSelectedDate(date) {
      if (!this.selectedDate) return false;
      // Parse the YYYY-MM-DD string
      const [year, month, day] = this.selectedDate.split('-').map(Number);

      return year === date.year &&
        (month - 1) === date.month &&  // month in selectedDate is 1-based, date.month is 0-based
        day === date.day;
    },

    isToday(date) {
      const today = new Date();
      return today.getFullYear() === date.year &&
        today.getMonth() === date.month &&
        today.getDate() === date.day;
    },

    isPastDate(date) {
      const today = new Date();
      const dateToCheck = new Date(date.year, date.month, date.day);
      const todayStart = new Date(today.getFullYear(), today.getMonth(), today.getDate());
      return dateToCheck < todayStart;
    },

    getTodayDate() {
      const today = new Date();
      return today.toISOString().split('T')[0];
    },
    selectMatch(match) {
      this.selectedMatch = match;
      this.signupMessage = '';
      this.motmMessage = '';
      // Reset to the collapsed default for whichever match this now is —
      // expand state is not remembered per match, see
      // canCollapseSelectedMatchSignup/showSignupInline.
      this.isSelectedMatchExpanded = false;
      this.loadSelectedRegistrations();
      this.loadSelectedMotmVotes();
    },

    // Loads the sign-up list of `selectedMatch` alone, the same one-request-
    // per-open-match cost MatchDetails.vue already pays — the list endpoint
    // deliberately carries no "am I registered" flag (see CLAUDE.md), so this
    // is the only way to answer it. A no-op for an unscheduled match, or when
    // nothing is selected (an empty list).
    async loadSelectedRegistrations() {
      if (!this.selectedMatch || !isScheduledMatch(this.selectedMatch)) {
        this.registrations = [];
        return;
      }
      this.isLoadingRegistrations = true;
      try {
        const entries = await getMatchRegistrations(this.selectedMatch.ID);
        this.registrations = Array.isArray(entries) ? entries : [];
      } catch (error) {
        console.error('Error loading registrations:', error);
        this.registrations = [];
      } finally {
        this.isLoadingRegistrations = false;
      }
    },

    // Signs the caller up for `selectedMatch`. The count on both this panel
    // and the match's own card (signupCountLabel reads the same
    // RegistrationCount) is bumped locally rather than waiting on a full
    // loadMatches() reload — safe to do without re-deriving anything
    // server-side, since a successful call always adds exactly one
    // registration, confirmed or waiting. Position/waiting status themselves
    // are never guessed client-side, which is why loadSelectedRegistrations()
    // still runs afterward — see MatchDetails.vue's own participate() for the
    // same split.
    async participate() {
      if (this.isUpdatingRegistration) return;
      this.isUpdatingRegistration = true;
      try {
        const entry = await registerForMatch(this.selectedMatch.ID);
        this.selectedMatch.RegistrationCount = (this.selectedMatch.RegistrationCount || 0) + 1;
        await this.loadSelectedRegistrations();
        if (entry && entry.IsWaiting) {
          this.showSignupMessage(`Signed up — you are #${entry.Position}, on the waiting list.`, 'success');
        } else {
          const position = entry && entry.Position ? ` You are #${entry.Position}.` : '';
          this.showSignupMessage(`You're in for this match.${position}`, 'success');
        }
      } catch (error) {
        console.error('Error signing up for match:', error);
        // A 409 says precisely why (not open yet, closed, already
        // registered) — worth showing verbatim, same pattern as
        // MatchDetails.vue. Reload either way: a rejection usually means this
        // panel's view of the list is stale.
        this.showSignupMessage(this.registrationErrorMessage(error, 'Error signing up for this match.'), 'error');
        await this.loadSelectedRegistrations();
      } finally {
        this.isUpdatingRegistration = false;
      }
    },

    // Same confirm-before-acting pattern as MatchDetails.vue's own
    // confirmWithdraw — withdrawing hands your place to the first player on
    // the waiting list, which is worth a heads-up before it happens.
    confirmWithdraw() {
      if (this.isUpdatingRegistration) return;
      const confirmed = window.confirm('Withdraw from this match? Your place goes to the first player on the waiting list.');
      if (!confirmed) return;
      this.withdraw();
    },

    async withdraw() {
      this.isUpdatingRegistration = true;
      try {
        await unregisterFromMatch(this.selectedMatch.ID);
        this.selectedMatch.RegistrationCount = Math.max(0, (this.selectedMatch.RegistrationCount || 1) - 1);
        await this.loadSelectedRegistrations();
        this.showSignupMessage('You have withdrawn from this match.', 'success');
      } catch (error) {
        console.error('Error withdrawing from match:', error);
        this.showSignupMessage(this.registrationErrorMessage(error, 'Error withdrawing from this match.'), 'error');
        await this.loadSelectedRegistrations();
      } finally {
        this.isUpdatingRegistration = false;
      }
    },

    registrationErrorMessage(error, fallback) {
      return error?.response?.data?.error || fallback;
    },

    showSignupMessage(text, type) {
      this.signupMessage = text;
      this.signupMessageType = type;
    },

    // --- Man of the Match voting, for `selectedMatch` alone -----------------
    // Loaded whenever the selection changes, exactly like
    // loadSelectedRegistrations() — a no-op for a match with no composed
    // roster, since there is nobody to vote for yet.
    async loadSelectedMotmVotes() {
      if (!this.selectedMatch || !teamsAreComposed(this.selectedMatch)) {
        this.motmVotes = { Tally: [], MyVoteFor: null };
        return;
      }
      this.isLoadingMotmVotes = true;
      try {
        const summary = await getMatchVotes(this.selectedMatch.ID);
        this.motmVotes = summary && Array.isArray(summary.Tally)
          ? summary
          : { Tally: [], MyVoteFor: null };
      } catch (error) {
        console.error('Error loading Man of the Match votes:', error);
        this.motmVotes = { Tally: [], MyVoteFor: null };
      } finally {
        this.isLoadingMotmVotes = false;
      }
    },

    // How many votes playerId currently holds in the loaded tally, or 0 if
    // none (a candidate with zero votes is simply absent from Tally).
    motmVoteCount(playerId) {
      const entry = this.motmVotes.Tally.find(candidate => candidate.PlayerID === playerId);
      return entry ? entry.Votes : 0;
    },

    isMyMotmVote(playerId) {
      return this.motmVotes.MyVoteFor === playerId;
    },

    // Mirrors the backend's ComputeMotmWinners: whoever has the *most*
    // votes in the tally, tie-inclusive — several players can all be
    // "leading" a match at once, same as the backend awards several MOTMs
    // for a genuine tie. Independent of the caller's own vote entirely.
    isCurrentMotmLeader(playerId) {
      const tally = this.motmVotes.Tally;
      if (!tally.length) return false;
      const maxVotes = Math.max(...tally.map(candidate => candidate.Votes));
      if (maxVotes <= 0) return false;
      const entry = tally.find(candidate => candidate.PlayerID === playerId);
      return !!entry && entry.Votes === maxVotes;
    },

    // The star's tooltip/aria-label: the 24h window takes priority over
    // vote/change-vote wording, since it explains why the star is disabled
    // rather than what clicking it used to do.
    motmStarTitle(player) {
      if (player.ID === this.currentPlayerId) {
        return 'You can\'t vote for yourself as Man of the Match';
      }
      if (!this.motmVotingOpen) {
        return 'Vote fermé 24h après le match';
      }
      return this.isMyMotmVote(player.ID)
        ? `Remove your Man of the Match vote for ${player.Name}`
        : `Vote for ${player.Name} as Man of the Match`;
    },

    // Clicking the star of the player the caller already voted for removes
    // the vote (a toggle); clicking any other candidate's star casts or
    // changes it — MatchVoteService.Vote is an upsert on the backend, so
    // there is no separate "already voted" conflict to handle here, unlike
    // participate() above.
    async toggleMotmVote(playerId) {
      if (this.isUpdatingMotmVote || !this.motmVotingOpen) return;
      this.isUpdatingMotmVote = true;
      const removing = this.isMyMotmVote(playerId);
      try {
        if (removing) {
          await removeMotmVote(this.selectedMatch.ID);
        } else {
          await voteForMotm(this.selectedMatch.ID, playerId);
        }
        await this.loadSelectedMotmVotes();
        this.motmMessage = removing
          ? 'Your Man of the Match vote has been removed.'
          : 'Your Man of the Match vote has been saved.';
        this.motmMessageType = 'success';
      } catch (error) {
        console.error('Error updating Man of the Match vote:', error);
        this.motmMessage = this.registrationErrorMessage(error, 'Error updating your Man of the Match vote.');
        this.motmMessageType = 'error';
        await this.loadSelectedMotmVotes();
      } finally {
        this.isUpdatingMotmVote = false;
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

    // Native "scroll" listeners don't stack for the same function reference
    // on the same element, so calling this more than once (mounted(), then
    // every loadMatches()) is safe — it just re-runs updateScrollButtons()
    // against whatever matchesBar looks like now, which is exactly what's
    // needed after the list's length changes.
    attachScrollListener() {
      this.$nextTick(() => {
        this.updateScrollButtons();
        if (this.$refs.matchesBar) {
          this.$refs.matchesBar.addEventListener('scroll', this.updateScrollButtons);
        }
      });
    },

    formatDate(dateString) {
      return formatCalendarDay(dateString);
    },

    // Exposed as a method rather than imported into the template directly:
    // this component has no setup()/computed path for a bare helper, and every
    // other formatter here is a method too.
    isScheduledMatch(match) {
      return isScheduledMatch(match);
    },

    // Gates the card's own team/score row and the "Selected Match Details"
    // preview's team columns — see teamsAreComposed's own comment. Unscheduled
    // matches have always had a populated roster, so this stays true for them;
    // a scheduled match hides both until an admin has actually assigned
    // someone to a team.
    showTeamRoster(match) {
      return !isScheduledMatch(match) || teamsAreComposed(match);
    },

    // Per-card equivalent of the `registrationState`/`registrationStateLabel`
    // computed pair above, which are scoped to `selectedMatch` alone and
    // can't answer "what state is *this* card's match in" for the rest of the
    // carousel. `deriveRegistrationState` only ever needs a match's own
    // scheduling fields — already on every entry `matches` holds — so this
    // costs no extra request, unlike `isRegistered` which needs the sign-up
    // list itself and stays scoped to whichever match is selected.
    cardRegistrationState(match) {
      return deriveRegistrationState(match, this.nowMs);
    },

    cardRegistrationLabel(match) {
      return registrationStateLabel(this.cardRegistrationState(match));
    },

    formatKickoff(match) {
      return formatDateTimeShort(match.ScheduledAt);
    },

    // RegistrationCount is present-but-zero on a scheduled match nobody has
    // signed up for yet (it is a *int with omitempty on the Go side precisely
    // so 0 and "not scheduled" stay distinguishable), so `?? 0` rather than
    // `|| 0` — not that it matters for 0, but it keeps the intent explicit
    // that only a genuinely absent count falls back.
    signupCountLabel(match) {
      const count = match.RegistrationCount ?? 0;
      return `${count} / ${match.MaxPlayers} signed up`;
    },

    formatDateShort(dateString) {
      return formatCalendarDayShort(dateString);
    },

    getTeamColor(colour) {
      if (colour && colour.startsWith('#')) {
        return colour;
      }

      const colorMap = {
        'red': '#ef4444',
        'blue': '#3b82f6',
        'green': '#10b981',
        'yellow': '#f59e0b',
        'purple': '#8b5cf6',
        'orange': '#f97316',
        'pink': '#ec4899',
        'cyan': '#06b6d4',
        'white': '#f8fafc',
        'black': '#1f2937'
      };

      return colorMap[colour.toLowerCase()] || '#6b7280';
    },

    formatPlayerNameForDisplay(name) {
      // Convert to title case for display (capitalize first letter of each word)
      return name.split(' ')
        .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
        .join(' ');
    },

    getTotalPlayers(match) {
      return match.Teams.reduce((total, team) => total + team.Players.length, 0);
    }
  }
};
</script>

<style scoped>
/* Component-specific styles that couldn't be moved to global */

/* Container */
.matches-panel {
  background-color: var(--bg-secondary);
}

/* Modal Header Gradient */
.modal-header-gradient {
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--secondary-color) 100%) !important;
  color: white !important;
  border-bottom: none !important;
}

.modal-header-gradient h3 {
  color: white !important;
}

.modal-header-gradient .modal-close {
  color: rgba(255, 255, 255, 0.8) !important;
}

.modal-header-gradient .modal-close:hover {
  background-color: rgba(255, 255, 255, 0.1) !important;
  color: white !important;
}

/* Modal Body Large */
.modal-body-large {
  max-height: 70vh;
  overflow-y: auto;
}

/* Date Picker Styles */
.date-picker-container {
  background-color: var(--bg-primary);
  border: 2px solid var(--border-color);
  border-radius: var(--border-radius);
  overflow: hidden;
  margin-bottom: 0;
}

.selected-date-display {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 1rem;
  padding: 0.75rem 1rem;
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--secondary-color) 100%);
  color: white;
  border-radius: var(--border-radius);
  font-weight: 500;
}

.selected-date-display svg {
  width: 16px;
  height: 16px;
}

/* Optional Scheduling */
.schedule-group {
  margin-bottom: 0;
}

.schedule-toggle {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  margin-bottom: 0;
  cursor: pointer;
  font-weight: 500;
  color: var(--text-primary);
}

.schedule-checkbox {
  width: 18px;
  height: 18px;
  accent-color: var(--primary-color);
  cursor: pointer;
  margin: 0;
}

.schedule-fields {
  margin-top: 1rem;
  padding: 1rem;
  background-color: var(--bg-primary);
  border: 2px solid var(--border-color);
  border-radius: var(--border-radius);
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.schedule-field label {
  margin-bottom: 0.35rem;
  font-size: 0.9rem;
}

.schedule-hint {
  margin: 0.35rem 0 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.schedule-error {
  margin-top: 0.75rem;
}

.date-picker-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem;
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--secondary-color) 100%);
  color: white;
}

.month-year {
  font-size: 1.1rem;
  font-weight: 600;
  margin: 0;
}

.nav-btn {
  background: transparent;
  border: none;
  color: white;
  cursor: pointer;
  padding: 0.5rem;
  border-radius: var(--border-radius);
  transition: background-color var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
}

.nav-btn:hover {
  background-color: rgba(255, 255, 255, 0.2);
}

.nav-btn svg {
  width: 18px;
  height: 18px;
}

.date-picker-grid {
  padding: 1rem;
  padding-bottom: 0.5rem;
}

.day-headers {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 0.25rem;
  margin-bottom: 0.5rem;
}

.day-header {
  text-align: center;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-secondary);
  padding: 0.5rem;
}

.days-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 0.25rem;
}

.day-button {
  aspect-ratio: 1;
  border: none;
  background: transparent;
  color: var(--text-primary);
  font-size: 0.875rem;
  font-weight: 500;
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 40px;
  position: relative;
}

.day-button:hover:not(.disabled) {
  background-color: var(--bg-tertiary);
  transform: scale(1.05);
}

.day-button.other-month {
  color: var(--text-light);
  opacity: 0.5;
}

.day-button.selected {
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--secondary-color) 100%);
  color: white;
  font-weight: 700;
  box-shadow: var(--shadow-sm);
}

.day-button.today:not(.selected) {
  background-color: var(--bg-tertiary);
  color: var(--primary-color);
  font-weight: 700;
  position: relative;
}

.day-button.today:not(.selected)::after {
  content: '';
  position: absolute;
  bottom: 4px;
  left: 50%;
  transform: translateX(-50%);
  width: 4px;
  height: 4px;
  background-color: var(--primary-color);
  border-radius: 50%;
}

.day-button.disabled {
  color: var(--text-light);
  opacity: 0.4;
  pointer-events: none;
  cursor: not-allowed;
}

.day-button.disabled:hover {
  transform: none;
  background: transparent;
}

/* Matches Section */
.matches-section {
  padding: 2rem 0;
}

/* Matches Layout */
.matches-layout {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

/* Horizontal Matches Bar */
/* Overrides card-base's own 1.5rem padding — .matches-bar below already
   adds its own, so the default would stack two layers of padding around
   the match list for no reason. */
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

/* Leading "+" card — replaces the old standalone "Create Match" toolbar,
   living at the start of the match list instead. Sticky rather than a
   plain flex item: with enough matches to need scrolling, a purely leading
   card would scroll out of view along with the rest of the list, so it
   stays pinned to the left edge of the scrollable area instead — the same
   "sticky first column" technique used for scrollable tables. */
.add-match-card {
  flex: 0 0 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px dashed var(--border-color);
  color: var(--primary-color);
  position: sticky;
  left: 0;
  z-index: 2;
  background-color: var(--bg-tertiary);
  box-shadow: 8px 0 8px -8px rgba(0, 0, 0, 0.15);
}

.add-match-card:hover {
  border-color: var(--primary-color);
  background-color: var(--bg-primary);
}

.add-match-card svg {
  width: 24px;
  height: 24px;
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

/* A scheduled match reads differently at a glance: a coloured left edge, so
   it is distinguishable from a played match even before the badge below is
   read (and for anyone who can't tell the badge's colour apart). The border
   is only the left one, so .active's full 2px accent border still wins
   visually on the selected card. */
.match-card-horizontal.scheduled {
  border-left-color: var(--primary-color);
}

/* The kick-off string is longer than a bare date, so the date row is allowed
   to wrap on a scheduled card rather than overflowing the 220px card. */
.match-card-horizontal.scheduled .match-date-horizontal {
  flex-wrap: wrap;
  margin-bottom: 0.4rem;
}

.match-signups-horizontal {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  flex-wrap: wrap;
  margin-bottom: 0.6rem;
}

.signup-count {
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-weight: 600;
}

.match-date-horizontal svg {
  width: 14px;
  height: 14px;
}

/* No margin-bottom: with the status dot removed below, this is now the
   card's last child — the card's own padding already provides the
   trailing space, an extra margin here would just add dead room under
   it. */
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

.team-color-horizontal {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 1px solid white;
  box-shadow: var(--shadow-sm);
}

.team-name-horizontal {
  font-weight: 600;
  color: var(--text-primary);
  text-transform: capitalize;
  font-size: 0.875rem;
}

.team-score-horizontal {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--primary-color);
  background-color: var(--bg-primary);
  padding: 0.125rem 0.5rem;
  border-radius: 6px;
  min-width: 30px;
  text-align: center;
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

/* Scroll Buttons */
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

/* Match Details Container */
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

.details-header-actions {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.edit-match-btn {
  text-decoration: none !important;
  color: white !important;
}

.edit-match-btn:hover {
  text-decoration: none !important;
  color: white !important;
}

/* Collapse/expand toggle for the sign-up chrome below, once the roster is
   composed (see canCollapseSelectedMatchSignup) — a plain text-and-chevron
   button, following this codebase's own convention for a small secondary
   action (compare .nav-btn's month-navigation chevrons in this same file). */
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

/* Points up while expanded (matching "this is what collapses it"), down
   while collapsed (matching "this is what expands it"). */
.signup-toggle-chevron.is-expanded {
  transform: rotate(180deg);
}

/* Sign-ups inline in the Matches tab preview — same visual language as
   MatchDetails.vue's own .signup-panel (state badge colours included), just
   condensed to one row plus a detail line, since this panel has no room for
   (and no need for) the full roster. */
.signup-inline {
  margin-bottom: 1.5rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--border-color);
}

.signup-inline-top {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.6rem;
}

.signup-count-inline {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

/* Same pill shape and colours as MatchDetails.vue's .signup-state-badge —
   duplicated rather than shared, the same way getTeamColor() is duplicated
   between the two files. */
.signup-state-badge {
  padding: 0.3rem 0.65rem;
  border-radius: 20px;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  background-color: var(--bg-tertiary);
  color: var(--text-secondary);
}

.signup-state-badge.open {
  background-color: #d1fae5;
  color: #065f46;
}

.signup-state-badge.not-open-yet {
  background-color: #fef3c7;
  color: #92400e;
}

.signup-state-badge.closed-by-admin,
.signup-state-badge.closed-at-kickoff {
  background-color: #e5e7eb;
  color: #374151;
}

.signup-inline-detail {
  margin: 0.5rem 0 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.signup-inline-actions {
  display: flex;
  gap: 0.6rem;
}

.signup-inline-message {
  margin: 0.6rem 0 0;
  font-size: 0.85rem;
}

.signup-inline-message.success {
  color: #065f46;
}

.signup-inline-message.error {
  color: var(--danger-color);
}

/* Confirmed/waiting sign-up lists, with names — duplicated verbatim from
   MatchDetails.vue's own .signup-panel (state badge colours already
   duplicated above), the same way getTeamColor() is duplicated between the
   two files. This panel became the only place a plain member can see this
   list once MatchDetails.vue turned admin-only (see router/index.js). */
.signup-loading {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.75rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.signup-lists {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
  margin-top: 1rem;
}

.signup-list-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0 0 0.75rem;
  font-size: 0.9rem;
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
  max-height: 14rem;
  overflow-y: auto;
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

.signup-position {
  min-width: 1.5rem;
  font-weight: 700;
  color: var(--text-secondary);
  font-variant-numeric: tabular-nums;
}

.signup-name {
  flex: 1;
  color: var(--text-primary);
  font-weight: 500;
}

.signup-you {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--primary-color);
}

.signup-list-waiting .signup-entry {
  opacity: 0.8;
}

.signup-empty {
  padding: 0.5rem 0.65rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
  font-style: italic;
}

@media (max-width: 768px) {
  .signup-lists {
    grid-template-columns: 1fr;
  }
}

/* Players by team — one column per team, each an independent list of its
   own scorers. Side by side on desktop; stacked on mobile (see the 768px
   media query) so seeing the second team never requires scrolling. */
.players-section {
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.no-roster-hint {
  margin: 0;
  font-size: 0.875rem;
  color: var(--text-secondary);
  font-style: italic;
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

/* Bounded to whatever's actually left below the match carousel and the
   "Edit Match" button above it, with its own scrollbar — a team with
   many players then scrolls inside its own column instead of growing the
   whole page and pushing everything above (season selector, match
   carousel, Edit Match) out of easy reach. The 620px offset is an
   estimate of that chrome's height (nav + context bar + tabs + carousel
   card + details header, all above this point); max() keeps a handful of
   rows visible even if that chrome runs taller than estimated on a given
   screen. The header row above the list stays put regardless, since it's
   a sibling outside this scroll region, not inside it. */
.team-players-list {
  list-style: none;
  margin: 0;
  padding: 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  max-height: max(9rem, calc(100vh - 660px));
  overflow-y: auto;
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
  /* Grows to fill the row, which is what pushes .player-motm and
     .goal-badge together at the end with just the row's own gap between
     them, instead of the old justify-content: space-between spreading all
     three children evenly (and drifting the star around depending on
     whether the vote-count pill was showing). */
  flex: 1 1 auto;
  min-width: 0;
}

.team-player-row .player-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.goal-badge {
  flex-shrink: 0;
  background-color: var(--bg-tertiary);
  color: var(--primary-color);
  font-weight: 700;
  padding: 0.15rem 0.6rem;
  border-radius: 999px;
  min-width: 1.75rem;
  text-align: center;
}

.team-players-list .empty-slot {
  padding: 0.5rem;
  text-align: center;
  color: var(--text-light);
  font-style: italic;
}

/* Man of the Match voting, per roster row — moved here from MatchDetails.vue
   entirely (see CLAUDE.md). The vote-count pill follows this codebase's own
   "muted pill" convention (.registration-badge.waiting in MatchDetails.vue,
   .left-group-tag in PointsStandingsTable.vue/ScorersTable.vue) rather than
   inventing a new style language. */
.player-motm {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  /* Fixed width, right-aligned content: the star's x-position stays
     constant across every row in the column whether or not the
     vote-count pill is present, which is what keeps the stars lined up
     vertically instead of drifting per-row. */
  justify-content: flex-end;
  gap: 0.4rem;
  min-width: 5.75rem;
}

.motm-vote-count {
  display: inline-flex;
  align-items: center;
  background-color: var(--bg-tertiary);
  color: var(--text-secondary);
  font-size: 0.7rem;
  font-weight: 600;
  padding: 0.15rem 0.5rem;
  border-radius: 999px;
  white-space: nowrap;
}

/* Whoever currently has the *most* votes for this match (tie-inclusive) —
   independent of the caller's own vote, unlike .motm-star-btn.is-voted
   below. Gold rather than the pill's usual muted grey, so the match's
   actual (derived) MOTM stands out even to someone who never voted at
   all. */
.motm-vote-count.is-leader {
  background-color: rgba(245, 158, 11, 0.16);
  color: #b45309;
  font-weight: 700;
}

.motm-leader-icon {
  width: 0.75rem;
  height: 0.75rem;
  margin-right: 0.2rem;
  color: #f59e0b;
}

.motm-star-btn {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 1.75rem;
  height: 1.75rem;
  padding: 0;
  border: none;
  background: none;
  color: var(--text-secondary);
  border-radius: 50%;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.motm-star-btn svg {
  width: 18px;
  height: 18px;
}

.motm-star-btn:hover:not(:disabled) {
  background-color: var(--bg-tertiary);
  color: #f59e0b;
}

/* The caller's own current vote — filled star, same accent used for goal
   totals/counts elsewhere on this card. */
.motm-star-btn.is-voted {
  color: #f59e0b;
}

.motm-star-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Still rendered (so .player-motm's column width/alignment never shifts),
   just invisible — see the template comment above this button. */
.motm-star-btn.is-self {
  visibility: hidden;
}

.motm-inline-message {
  margin: 0.75rem 0 0;
  font-size: 0.85rem;
}

.motm-inline-message.success {
  color: #065f46;
}

.motm-inline-message.error {
  color: var(--danger-color);
}

/* Transitions */
.modal-enter-active,
.modal-leave-active {
  transition: all var(--transition-smooth);
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
  transform: scale(0.95);
}

.modal-enter-active .modal-container,
.modal-leave-active .modal-container {
  transition: all var(--transition-smooth);
}

.modal-enter-from .modal-container,
.modal-leave-to .modal-container {
  transform: translateY(20px);
}

.match-details-enter-active,
.match-details-leave-active {
  transition: all var(--transition-smooth);
}

.match-details-enter-from,
.match-details-leave-to {
  opacity: 0;
  transform: translateY(20px);
}

/* Responsive Design */
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

  .add-match-card {
    flex-basis: 64px;
  }

  .match-details-container {
    padding: 1rem;
  }

  /* Create Match modal's calendar — smaller header/grid/cells so the
     whole picker takes less of the screen on a small phone. */
  .modal-body-large {
    padding: 1rem;
  }

  .form-group {
    margin-bottom: 1rem;
  }

  .date-picker-header {
    padding: 0.75rem;
  }

  .month-year {
    font-size: 1rem;
  }

  .date-picker-grid {
    padding: 0.75rem;
    padding-bottom: 0.5rem;
  }

  .day-header {
    padding: 0.35rem;
    font-size: 0.75rem;
  }

  .day-button {
    min-height: 34px;
    font-size: 0.8rem;
  }

  .selected-date-display {
    padding: 0.5rem 0.75rem;
    margin-top: 0.75rem;
    font-size: 0.85rem;
  }

  .details-title-section {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }

  /* "Sunday, August 23, 2026 - Match Details" at the desktop 1.5rem size
     wraps to two or three lines on a narrow screen. */
  .details-header h3 {
    font-size: 1.1rem;
  }

  .edit-match-btn {
    justify-content: center;
  }

  /* Stacked, one team per row, instead of side by side. This used to be
     side-by-side deliberately (stacking traded the horizontal squeeze for
     a lot of vertical scrolling) — reversed once real testing showed the
     squeeze itself had gotten worse than that trade-off: the Man of the
     Match star sits in its own fixed-width column right before the goal
     badge (see .player-motm), which on a ~150px-wide column left barely
     any room for a name at all — a long one (e.g. "hubert bonniseur de la
     batte") rendered as a single truncated letter. A full-width column
     gives the name room to actually be read; the extra scrolling is the
     smaller cost. */
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

  .team-player-row .player-info {
    gap: 0.35rem;
  }

  .team-player-row .player-name {
    font-size: 0.85rem;
  }

  .goal-badge {
    font-size: 0.75rem;
    padding: 0.1rem 0.4rem;
    min-width: 1.4rem;
  }

  /* Touch scrolling already works natively on the horizontal match list —
     these arrows just take up space without doing anything a swipe
     doesn't already do on a touch screen. */
  .scroll-btn {
    display: none;
  }

  .modal-container {
    margin: 1rem;
    max-width: none;
  }

  .modal-body-large {
    padding: 1.5rem;
  }
}

@media (max-width: 480px) {
  .match-card-horizontal {
    flex: 0 0 170px;
    padding: 0.5rem;
  }

  /* .match-card-horizontal's flex shorthand above would otherwise widen
     this to 200px too — it also carries that class — undoing the 64px
     .add-match-card already set at the 768px breakpoint. */
  .add-match-card {
    flex: 0 0 64px;
  }

  .match-details-container {
    padding: 0.75rem;
  }
}
</style>