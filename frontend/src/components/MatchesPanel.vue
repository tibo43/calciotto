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
                :class="{ 'active': selectedMatch?.ID === match.ID }" @click="selectMatch(match)">
                <!-- Match Date -->
                <div class="match-date-horizontal">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
                    <line x1="16" y1="2" x2="16" y2="6" />
                    <line x1="8" y1="2" x2="8" y2="6" />
                    <line x1="3" y1="10" x2="21" y2="10" />
                  </svg>
                  <span>{{ formatDateShort(match.Date) }}</span>
                </div>

                <!-- Teams and Scores -->
                <div class="teams-horizontal">
                  <div v-for="(team, index) in match.Teams" :key="team.ID" class="team-horizontal">
                    <div class="team-info-horizontal">
                      <div class="team-color-horizontal" :style="{ backgroundColor: getTeamColor(team.Colour) }"></div>
                      <span class="team-name-horizontal">{{ team.Name }}</span>
                    </div>
                    <div class="team-score-horizontal">{{ team.Score }}</div>
                    <div v-if="index < match.Teams.length - 1" class="vs-separator">vs</div>
                  </div>
                </div>

                <!-- Match Status -->
                <div class="match-status-horizontal">
                  <div class="status-indicator-horizontal" :class="getMatchStatus(match)"></div>
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
                  <!-- The destination page (MatchDetails.vue) stays viewable by
                       any member — only its own editing controls are
                       admin-gated — so this link is never hidden, just
                       relabelled: showing "Edit Match" with a pencil icon to a
                       non-admin implied an ability that isn't actually there. -->
                  <router-link
                    :to="`/matches/${selectedMatch.ID}/edit`"
                    class="btn-base btn-primary btn-small edit-match-btn"
                  >
                    <svg v-if="isAdmin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                    </svg>
                    <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                      <circle cx="12" cy="12" r="3" />
                    </svg>
                    {{ isAdmin ? 'Edit Match' : 'View Match' }}
                  </router-link>
                </div>
                <div class="details-divider"></div>
              </div>

              <div class="players-section">
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
                        <span class="goal-badge">{{ player.GoalNumber || 0 }}</span>
                      </li>
                      <li v-if="!team.Players.length" class="empty-slot">No players yet</li>
                    </ul>
                  </div>
                </div>
              </div>
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
import { getMatchesDetails, createMatch } from '@/services/api';

// The Matches sub-tab of MatchesAndStandings.vue: the match carousel, the
// selected match's preview, and the admin-only create-match modal. Everything
// it is scoped by — the active group, the caller's admin role, the selected
// season — is resolved once by the page and passed down, so this component
// only ever loads matches.
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
      // Custom Date Picker
      currentMonth: new Date().getMonth(),
      currentYear: new Date().getFullYear(),
      dayHeaders: ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su']
    };
  },
  async created() {
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

        // Auto-select the newest match
        if (this.matches.length > 0) {
          this.selectedMatch = this.matches[0];
        }
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

      this.isCreating = true;
      this.dateError = '';

      this.match.Date = this.selectedDate;
      try {
        // Call your createMatch API function
        const response = await createMatch(this.match, this.activeGroupId);

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
        this.dateError = 'Failed to create match. Please try again.';
      } finally {
        this.isCreating = false;
      }
    },

    closeModal() {
      this.showCreateModal = false;
      this.selectedDate = '';
      this.dateError = '';
      this.isCreating = false;
      this.match = {};
    },

    formatSelectedDate(dateStr) {
      const date = new Date(dateStr);
      return date.toLocaleDateString('en-US', {
        weekday: 'short',
        year: 'numeric',
        month: 'short',
        day: 'numeric'
      });
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
      try {
        const date = new Date(dateString);
        return date.toLocaleDateString('en-US', {
          weekday: 'short',
          year: 'numeric',
          month: 'short',
          day: 'numeric'
        });
      } catch (error) {
        return dateString;
      }
    },

    formatDateShort(dateString) {
      try {
        const date = new Date(dateString);
        return date.toLocaleDateString('en-US', {
          year: 'numeric',
          month: 'short',
          day: 'numeric'
        });
      } catch (error) {
        return dateString;
      }
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

    getTotalGoals(match) {
      return match.Teams.reduce((total, team) => total + (team.Score || 0), 0);
    },

    getTotalPlayers(match) {
      return match.Teams.reduce((total, team) => total + team.Players.length, 0);
    },

    getMatchStatus(match) {
      const totalGoals = this.getTotalGoals(match);
      if (totalGoals === 0) return 'upcoming';
      return 'completed';
    },

    getMatchStatusText(match) {
      const status = this.getMatchStatus(match);
      switch (status) {
        case 'upcoming': return 'Scheduled';
        case 'completed': return 'Completed';
        default: return 'Unknown';
      }
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
  gap: 1.5rem;
}

/* Horizontal Matches Bar */
.matches-bar-container {
  position: relative;
  overflow: hidden;
}

.matches-bar {
  display: flex;
  gap: 1rem;
  padding: 1rem;
  overflow-x: auto;
  scroll-behavior: smooth;
}

.match-card-horizontal {
  flex: 0 0 220px;
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  padding: 0.85rem;
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

.match-date-horizontal svg {
  width: 14px;
  height: 14px;
}

.teams-horizontal {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
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

.match-status-horizontal {
  display: flex;
  justify-content: center;
}

.status-indicator-horizontal {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  animation: pulse-status 2s infinite;
}

.status-indicator-horizontal.upcoming {
  background-color: #f59e0b;
}

.status-indicator-horizontal.completed {
  background-color: var(--primary-color);
}

@keyframes pulse-status {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
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

.edit-match-btn {
  text-decoration: none !important;
  color: white !important;
}

.edit-match-btn:hover {
  text-decoration: none !important;
  color: white !important;
}

/* Players by team — one column per team, each an independent list of its
   own scorers. Side by side on desktop; stacked on mobile (see the 768px
   media query) so seeing the second team never requires scrolling. */
.players-section {
  overflow: hidden;
  display: flex;
  flex-direction: column;
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
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.4rem 0.5rem;
  border-radius: var(--border-radius);
  transition: background-color var(--transition-fast);
}

.team-player-row:hover {
  background-color: var(--bg-secondary);
}

.team-player-row .player-info {
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
    gap: 1rem;
  }

  .matches-bar {
    padding: 0.75rem;
    gap: 0.75rem;
  }

  .match-card-horizontal {
    flex: 0 0 200px;
    padding: 0.75rem;
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

  /* Keep both teams side by side even on mobile — stacking them (an
     earlier attempt) fixed the horizontal scroll but traded it for a lot
     of vertical scrolling instead. Everything inside a column is shrunk
     to fit two ~150px-wide columns without overflowing. */
  .teams-columns {
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