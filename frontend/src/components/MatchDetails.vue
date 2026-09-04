<template>
  <div class="match-detail-container">
    <!-- Loading State -->
    <div v-if="isLoading" class="loading-container">
      <div class="loading-spinner"></div>
      <p class="loading-text">Loading match details...</p>
    </div>

    <!-- Match Content -->
    <div v-else-if="match && match.Teams && match.Teams.length > 0" class="match-content">
      <div class="container">
        <!-- Match Score Overview -->
        <div class="score-overview card-base">
          <div class="match-title">
            <div class="match-title-left">
              <button @click="goBack" class="match-back-btn" aria-label="Back to Matches">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="15,18 9,12 15,6" />
                </svg>
              </button>
            </div>
            <div class="match-title-right">
              <span class="match-date">{{ formatDate(match?.Date) }}</span>
              <div class="match-status-badge" :class="getMatchStatus()">
                {{ getMatchStatusText() }}
              </div>
            </div>
          </div>

          <div v-if="showTeamRoster" class="teams-score">
            <!-- Team 1 -->
            <div class="team-score">
              <div class="team-info">
                <div class="team-color" :style="{ backgroundColor: getTeamColor(match.Teams[0].Colour) }"></div>
                <h3 class="team-name">{{ match.Teams[0].Name }}</h3>
              </div>
              <div class="score">{{ match.Teams[0].Score || 0 }}</div>
            </div>

            <!-- VS Divider -->
            <div class="vs-divider">
              <div class="vs-circle">
                <span>VS</span>
              </div>
            </div>

            <!-- Team 2 -->
            <div class="team-score">
              <div class="team-info">
                <div class="team-color" :style="{ backgroundColor: getTeamColor(match.Teams[1].Colour) }"></div>
                <h3 class="team-name">{{ match.Teams[1].Name }}</h3>
              </div>
              <div class="score">{{ match.Teams[1].Score || 0 }}</div>
            </div>
          </div>
        </div>

        <!-- Sign-up panel — scheduled matches only. An ordinary match carries
             none of these fields, so the whole block is absent for it, exactly
             as it was before this feature existed.

             Note what is NOT admin-gated here: Participate and Withdraw. Every
             other control on this page is (see v-if="isAdmin" throughout), but
             signing yourself up is the one thing an ordinary member comes here
             to do — gating it would ship a feature only admins could use. Only
             Close/Reopen are admin actions. -->
        <div v-if="isScheduled" class="signup-panel card-base">
          <div class="signup-header">
            <div class="signup-kickoff">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10" />
                <polyline points="12,6 12,12 16,14" />
              </svg>
              <div class="signup-kickoff-text">
                <span class="signup-label">Kick-off</span>
                <span class="signup-kickoff-value">{{ kickoffLabel }}</span>
              </div>
            </div>

            <!-- Badge, count and Participate/Withdraw all on one line: the
                 badge alone used to read as a status display with nothing to
                 act on right next to it. Close/Reopen/Fill-teams stay below in
                 .signup-actions — those are admin-only occasional actions, not
                 the one a member comes here to take. -->
            <div class="signup-status-group">
              <div class="signup-state-badge" :class="registrationState">{{ registrationStateLabel }}</div>
              <span class="signup-count-inline">{{ signupCountLabel }}</span>
              <!-- Open to every member, admin or not. -->
              <button v-if="canParticipate" @click="participate" :disabled="isUpdatingRegistration"
                class="btn-base btn-primary btn-small participate-btn">
                <svg v-if="!isUpdatingRegistration" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                  stroke-width="2">
                  <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                  <circle cx="8.5" cy="7" r="4" />
                  <line x1="20" y1="8" x2="20" y2="14" />
                  <line x1="23" y1="11" x2="17" y2="11" />
                </svg>
                <div v-else class="loading-spinner-small"></div>
                {{ isUpdatingRegistration ? 'Signing up...' : 'Participate' }}
              </button>

              <!-- Disappears rather than failing when the list closes:
                   DELETE /matches/:id/registrations is gated on the very same
                   window as the POST, so a visible Withdraw button on a closed
                   list would be a button that always 409s. -->
              <button v-if="canWithdraw" @click="confirmWithdraw" :disabled="isUpdatingRegistration"
                class="btn-base btn-cancel btn-small withdraw-btn">
                <svg v-if="!isUpdatingRegistration" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                  stroke-width="2">
                  <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                  <circle cx="8.5" cy="7" r="4" />
                  <line x1="23" y1="11" x2="17" y2="11" />
                </svg>
                <div v-else class="loading-spinner-small"></div>
                {{ isUpdatingRegistration ? 'Withdrawing...' : 'Withdraw' }}
              </button>
            </div>
          </div>

          <p class="signup-state-detail">{{ registrationStateDetail }}</p>

          <div class="signup-actions">
            <button v-if="canCloseRegistrations" @click="closeRegistrations" :disabled="isUpdatingRegistrationState"
              class="btn-base btn-cancel btn-small">
              <div v-if="isUpdatingRegistrationState" class="loading-spinner-small"></div>
              {{ isUpdatingRegistrationState ? 'Closing...' : 'Close sign-ups' }}
            </button>

            <button v-if="canReopenRegistrations" @click="reopenRegistrations" :disabled="isUpdatingRegistrationState"
              class="btn-base btn-cancel btn-small">
              <div v-if="isUpdatingRegistrationState" class="loading-spinner-small"></div>
              {{ isUpdatingRegistrationState ? 'Reopening...' : 'Reopen sign-ups' }}
            </button>

            <!-- A plain wa.me link, not a click handler: it's a URL WhatsApp
                 itself publishes, opened in a new tab like any other outbound
                 link. With no phone number, WhatsApp prompts the admin to pick
                 who to send it to — any contact or group they're already in —
                 so posting into a WhatsApp group needs nothing from this app
                 beyond building the message text. Only offered while sign-ups
                 are actually open: there's nothing to invite people to once
                 the list is closed. -->
            <a v-if="isAdmin && registrationsOpen" :href="whatsappShareUrl" target="_blank" rel="noopener"
              class="btn-base btn-cancel btn-small whatsapp-share-btn">
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path
                  d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.472-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.148-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z" />
                <path
                  d="M12.004 2c-5.514 0-9.997 4.483-9.997 9.997 0 1.762.464 3.485 1.346 5.002L2 22l5.14-1.334a9.958 9.958 0 0 0 4.862 1.237h.004c5.514 0 9.997-4.483 9.997-9.997 0-2.67-1.04-5.182-2.928-7.07A9.933 9.933 0 0 0 12.004 2zm0 18.183h-.003a8.19 8.19 0 0 1-4.17-1.142l-.299-.178-3.05.793.814-2.973-.195-.306a8.18 8.18 0 0 1-1.256-4.38c0-4.523 3.68-8.203 8.203-8.203 2.19 0 4.25.853 5.799 2.404a8.146 8.146 0 0 1 2.403 5.803c0 4.523-3.681 8.202-8.246 8.202z" />
              </svg>
              Share on WhatsApp
            </a>

            <!-- Admin-only, and only once the list is closed: the product flow
                 is "close sign-ups in order to compose the teams", and offering
                 this mid-registration would invite an admin to build teams from
                 a roster still changing under them. -->
            <button v-if="canFillTeamsFromSignups" @click="openComposeChoice"
              class="btn-base btn-cancel btn-small fill-teams-btn">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
                <polyline points="16,11 18,13 22,9" />
              </svg>
              Fill teams from sign-ups
            </button>
          </div>

          <div v-if="isLoadingRegistrations" class="signup-loading">
            <div class="loading-spinner-small"></div>
            <span>Loading sign-ups...</span>
          </div>

          <!-- The confirmed/waiting split comes straight from each entry's
               server-side IsWaiting, never from comparing Position against
               MaxPlayers here: the two would disagree the moment an admin
               changes the cap. -->
          <div v-else class="signup-lists">
            <div class="signup-list">
              <h4 class="signup-list-title">
                Confirmed
                <span class="count-badge">{{ confirmedRegistrations.length }} / {{ match.MaxPlayers }}</span>
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

        <!-- Team Management Section -->
        <div v-if="match.Teams && match.Teams.length > 0" class="team-management card-base card-large">
          <div class="management-header">
            <!-- Global actions — they apply to the whole match, not to
                 whichever team tab happens to be selected, so they come
                 before the team switcher rather than being grouped with it. -->
            <div class="action-buttons">
              <button v-if="isAdmin" @click="saveChanges" class="btn-base btn-primary btn-small" :disabled="isSaving || !showTeamRoster"
                :title="showTeamRoster ? '' : 'Nothing to save yet — compose the teams first'">
                <svg v-if="!isSaving" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
                  <polyline points="17,21 17,13 7,13 7,21" />
                  <polyline points="7,3 7,8 15,8" />
                </svg>
                <div v-else class="loading-spinner-small"></div>
                {{ isSaving ? 'Saving...' : 'Save Changes' }}
              </button>
              <!-- Clearly separated from Save Changes so a click can't be
                   mistaken between the two destructive/non-destructive actions. -->
              <button v-if="isAdmin" @click="confirmDeleteMatch" class="btn-base btn-danger btn-small delete-match-btn"
                :disabled="isDeleting">
                <svg v-if="!isDeleting" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3,6 5,6 21,6" />
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                </svg>
                <div v-else class="loading-spinner-small"></div>
                {{ isDeleting ? 'Deleting...' : 'Delete Match' }}
              </button>
            </div>

            <!-- Team switcher + "Add player to this team" right next to it —
                 Add Player acts on whichever tab is active, so it lives here
                 rather than among the global actions above. -->
            <div v-if="showTeamRoster" class="tabs-buttons">
              <button v-for="(team, index) in match.Teams" :key="team.ID" @click="activeTeam = index"
                :class="['tab-button', { active: activeTeam === index }]">
                <div class="team-color-small" :style="{ backgroundColor: getTeamColor(team.Colour) }"></div>
                {{ team.Name }} ({{ team.Players ? team.Players.length : 0 }})
              </button>
              <button v-if="isAdmin" @click="showModal" class="add-player-icon-btn" aria-label="Add player">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19" />
                  <line x1="5" y1="12" x2="19" y2="12" />
                </svg>
              </button>
            </div>
            <!-- The roster itself only shows up once there's one to show —
                 see showTeamRoster. Composing it is "Fill teams from
                 sign-ups", in the sign-up panel above, once sign-ups close. -->
            <p v-else class="no-roster-hint">
              Teams will appear here once sign-ups close and are composed.
            </p>
          </div>

          <!-- Players List — one compact row per player (avatar, name, goal
               counter, remove) instead of a header row plus a separate goal
               row, so the list needs far less scrolling. -->
          <div v-if="showTeamRoster && match.Teams[activeTeam] && match.Teams[activeTeam].Players" class="players-grid">
            <div v-for="(player, playerIndex) in match.Teams[activeTeam].Players" :key="playerIndex"
              class="player-card">
              <div class="player-info">
                <h4 class="player-name">{{ formatPlayerNameForDisplay(player.Name) }}</h4>
              </div>

              <div class="goal-management">
                <button v-if="isAdmin" @click="updateGoals(playerIndex, -1)"
                  :disabled="!player.GoalNumber || player.GoalNumber <= 0" class="goal-btn decrease">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="5" y1="12" x2="19" y2="12" />
                  </svg>
                </button>

                <span class="goal-count">{{ player.GoalNumber || 0 }}</span>

                <button v-if="isAdmin" @click="updateGoals(playerIndex, 1)" class="goal-btn increase">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="12" y1="5" x2="12" y2="19" />
                    <line x1="5" y1="12" x2="19" y2="12" />
                  </svg>
                </button>
              </div>

              <button v-if="isAdmin" @click="removePlayer(playerIndex)" class="btn-danger-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
            </div>
          </div>
        </div>

        <!-- Man of the Match voting — any match with a composed roster, not
             just a scheduled one, and not admin-gated: unlike the sign-up
             panel's Close/Reopen, there is no admin-only action here at all.
             Voter eligibility is deliberately broader than "played in this
             match" (a sub or a watching member can judge too), which is why
             this isn't behind isAdmin — but the dropdown itself only offers
             roster players other than the caller, since the backend rejects
             a self-vote outright and there is nothing to gain from letting a
             member discover that from a 400. -->
        <div v-if="isRosterComposed" class="motm-panel card-base">
          <div class="motm-header">
            <h3 class="motm-title">Man of the Match</h3>
            <span v-if="hasVotedForMotm" class="motm-my-vote">
              You voted for {{ formatPlayerNameForDisplay(myMotmVoteName) }}
            </span>
          </div>

          <div class="motm-vote-form">
            <select v-model="selectedMotmCandidateId" class="motm-candidate-select" :disabled="isUpdatingMotmVote"
              aria-label="Choose who deserves Man of the Match">
              <option value="" disabled>Choose a player…</option>
              <option v-for="player in motmCandidates" :key="player.ID" :value="player.ID">
                {{ formatPlayerNameForDisplay(player.Name) }}
              </option>
            </select>
            <button @click="castMotmVote" class="btn-base btn-primary btn-small"
              :disabled="isUpdatingMotmVote || !selectedMotmCandidateId">
              <div v-if="isUpdatingMotmVote" class="loading-spinner-small"></div>
              {{ isUpdatingMotmVote ? 'Saving...' : (hasVotedForMotm ? 'Change vote' : 'Vote') }}
            </button>
            <button v-if="hasVotedForMotm" @click="removeMotmVoteAction" class="btn-base btn-secondary btn-small"
              :disabled="isUpdatingMotmVote">
              Remove vote
            </button>
          </div>

          <div v-if="isLoadingMotmVotes" class="motm-loading">
            <div class="loading-spinner-small"></div>
          </div>
          <ul v-else-if="motmVotes.Tally.length > 0" class="motm-tally">
            <li v-for="candidate in motmVotes.Tally" :key="candidate.PlayerID" class="motm-tally-entry">
              <span class="motm-tally-name">{{ formatPlayerNameForDisplay(candidate.Name) }}</span>
              <span class="motm-tally-votes">{{ candidate.Votes }} vote{{ candidate.Votes === 1 ? '' : 's' }}</span>
            </li>
          </ul>
          <p v-else class="motm-empty">Nobody has voted yet.</p>
        </div>
      </div>
    </div>

    <!-- Error State -->
    <div v-else class="error-state">
      <div class="empty-content">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <line x1="15" y1="9" x2="9" y2="15" />
          <line x1="9" y1="9" x2="15" y2="15" />
        </svg>
        <h3 class="empty-title">Match not found</h3>
        <p class="empty-description">The match you're looking for doesn't exist or failed to load.</p>
        <button @click="goBack" class="btn-base btn-primary">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="15,18 9,12 15,6" />
          </svg>
          Go Back
        </button>
      </div>
    </div>

    <!-- Compose-choice modal — "Fill teams from sign-ups" opens this rather
         than acting immediately: there are now two ways to build the roster
         from the sign-up list, and this is where an admin picks one. -->
    <div v-if="showComposeChoiceModal" class="modal-overlay" @click="closeComposeChoiceModal">
      <div class="modal-container compose-choice-modal" @click.stop>
        <div class="modal-header">
          <h3>Compose the teams</h3>
          <button @click="closeComposeChoiceModal" class="modal-close">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
        <div class="modal-body compose-choice-body">
          <!-- Hidden, not disabled, once nothing is left to auto-place —
               see allConfirmedAlreadyPlaced. -->
          <button v-if="!allConfirmedAlreadyPlaced" @click="chooseAutoFill" class="compose-choice-option compose-choice-auto">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="compose-choice-icon">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
              <circle cx="9" cy="7" r="4" />
              <polyline points="16,11 18,13 22,9" />
            </svg>
            <span class="compose-choice-text">
              <strong>Auto-split</strong>
              <!-- The single most important sentence in this option: it only
                   fills the two team tabs below, and an admin who walks away
                   without pressing Save Changes loses the lot. -->
              <span>
                Splits the {{ confirmedRegistrations.length }} confirmed
                player{{ confirmedRegistrations.length === 1 ? '' : 's' }} across the two teams in sign-up order.
                Anyone already in a team stays where they are. Nothing is saved — review the teams below, then press
                "Save Changes".
              </span>
            </span>
          </button>

          <button @click="chooseManual" class="compose-choice-option compose-choice-manual">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="compose-choice-icon">
              <circle cx="12" cy="12" r="10" />
              <path d="M16 12l-4-4-4 4" />
              <path d="M16 16l-4-4-4 4" />
            </svg>
            <span class="compose-choice-text">
              <strong>Build manually</strong>
              <span>
                Pick players yourself. Sign-ups appear first — confirmed before waiting — the rest of the group is
                still one search away.
              </span>
            </span>
          </button>
        </div>
      </div>
    </div>

    <!-- Enhanced Multi-Player Modal -->
    <div v-if="showAddPlayerModal" class="modal-overlay" @click="closeModal">
      <div class="enhanced-multi-player-modal" @click.stop>
        <div class="modal-header enhanced-modal-header">
          <h3>Add Players to {{ match.Teams[activeTeam].Name }} Team</h3>
          <button @click="closeModal" class="modal-close">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div class="modal-body enhanced-modal-body">
          <!-- Search Section -->
          <div class="search-section">
            <div class="form-group">
              <label for="playerSearch">Search Players</label>
              <div class="input-wrapper">
                <input v-model="playerSearchTerm" type="text" id="playerSearch" class="form-input"
                  placeholder="Search for players to add..." @input="onPlayerSearch">
                <div v-if="isLoadingPlayers" class="search-loading">
                  <div class="spinner-small"></div>
                </div>
              </div>
              <div v-if="showCreatePlayerOption" class="create-player-section">
                <div class="create-player-prompt">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="info-icon">
                    <circle cx="12" cy="12" r="10" />
                    <line x1="12" y1="16" x2="12" y2="12" />
                    <line x1="12" y1="8" x2="12.01" y2="8" />
                  </svg>
                  <span>Player "{{ formatPlayerNameForDisplay(playerSearchTerm) }}" not found.</span>
                </div>
                <button @click="createNewPlayer" :disabled="isCreatingPlayer" class="create-player-btn">
                  <svg v-if="!isCreatingPlayer" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                    <circle cx="8.5" cy="7" r="4" />
                    <line x1="20" y1="8" x2="20" y2="14" />
                    <line x1="23" y1="11" x2="17" y2="11" />
                  </svg>
                  <div v-else class="spinner-small"></div>
                  {{ isCreatingPlayer ? 'Creating...' : `Create "${formatPlayerNameForDisplay(playerSearchTerm)}"` }}
                </button>
              </div>
            </div>
          </div>

          <!-- Two-column layout -->
          <div class="players-columns">
            <!-- Selected Players Column -->
            <div class="column selected-column">
              <div class="column-header">
                <h4>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="header-icon">
                    <path d="M9 12l2 2 4-4" />
                    <circle cx="12" cy="12" r="10" />
                  </svg>
                  Selected Players
                  <span class="count-badge">{{ selectedPlayers.length }}</span>
                </h4>
              </div>

              <div class="column-content custom-scrollbar">
                <div v-if="selectedPlayers.length === 0" class="empty-state">
                  <div class="empty-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="12" cy="12" r="10" />
                      <line x1="12" y1="8" x2="12" y2="16" />
                      <line x1="8" y1="12" x2="16" y2="12" />
                    </svg>
                  </div>
                  <p>No players selected</p>
                  <span>Select players from the right to add them</span>
                </div>

                <div v-else class="selected-players-list">
                  <div v-for="(player, index) in selectedPlayers" :key="`selected-${player.ID || player.Name}`"
                    class="selected-player-item">
                    <div class="player-info">
                      <span class="player-name">{{ formatPlayerNameForDisplay(player.Name) }}</span>
                    </div>
                    <button @click="removeSelectedPlayer(index)" class="remove-selected-btn">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <line x1="18" y1="6" x2="6" y2="18" />
                        <line x1="6" y1="6" x2="18" y2="18" />
                      </svg>
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <!-- Available Players Column -->
            <div class="column available-column">
              <div class="column-header">
                <h4>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="header-icon">
                    <circle cx="12" cy="12" r="10" />
                    <path d="M16 12l-4-4-4 4" />
                    <path d="M16 16l-4-4-4 4" />
                  </svg>
                  Available Players
                  <span class="count-badge">{{ filteredAvailablePlayers.length }}</span>
                </h4>
              </div>

              <div class="column-content custom-scrollbar">
                <div v-if="filteredAvailablePlayers.length === 0 && !isLoadingPlayers" class="empty-state">
                  <div class="empty-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="12" cy="12" r="10" />
                      <line x1="4.93" y1="4.93" x2="19.07" y2="19.07" />
                    </svg>
                  </div>
                  <p v-if="playerSearchTerm">No players found</p>
                  <p v-else>No available players</p>
                  <span v-if="playerSearchTerm">Try adjusting your search term</span>
                </div>

                <div v-else class="available-players-list">
                  <button v-for="player in filteredAvailablePlayers" :key="`available-${player.ID || player.Name}`"
                    @click="addPlayerToSelection(player)" class="available-player-item"
                    :disabled="isPlayerSelected(player) || isPlayerInAnyTeam(player.Name)">
                    <div class="player-info">
                      <span class="player-name">{{ formatPlayerNameForDisplay(player.Name) }}</span>
                      <!-- Only present for a scheduled match with no search
                           term active — see tierCandidatesBySignup. -->
                      <span v-if="player.registrationBadge" class="registration-badge"
                        :class="{ waiting: player.registrationBadge.waiting }">
                        #{{ player.registrationBadge.position }}{{ player.registrationBadge.waiting ? ' waiting' : '' }}
                      </span>
                    </div>
                    <div class="player-status">
                      <span v-if="isPlayerSelected(player)" class="status-selected">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <path d="M9 12l2 2 4-4" />
                          <circle cx="12" cy="12" r="10" />
                        </svg>
                      </span>
                      <span v-else-if="isPlayerInAnyTeam(player.Name)" class="status-in-match">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <circle cx="12" cy="12" r="10" />
                          <line x1="15" y1="9" x2="9" y2="15" />
                          <line x1="9" y1="9" x2="15" y2="15" />
                        </svg>
                      </span>
                      <span v-else class="status-available">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <circle cx="12" cy="12" r="10" />
                          <line x1="12" y1="8" x2="12" y2="16" />
                          <line x1="8" y1="12" x2="16" y2="12" />
                        </svg>
                      </span>
                    </div>
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div class="modal-footer enhanced-footer">
            <div class="footer-info">
              <div class="selection-summary">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="summary-icon">
                  <path d="M9 12l2 2 4-4" />
                  <circle cx="12" cy="12" r="10" />
                </svg>
                <span v-if="selectedPlayers.length > 0" class="selection-count">
                  {{ selectedPlayers.length }} player{{ selectedPlayers.length !== 1 ? 's' : '' }} selected
                </span>
                <span v-else class="selection-count empty">
                  No players selected
                </span>
              </div>
            </div>
            <div class="footer-buttons">
              <button @click="closeModal" class="btn-base btn-cancel">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
                Cancel
              </button>
              <button @click="addSelectedPlayersToTeam" :disabled="selectedPlayers.length === 0"
                class="btn-base btn-primary enhanced-confirm">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M9 12l2 2 4-4" />
                  <circle cx="12" cy="12" r="10" />
                </svg>
                Add {{ selectedPlayers.length || '' }} Player{{ selectedPlayers.length !== 1 ? 's' : '' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Toast Container -->
    <div v-if="message" class="toast-container">
      <div :class="['toast-message', messageType]" @click="dismissMessage" :key="messageKey">
        {{ message }}
      </div>
    </div>
  </div>
</template>

<script>
import {
  getMatchDetailsByID,
  updateMatch,
  deleteMatch,
  getGroupMembers,
  createPlayer,
  getMatchRegistrations,
  registerForMatch,
  unregisterFromMatch,
  closeMatchRegistrations,
  reopenMatchRegistrations,
  getMatchVotes,
  voteForMotm,
  removeMotmVote,
  getToken
} from '@/services/api';
import { resolveActiveGroup } from '@/services/activeGroup';
import { formatDateTimeForDisplay, formatCalendarDayLong } from '@/services/datetime';
import {
  isScheduledMatch,
  deriveRegistrationState,
  registrationsAreOpen,
  registrationStateLabel,
  fillTeamsFromRegistrations,
  teamsAreComposed,
  REGISTRATION_NOT_OPEN_YET,
  REGISTRATION_OPEN,
  REGISTRATION_CLOSED_BY_ADMIN,
  REGISTRATION_CLOSED_AT_KICKOFF
} from '@/services/matchRegistration';
import { buildWhatsAppShareText, buildWhatsAppShareUrl } from '@/services/whatsappShare';
import { encodeMatchId } from '@/services/shortLink';

// Same shape as Profile.vue's own helper — the app has no auth store, and the
// player id is only ever needed to answer "which of these rows is me", so both
// pages decode the JWT payload locally rather than a third place holding state.
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
  name: 'MatchDetail',
  data() {
    return {
      match: null,
      matchSnapshot: null,
      isLoading: true,
      isSaving: false,
      isDeleting: false,
      // Resolved once on load, reused when loading the match (scoped to the
      // same group) and when deleting it.
      activeGroupId: '',
      // Whether the caller is an admin of the active group — gates every
      // editing control (add/remove player, goals, save, delete). Editing and
      // deleting a match are admin-only on the backend (RequireGroupAdmin in
      // main.go); a non-admin can still view the match read-only.
      isAdmin: false,
      activeTeam: 0,
      showAddPlayerModal: false,
      // "Fill teams from sign-ups" opens this chooser rather than acting
      // immediately — see canFillTeamsFromSignups/allConfirmedAlreadyPlaced.
      showComposeChoiceModal: false,
      message: '',
      messageType: 'success',
      // New properties for player dropdown
      allPlayers: [],
      selectedPlayers: [],
      filteredAvailablePlayers: [],
      isLoadingPlayers: false,
      playerSearchTerm: '',
      messageKey: 0,
      showCreatePlayerOption: false,
      isCreatingPlayer: false,
      // --- Sign-ups (scheduled matches only) ---
      // The ordered list from GET /matches/:id/registrations, exactly as the
      // server sent it: never reordered or patched locally, since the whole
      // waiting-list design is server-derived ordering.
      registrations: [],
      isLoadingRegistrations: false,
      // Participate/Withdraw in flight vs. Close/Reopen in flight — separate
      // flags so an admin closing the list doesn't grey out the member-facing
      // buttons (and vice versa).
      isUpdatingRegistration: false,
      isUpdatingRegistrationState: false,
      // Kept out of `match` on purpose. Close/Reopen change this flag
      // server-side, and writing it back into `match` would make
      // hasUnsavedChanges() see the match as dirty against matchSnapshot and
      // pop the "leave without saving?" confirm — or, if the snapshot were
      // refreshed too, silently swallow real unsaved goal edits. Seeded from
      // the match on load, then owned here.
      registrationsClosedAt: null,
      // Sampled ONCE, in created(). There is deliberately no polling timer and
      // no reactive clock: this app has no reactive store and re-reads its data
      // on navigation, so a player who had this page open from before sign-ups
      // opened sees the Participate button after a reload — which is also how
      // they find out at all, since nothing notifies them.
      nowMs: 0,
      currentPlayerId: '',
      // --- Man of the Match voting ---
      // { Tally: [{PlayerID, Name, Votes}], MyVoteFor } exactly as
      // GET /matches/:id/votes sent it. Like the sign-up list, always fully
      // re-fetched rather than patched locally after a vote/unvote — the
      // whole point is showing the real tally, not a locally-guessed one.
      motmVotes: { Tally: [], MyVoteFor: null },
      isLoadingMotmVotes: false,
      isUpdatingMotmVote: false,
      // The candidate currently selected in the dropdown; seeded from the
      // caller's existing vote (if any) each time the tally reloads, so
      // "Change vote" opens on what is currently true rather than blank.
      selectedMotmCandidateId: '',
    };
  },
  async created() {
    this.nowMs = Date.now();
    this.currentPlayerId = currentPlayerIdFromToken();
    try {
      const { groups, activeGroupId } = await resolveActiveGroup();
      this.activeGroupId = activeGroupId;
      this.isAdmin = groups.find(g => g.id === activeGroupId)?.role === 'admin';
    } catch (error) {
      // Degrade instead of breaking the page: no group_id (backend's own
      // first-group fallback) and isAdmin left false, which just hides the
      // admin-only controls.
      console.error('Error resolving the active group:', error);
    }
    await this.loadMatch();
  },
  beforeRouteLeave(to, from, next) {
    if (this.hasUnsavedChanges()) {
      const leave = window.confirm('You have unsaved changes. Leave without saving?');
      next(leave);
      return;
    }
    next();
  },
  computed: {
    // A scheduled match is one that carries a kick-off; an ordinary match has
    // none of the scheduling keys at all.
    isScheduled() {
      return isScheduledMatch(this.match);
    },

    // Gates the score header and the whole roster/team-management block
    // below it (tabs, Add Player, the player list): an unscheduled match has
    // always had a populated roster from the moment it's created, so this is
    // always true for one and changes nothing. A scheduled match starts (and
    // stays, through the whole sign-up window) with both teams empty — that's
    // its normal state, not a placeholder worth rendering — so those stay
    // hidden until an admin has actually put someone on a team, whether by
    // hand or via "Fill teams from sign-ups". Save Changes/Delete Match are
    // deliberately NOT behind this gate (see the template): an admin needs to
    // be able to cancel a still-empty scheduled match regardless.
    showTeamRoster() {
      return !this.isScheduled || teamsAreComposed(this.match);
    },

    // Gates the Man of the Match panel: any match — scheduled or not — with
    // at least one player actually placed on a team. Unlike showTeamRoster,
    // this is not "or unscheduled": an unscheduled match's roster is
    // essentially always composed already, but the rule here is simply
    // "is there anyone to vote for", so it reuses teamsAreComposed directly.
    isRosterComposed() {
      return teamsAreComposed(this.match);
    },

    // Every roster player across both teams, deduplicated by id, minus the
    // caller themselves — MatchVoteService rejects a self-vote outright
    // (ErrCannotVoteForSelf), so offering yourself in the dropdown would
    // only set a member up for a 400 with nothing to gain from it.
    motmCandidates() {
      if (!this.match || !Array.isArray(this.match.Teams)) return [];
      const seen = new Set();
      const candidates = [];
      this.match.Teams.forEach(team => {
        (team.Players || []).forEach(player => {
          if (!player.ID || player.ID === this.currentPlayerId || seen.has(player.ID)) return;
          seen.add(player.ID);
          candidates.push(player);
        });
      });
      return candidates;
    },

    hasVotedForMotm() {
      return Boolean(this.motmVotes.MyVoteFor);
    },

    // Looked up from the tally rather than stored separately: the caller's
    // own vote is guaranteed to appear there (a tally entry exists for every
    // candidate with at least one vote, and casting one is exactly that).
    myMotmVoteName() {
      if (!this.motmVotes.MyVoteFor) return '';
      const entry = this.motmVotes.Tally.find(candidate => candidate.PlayerID === this.motmVotes.MyVoteFor);
      return entry ? entry.Name : '';
    },

    // The closed flag comes from local state rather than from `match` (see the
    // data comment on registrationsClosedAt); everything else is read straight
    // off the match.
    registrationState() {
      if (!this.match) return '';
      return deriveRegistrationState({
        ScheduledAt: this.match.ScheduledAt,
        RegistrationOpensAt: this.match.RegistrationOpensAt,
        RegistrationsClosedAt: this.registrationsClosedAt
      }, this.nowMs);
    },

    registrationsOpen() {
      return registrationsAreOpen(this.registrationState);
    },

    kickoffLabel() {
      return formatDateTimeForDisplay(this.match && this.match.ScheduledAt);
    },

    registrationStateLabel() {
      return registrationStateLabel(this.registrationState);
    },

    // Mirrors MatchesPanel.vue's own signupCountLabel wording ("X / 16 signed
    // up") so the two views never disagree — but reads count from the live
    // `registrations` list this page already loads, rather than
    // match.RegistrationCount, which nothing here ever refreshes locally.
    signupCountLabel() {
      return `${this.registrations.length} / ${this.match.MaxPlayers} signed up`;
    },

    // The full /matches/:id/edit URL reads badly in a chat next to "Join
    // Calciotto" — window.location.origin + the encoded id is the same
    // "tinylink" the /m/:code route (router/index.js) decodes back.
    whatsappShareUrl() {
      const shortUrl = `${window.location.origin}/m/${encodeMatchId(this.match.ID)}`;
      const text = buildWhatsAppShareText({
        kickoffLabel: this.kickoffLabel,
        matchUrl: shortUrl
      });
      return buildWhatsAppShareUrl(text);
    },

    // Deliberately says nothing about being told when sign-ups open: there is
    // no notification of any kind in this feature, so the copy must not imply
    // one — players find out by opening the app.
    registrationStateDetail() {
      switch (this.registrationState) {
        case REGISTRATION_OPEN:
          return 'Sign-ups are open until an admin closes them. Going over the maximum is fine — extra players join the waiting list.';
        case REGISTRATION_NOT_OPEN_YET:
          return `Sign-ups open on ${formatDateTimeForDisplay(this.match.RegistrationOpensAt)}. Nothing will notify you, so check back here then.`;
        case REGISTRATION_CLOSED_BY_ADMIN:
          return 'An admin has closed sign-ups. You can no longer sign up or withdraw.';
        case REGISTRATION_CLOSED_AT_KICKOFF:
          return 'Kick-off has passed, so sign-ups are closed for good.';
        default:
          return '';
      }
    },

    // Both of these hang off the same `registrationsOpen`, because the backend
    // gates signing up and withdrawing on the identical window.
    isRegistered() {
      if (!this.currentPlayerId) return false;
      return this.registrations.some(entry => entry.PlayerID === this.currentPlayerId);
    },

    canParticipate() {
      return this.isScheduled && this.registrationsOpen && !this.isLoadingRegistrations && !this.isRegistered;
    },

    canWithdraw() {
      return this.isScheduled && this.registrationsOpen && !this.isLoadingRegistrations && this.isRegistered;
    },

    // Admin-only, and only in the state each action can actually change: there
    // is nothing to close on an already-closed list, and reopening one that
    // kick-off closed would clear the flag without reopening anything.
    canCloseRegistrations() {
      return this.isScheduled && this.isAdmin && this.registrationState === REGISTRATION_OPEN;
    },

    canReopenRegistrations() {
      return this.isScheduled && this.isAdmin && this.registrationState === REGISTRATION_CLOSED_BY_ADMIN;
    },

    // "Fill teams from sign-ups" is offered to an admin only once the list is
    // closed — either by them (closed-by-admin) or by kick-off passing
    // (closed-at-kickoff), the two states registrationsAreOpen() rejects. The
    // roster is composed when it is final, which is the flow the product
    // described: an admin closes sign-ups *in order to* pick the teams.
    // Gates the button that opens the compose-choice modal (Auto-split vs
    // Build manually), not either action itself — the button stays offered
    // even once every confirmed sign-up already has a team, because "build
    // manually" (add a late substitute, reassign someone from the waiting
    // list, etc.) is still meaningful then. allConfirmedAlreadyPlaced instead
    // gates only the Auto-split *option* inside the modal — see there.
    canFillTeamsFromSignups() {
      return this.isScheduled
        && this.isAdmin
        && !this.isLoadingRegistrations
        && (this.registrationState === REGISTRATION_CLOSED_BY_ADMIN
          || this.registrationState === REGISTRATION_CLOSED_AT_KICKOFF);
    },

    // True once every confirmed sign-up already has a team — offering
    // Auto-split at that point would promise composition work there's none
    // left to do (running it would be a same-state no-op). Reuses
    // fillTeamsFromRegistrations itself — the exact function Auto-split
    // calls — rather than re-deriving "already placed" a second way that
    // could quietly drift from it.
    //
    // Deliberately narrower than "nothing to fill": an empty confirmed list
    // returns false here on purpose, leaving Auto-split offered and
    // fillTeamsFromSignups()'s own "nothing to fill the teams with" error
    // message intact — an admin who closed sign-ups on an empty list should
    // find out why clicking does nothing, not have the option vanish with no
    // explanation.
    allConfirmedAlreadyPlaced() {
      if (this.confirmedRegistrations.length === 0) return false;
      if (!this.match.Teams || this.match.Teams.length < 2) return false;
      const rosters = this.match.Teams.map(team => team.Players || []);
      return fillTeamsFromRegistrations(this.confirmedRegistrations, rosters).addedCount === 0;
    },

    confirmedRegistrations() {
      return this.registrations.filter(entry => !entry.IsWaiting);
    },

    waitingRegistrations() {
      return this.registrations.filter(entry => entry.IsWaiting);
    }
  },
  methods: {
    hasUnsavedChanges() {
      if (this.isSaving || !this.match) return false;
      return JSON.stringify(this.match) !== this.matchSnapshot;
    },

    async loadMatch() {
      this.isLoading = true;
      try {
        const matchId = this.$route.params.id;
        // GetMatchDetailsByID filters on (match id, group id), so a match from
        // another group reads as "not found" — which is what we want: the URL
        // alone must not pull a match out of a group we aren't scoped to.
        // updateMatch needs no group of its own; it posts back this.match,
        // whose GroupID already comes from this response.
        this.match = await getMatchDetailsByID(matchId, this.activeGroupId);

        // Ensure each player has GoalNumber property
        if (this.match && this.match.Teams) {
          this.match.Teams.forEach(team => {
            if (team.Players) {
              team.Players.forEach(player => {
                if (!player.GoalNumber) {
                  player.GoalNumber = 0;
                }
              });
            }
          });
        }

        this.matchSnapshot = JSON.stringify(this.match);

        // Snapshot the closed flag out of the match and into local state, so
        // Close/Reopen never touch `match` (and so never make it look dirty).
        this.registrationsClosedAt = (this.match && this.match.RegistrationsClosedAt) || null;
      } catch (error) {
        console.error('Error fetching match:', error);
        this.showMessage('Error loading match details', 'error');
      } finally {
        this.isLoading = false;
      }

      // After isLoading flips, so the panel renders its own spinner rather than
      // holding the whole page back on a list that no unscheduled match has.
      if (this.isScheduled) {
        await this.loadRegistrations();
      }
      // Same reasoning: only a match with an actual roster has anything to
      // vote on, and this panel renders its own spinner too.
      if (teamsAreComposed(this.match)) {
        await this.loadMotmVotes();
      }
    },

    // --- Sign-ups -----------------------------------------------------------
    // Always a full re-fetch, never a local patch: position and IsWaiting are
    // derived server-side from the stored order, and the promotion of the first
    // reserve when someone withdraws is exactly the thing the list is meant to
    // show happening. Patching locally is how the display drifts from reality.
    async loadRegistrations() {
      this.isLoadingRegistrations = true;
      try {
        const entries = await getMatchRegistrations(this.match.ID);
        this.registrations = Array.isArray(entries) ? entries : [];
      } catch (error) {
        console.error('Error loading registrations:', error);
        this.showMessage('Error loading the sign-up list', 'error');
        this.registrations = [];
      } finally {
        this.isLoadingRegistrations = false;
      }
    },

    async participate() {
      if (this.isUpdatingRegistration) return;
      this.isUpdatingRegistration = true;
      try {
        // Landing past the maximum is a success, not an error — the entry that
        // comes back is the only place the caller learns it, so say so
        // explicitly instead of a bare "signed up".
        const entry = await registerForMatch(this.match.ID);
        await this.loadRegistrations();
        if (entry && entry.IsWaiting) {
          this.showMessage(`Signed up — you are #${entry.Position}, on the waiting list.`, 'success');
        } else {
          const position = entry && entry.Position ? ` You are #${entry.Position}.` : '';
          this.showMessage(`You're in for this match.${position}`, 'success');
        }
      } catch (error) {
        console.error('Error signing up for match:', error);
        // A 409 says precisely why (not open yet, closed, already registered) —
        // worth showing verbatim rather than flattening, same pattern as
        // createNewPlayer above. Reload either way: a rejection usually means
        // this page's view of the list is stale.
        this.showMessage(this.registrationErrorMessage(error, 'Error signing up for this match.'), 'error');
        await this.loadRegistrations();
      } finally {
        this.isUpdatingRegistration = false;
      }
    },

    // Same window as signing up, so the button this confirms is only ever
    // rendered while the list is open — see canWithdraw.
    confirmWithdraw() {
      if (this.isUpdatingRegistration) return;
      const confirmed = window.confirm('Withdraw from this match? Your place goes to the first player on the waiting list.');
      if (!confirmed) return;
      this.withdraw();
    },

    async withdraw() {
      this.isUpdatingRegistration = true;
      try {
        await unregisterFromMatch(this.match.ID);
        await this.loadRegistrations();
        this.showMessage('You have withdrawn from this match.', 'success');
      } catch (error) {
        console.error('Error withdrawing from match:', error);
        this.showMessage(this.registrationErrorMessage(error, 'Error withdrawing from this match.'), 'error');
        await this.loadRegistrations();
      } finally {
        this.isUpdatingRegistration = false;
      }
    },

    async closeRegistrations() {
      if (this.isUpdatingRegistrationState) return;
      this.isUpdatingRegistrationState = true;
      try {
        await closeMatchRegistrations(this.match.ID);
        // The backend stamps its own timestamp; only the fact that one exists
        // matters to the state derivation, so a local instant is enough until
        // the next load reads the real one back.
        this.registrationsClosedAt = new Date().toISOString();
        this.showMessage('Sign-ups closed.', 'success');
      } catch (error) {
        console.error('Error closing sign-ups:', error);
        this.showMessage(this.registrationErrorMessage(error, 'Error closing sign-ups.'), 'error');
      } finally {
        this.isUpdatingRegistrationState = false;
      }
    },

    async reopenRegistrations() {
      if (this.isUpdatingRegistrationState) return;
      this.isUpdatingRegistrationState = true;
      try {
        await reopenMatchRegistrations(this.match.ID);
        this.registrationsClosedAt = null;
        this.showMessage('Sign-ups reopened.', 'success');
      } catch (error) {
        console.error('Error reopening sign-ups:', error);
        this.showMessage(this.registrationErrorMessage(error, 'Error reopening sign-ups.'), 'error');
      } finally {
        this.isUpdatingRegistrationState = false;
      }
    },

    registrationErrorMessage(error, fallback) {
      return error?.response?.data?.error || fallback;
    },

    // --- Man of the Match voting ---------------------------------------
    // Always a full re-fetch after a change, same reasoning as
    // loadRegistrations: the tally is server-derived and patching it locally
    // is how the display drifts from reality.
    async loadMotmVotes() {
      this.isLoadingMotmVotes = true;
      try {
        const summary = await getMatchVotes(this.match.ID);
        this.motmVotes = summary && Array.isArray(summary.Tally)
          ? summary
          : { Tally: [], MyVoteFor: null };
        this.selectedMotmCandidateId = this.motmVotes.MyVoteFor || '';
      } catch (error) {
        console.error('Error loading Man of the Match votes:', error);
        this.showMessage('Error loading Man of the Match votes', 'error');
        this.motmVotes = { Tally: [], MyVoteFor: null };
      } finally {
        this.isLoadingMotmVotes = false;
      }
    },

    // Casting a vote is an upsert on the backend (MatchVoteService.Vote), so
    // this is used for both the first vote and every later change of mind —
    // there is no separate "conflict" path to handle, unlike registerForMatch.
    async castMotmVote() {
      if (this.isUpdatingMotmVote || !this.selectedMotmCandidateId) return;
      this.isUpdatingMotmVote = true;
      try {
        await voteForMotm(this.match.ID, this.selectedMotmCandidateId);
        await this.loadMotmVotes();
        this.showMessage('Your Man of the Match vote has been saved.', 'success');
      } catch (error) {
        console.error('Error casting Man of the Match vote:', error);
        this.showMessage(this.registrationErrorMessage(error, 'Error casting your vote.'), 'error');
        await this.loadMotmVotes();
      } finally {
        this.isUpdatingMotmVote = false;
      }
    },

    // A no-op success on the backend even with no existing vote, so there is
    // nothing to confirm here the way confirmWithdraw does — removing a vote
    // has no effect on anyone else's place the way withdrawing does.
    async removeMotmVoteAction() {
      if (this.isUpdatingMotmVote) return;
      this.isUpdatingMotmVote = true;
      try {
        await removeMotmVote(this.match.ID);
        this.selectedMotmCandidateId = '';
        await this.loadMotmVotes();
        this.showMessage('Your Man of the Match vote has been removed.', 'success');
      } catch (error) {
        console.error('Error removing Man of the Match vote:', error);
        this.showMessage(this.registrationErrorMessage(error, 'Error removing your vote.'), 'error');
      } finally {
        this.isUpdatingMotmVote = false;
      }
    },

    // Populates the two team rosters from the confirmed sign-up list and stops
    // there. It writes nothing to the server on purpose: mutating `match` marks
    // it dirty, so the existing "Save Changes" button persists it through the
    // PUT /matches/:id diff (which already creates/updates/deletes match_players
    // rows), and the existing beforeRouteLeave guard catches an admin who walks
    // away. That also means the split can be corrected by hand before anything
    // is written — the whole point, since this is a mechanical split and not a
    // balanced one.
    // The "Fill teams from sign-ups" button's own click target: offers a
    // choice rather than acting immediately, since there are now two ways to
    // compose the roster from the sign-up list — see the modal below.
    openComposeChoice() {
      if (!this.canFillTeamsFromSignups) return;
      this.showComposeChoiceModal = true;
    },

    closeComposeChoiceModal() {
      this.showComposeChoiceModal = false;
    },

    chooseAutoFill() {
      this.showComposeChoiceModal = false;
      this.fillTeamsFromSignups();
    },

    // Opens the existing Add Player modal — the same one the per-team "+"
    // icon opens once a roster exists — which is what makes this the bridge
    // across the one gap that icon can't reach on its own: before any player
    // has been placed, .team-management (and its "+" icon) stays hidden by
    // showTeamRoster, but this button lives in .signup-actions, a sibling
    // that's never behind that gate. filterAvailablePlayers() is what makes
    // the modal itself worth opening here rather than at the icon alone: for
    // a scheduled match it always tiers sign-ups first, regardless of which
    // of the two entry points opened it.
    async chooseManual() {
      this.showComposeChoiceModal = false;
      await this.showModal();
    },

    fillTeamsFromSignups() {
      if (!this.canFillTeamsFromSignups) return;
      if (!this.match || !this.match.Teams || this.match.Teams.length < 2) return;

      const { rosters, addedCount, skippedCount } = fillTeamsFromRegistrations(
        this.registrations,
        this.match.Teams.map(team => team.Players || [])
      );

      if (addedCount === 0) {
        this.showMessage(
          skippedCount > 0
            ? 'Everyone on the confirmed sign-up list is already in a team.'
            : 'Nobody is on the confirmed sign-up list, so there is nothing to fill the teams with.',
          'error'
        );
        return;
      }

      this.match.Teams.forEach((team, index) => {
        team.Players = rosters[index];
      });
      // Scores are deliberately left alone: every player added here starts at 0
      // goals, so each team's sum is unchanged, and recomputing would risk
      // overwriting a score an admin has already edited on this page.

      const skipped = skippedCount > 0
        ? ` ${skippedCount} already in a team ${skippedCount === 1 ? 'was' : 'were'} left where they are.`
        : '';
      this.showMessage(
        `Added ${addedCount} player${addedCount === 1 ? '' : 's'} to the teams.${skipped} Nothing is saved yet — press "Save Changes".`,
        'success'
      );
    },

    // Load all players when modal opens
    async loadAllPlayers() {
      if (this.allPlayers && this.allPlayers.length > 0) {
        this.filterAvailablePlayers();
        return;
      }

      this.isLoadingPlayers = true;
      try {
        await this.reloadAllPlayers();
      } finally {
        this.isLoadingPlayers = false;
      }
    },

    // Filter players to show only those not in current team
    filterAvailablePlayers() {
      if (!this.allPlayers || !Array.isArray(this.allPlayers) || this.allPlayers.length === 0) {
        this.filteredAvailablePlayers = [];
        this.checkCreatePlayerOption();
        return;
      }

      if (!this.match || !this.match.Teams || !this.match.Teams[this.activeTeam]) {
        this.filteredAvailablePlayers = [];
        this.checkCreatePlayerOption();
        return;
      }

      const currentTeamPlayers = this.match.Teams[this.activeTeam].Players || [];
      const currentTeamPlayerNames = currentTeamPlayers.map(p => p.Name.toLowerCase());

      const candidates = this.allPlayers.filter(player =>
        player && player.Name && !currentTeamPlayerNames.includes(player.Name.toLowerCase())
      );

      const searchTerm = this.playerSearchTerm && this.playerSearchTerm.trim().toLowerCase();

      // Once there's something to search for, it searches the whole group —
      // sign-up or not. Tiering only shapes the *default*, empty-search view.
      if (searchTerm) {
        this.filteredAvailablePlayers = candidates.filter(player =>
          player.Name.toLowerCase().includes(searchTerm)
        );
        this.checkCreatePlayerOption();
        return;
      }

      this.filteredAvailablePlayers = this.isScheduled
        ? this.tierCandidatesBySignup(candidates)
        : candidates;
      this.checkCreatePlayerOption();
    },

    // For a scheduled match, with no search term: confirmed sign-ups first
    // (in Position order), then the waiting list, and the rest of the group
    // left out of this default view entirely — reachable the moment a search
    // term narrows the list above. The people most likely to actually play
    // are what an admin composing by hand wants in front of them; a group
    // that outgrew its regular sign-ups would otherwise bury them in a list
    // of everyone who's ever joined.
    //
    // registrationBadge carries just enough for the template to render a
    // marker (`#<position>`, muted when waiting) without it re-deriving
    // anything position-related itself — that stays server-derived, same
    // rule as everywhere else this feature touches Position/IsWaiting.
    tierCandidatesBySignup(candidates) {
      const byId = new Map(candidates.map(player => [player.ID, player]));
      const tiered = [];
      const placed = new Set();

      const addTier = (entries, waiting) => {
        entries.forEach(entry => {
          const player = byId.get(entry.PlayerID);
          if (player && !placed.has(player.ID)) {
            tiered.push({ ...player, registrationBadge: { position: entry.Position, waiting } });
            placed.add(player.ID);
          }
        });
      };

      addTier(this.confirmedRegistrations, false);
      addTier(this.waitingRegistrations, true);

      return tiered;
    },

    // Check if we should show the create player option
    checkCreatePlayerOption() {
      if (!this.playerSearchTerm || this.playerSearchTerm.trim().length < 2) {
        this.showCreatePlayerOption = false;
        return;
      }

      const searchTerm = this.playerSearchTerm.trim().toLowerCase();
      const exactMatch = this.allPlayers.some(player =>
        player.Name.toLowerCase() === searchTerm
      );

      // Show create option if no exact match found and search term is not empty
      this.showCreatePlayerOption = !exactMatch && this.filteredAvailablePlayers.length === 0;
    },

    // Create a new player
    async createNewPlayer() {
      if (!this.playerSearchTerm || this.playerSearchTerm.trim().length < 2) {
        this.showMessage('Please enter a valid player name', 'error');
        return;
      }

      const playerName = this.playerSearchTerm.trim();
      const playerNameLowerCase = playerName.toLowerCase(); // Backend gets lowercase

      // Check if player already exists (case-insensitive)
      const existingPlayer = this.allPlayers.find(player =>
        player.Name.toLowerCase() === playerNameLowerCase
      );

      if (existingPlayer) {
        this.showMessage('Player already exists', 'error');
        return;
      }

      this.isCreatingPlayer = true;
      try {
        const newPlayerData = {
          Name: playerNameLowerCase, // Send lowercase to backend
          // Exact key, lowercase with an underscore: CreatePlayer binds this
          // into a `json:"group_id"` field, and Gin's case-insensitive JSON
          // binding does not bridge the underscore — sending `GroupID`
          // instead would silently fail to bind and the new player would
          // fall through to the backend's own-first-group fallback instead
          // of joining this match's group.
          group_id: this.match.GroupID
        };

        await createPlayer(newPlayerData);

        // RELOAD ALL PLAYERS FROM DATABASE to ensure we have fresh data
        await this.reloadAllPlayers();

        // Find the newly created player in the fresh data
        const freshPlayer = this.allPlayers.find(player =>
          player.Name.toLowerCase() === playerNameLowerCase
        );

        if (freshPlayer) {
          // Add the new player to selection immediately
          this.addPlayerToSelection(freshPlayer);
        }

        // Clear search and hide create option
        this.playerSearchTerm = '';
        this.showCreatePlayerOption = false;
        this.filterAvailablePlayers();

        this.showMessage(`Player "${this.formatPlayerNameForDisplay(playerNameLowerCase)}" created and added to selection!`, 'success');

      } catch (error) {
        console.error('Error creating player:', error);
        // The backend's own per-group duplicate-name check (a case our own
        // client-side check against this.allPlayers can't catch, since that
        // check is global rather than scoped to this match's group) returns a
        // specific message worth showing verbatim, same pattern as
        // ForgotPassword.vue/Profile.vue. Fall back to a generic message for
        // any other kind of failure.
        const backendMessage = error.response?.data?.error;
        this.showMessage(backendMessage || 'Error creating player. Please try again.', 'error');
      } finally {
        this.isCreatingPlayer = false;
      }
    },

    // Scoped to this match's own group (this.activeGroupId, resolved once in
    // created() via resolveActiveGroup()) rather than every player in the
    // database: a player removed from the group — or belonging to a
    // different group entirely — must not be offered here.
    async reloadAllPlayers() {
      if (!this.activeGroupId) {
        // No group resolved (resolveActiveGroup failed) — degrade to an
        // empty list rather than requesting a malformed URL.
        this.allPlayers = [];
        this.filterAvailablePlayers();
        return;
      }
      try {
        const members = await getGroupMembers(this.activeGroupId);
        // getGroupMembers returns PlayerWithRole, which embeds Player's own
        // lowercase JSON fields (id, name) plus role — translate to the
        // PascalCase {ID, Name} shape the rest of this component already
        // uses for players (matching PlayerCustom's convention from the
        // getPlayers() call this replaces), role is not needed here.
        this.allPlayers = Array.isArray(members)
          ? members.map(member => ({ ID: member.id, Name: member.name }))
          : [];
        this.filterAvailablePlayers();
      } catch (error) {
        console.error('Error reloading players:', error);
        this.showMessage('Error reloading players list', 'error');
        // Don't reset allPlayers on error, keep the current data
      }
    },

    // Handle search input
    onPlayerSearch() {
      // Clear any existing timeout
      if (this.searchTimeout) {
        clearTimeout(this.searchTimeout);
      }

      // Debounce the search
      this.searchTimeout = setTimeout(() => {
        this.filterAvailablePlayers();
      }, 300);
    },

    addPlayerToSelection(player) {
      if (this.isPlayerSelected(player) || this.isPlayerInAnyTeam(player.Name)) {
        return;
      }

      const playerToAdd = {
        ID: player.ID,
        Name: player.Name,
        initialGoals: 0  // Default to 0 goals
      };

      this.selectedPlayers.push(playerToAdd);
    },

    // Remove a player from selection
    removeSelectedPlayer(index) {
      this.selectedPlayers.splice(index, 1);
    },

    // Check if player is already selected
    isPlayerSelected(player) {
      return this.selectedPlayers.some(selectedPlayer =>
        selectedPlayer.Name.toLowerCase() === player.Name.toLowerCase()
      );
    },

    // Add all selected players to the team
    async addSelectedPlayersToTeam() {
      if (this.selectedPlayers.length === 0) {
        this.showMessage('Please select at least one player', 'error');
        return;
      }

      if (!this.match.Teams || !this.match.Teams[this.activeTeam]) {
        console.error('Invalid team data');
        return;
      }

      if (!this.match.Teams[this.activeTeam].Players) {
        this.match.Teams[this.activeTeam].Players = [];
      }

      // Add each selected player to the team
      this.selectedPlayers.forEach(selectedPlayer => {
        // Double-check if player is not already in the team
        if (!this.isPlayerInCurrentTeam(selectedPlayer.Name)) {
          const newPlayer = {
            ID: selectedPlayer.ID,
            Name: selectedPlayer.Name,
            GoalNumber: selectedPlayer.initialGoals || 0
          };

          this.match.Teams[this.activeTeam].Players.push(newPlayer);
        }
      });

      // Update team score
      this.updateTeamScore();

      // Close modal and show success message
      this.closeModal();
    },

    // Check if player is in current team
    isPlayerInCurrentTeam(playerName) {
      if (!this.match.Teams || !this.match.Teams[this.activeTeam] || !this.match.Teams[this.activeTeam].Players) {
        return false;
      }

      return this.match.Teams[this.activeTeam].Players.some(player =>
        player.Name.toLowerCase() === playerName.toLowerCase()
      );
    },

    // Check if player is in any team in this match
    isPlayerInAnyTeam(playerName) {
      if (!this.match.Teams) return false;

      return this.match.Teams.some(team =>
        team.Players && team.Players.some(player =>
          player.Name.toLowerCase() === playerName.toLowerCase()
        )
      );
    },

    // Show modal and load players
    async showModal() {
      this.showAddPlayerModal = true;
      await this.loadAllPlayers();
    },

    closeModal() {
      this.showAddPlayerModal = false;
      this.selectedPlayers = [];
      this.playerSearchTerm = '';
      this.filteredAvailablePlayers = [];
      this.showCreatePlayerOption = false;
      this.isCreatingPlayer = false;
      if (this.searchTimeout) {
        clearTimeout(this.searchTimeout);
      }
    },

    // Existing methods remain the same...
    goBack() {
      this.$router.go(-1);
    },

    updateGoals(playerIndex, change) {
      if (!this.match.Teams || !this.match.Teams[this.activeTeam] || !this.match.Teams[this.activeTeam].Players) {
        console.error('Invalid team or players data');
        return;
      }

      const player = this.match.Teams[this.activeTeam].Players[playerIndex];
      if (!player) {
        console.error('Player not found at index:', playerIndex);
        return;
      }

      const newGoals = (player.GoalNumber || 0) + change;

      if (newGoals >= 0) {
        player.GoalNumber = newGoals;
        this.updateTeamScore();
      }
    },

    updateTeamScore() {
      if (!this.match.Teams || !this.match.Teams[this.activeTeam] || !this.match.Teams[this.activeTeam].Players) {
        return;
      }

      const team = this.match.Teams[this.activeTeam];
      team.Score = team.Players.reduce((total, player) => total + (player.GoalNumber || 0), 0);
    },

    removePlayer(playerIndex) {
      if (!this.match.Teams || !this.match.Teams[this.activeTeam] || !this.match.Teams[this.activeTeam].Players) {
        console.error('Invalid team or players data');
        return;
      }

      this.match.Teams[this.activeTeam].Players.splice(playerIndex, 1);
      this.updateTeamScore();
    },

    hasEmptyTeam() {
      if (!this.match || !this.match.Teams) return true;

      return this.match.Teams.some(team =>
        !team.Players || team.Players.length === 0
      );
    },

    async saveChanges() {
      // This guard predates scheduled matches, back when a match's only
      // lifecycle was "an admin records a game that already happened" — an
      // empty team there really was a mistake worth catching. For a scheduled
      // match, an empty (or lopsided) team is the *normal* state right up
      // until sign-ups close and someone composes the roster: right after
      // creation both teams are empty by construction, and even a single
      // confirmed sign-up leaves one team empty until a second player joins
      // it. Blocking Save there would make it impossible to ever persist a
      // scheduled match's own schedule fields, or to save a partial manual
      // pick, without an unrelated roster requirement in the way. The backend
      // enforces no such minimum either — MatchService.UpdateMatch's diff
      // persists whatever teams/players it's given, including empty ones — so
      // this was purely a frontend-only restriction to lift.
      if (!this.isScheduled && this.hasEmptyTeam()) {
        this.showMessage('Each team requires at least 1 player', 'error');
        return;
      }
      this.isSaving = true;
      try {
        await updateMatch(this.match.ID, this.match);
        // Update the snapshot before navigating away: beforeRouteLeave's
        // hasUnsavedChanges() check runs as part of this same navigation,
        // and would otherwise see the just-saved match as still "dirty"
        // against the pre-save snapshot and pop the "leave without
        // saving?" confirm right after a successful save.
        this.matchSnapshot = JSON.stringify(this.match);
        this.goBack();
      } catch (error) {
        console.error('Error saving match:', error);
        this.showMessage('Error saving changes', 'error');
      } finally {
        this.isSaving = false;
      }
    },

    // Same confirm-before-acting pattern as beforeRouteLeave's unsaved-changes
    // guard above — a plain window.confirm rather than a custom modal, since
    // that's the existing precedent in this file for a destructive/irreversible
    // action the user could regret.
    confirmDeleteMatch() {
      if (this.isDeleting) return;
      const confirmed = window.confirm('Delete this match? This cannot be undone.');
      if (!confirmed) return;
      this.deleteMatchNow();
    },

    async deleteMatchNow() {
      this.isDeleting = true;
      try {
        await deleteMatch(this.match.ID, this.activeGroupId);
        this.goBack();
      } catch (error) {
        console.error('Error deleting match:', error);
        this.showMessage('Error deleting match. Please try again.', 'error');
      } finally {
        this.isDeleting = false;
      }
    },

    showMessage(text, type = 'success') {
      this.message = text;
      this.messageType = type;
      this.messageKey++; // Trigger re-render for animations

      // Auto-dismiss after 4 seconds
      if (this.messageTimeout) {
        clearTimeout(this.messageTimeout);
      }

      this.messageTimeout = setTimeout(() => {
        this.dismissMessage();
      }, 3000);
    },

    dismissMessage() {
      const messageEl = document.querySelector('.message');
      if (messageEl) {
        messageEl.classList.add('toast-exit');
        setTimeout(() => {
          this.message = '';
          if (this.messageTimeout) {
            clearTimeout(this.messageTimeout);
          }
        }, 300);
      }
    },

    formatDate(dateString) {
      return formatCalendarDayLong(dateString);
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

    getTotalGoals() {
      if (!this.match || !this.match.Teams) return 0;
      return this.match.Teams.reduce((total, team) => total + (team.Score || 0), 0);
    },

    getTotalPlayers() {
      if (!this.match || !this.match.Teams) return 0;
      return this.match.Teams.reduce((total, team) => total + (team.Players ? team.Players.length : 0), 0);
    },

    // Predates scheduled matches: "any goal recorded" was a reasonable proxy
    // for "has this been played" back when a match was always created and
    // scored immediately after the fact. For a scheduled match that's no
    // longer true — an admin can compose the roster and enter goals (via
    // "Fill teams from sign-ups" plus manual editing) *before* kick-off, and
    // the goal count alone would then call a match that hasn't happened yet
    // "Completed". Kick-off is checked first for exactly that case: before
    // it, the match is unambiguously upcoming regardless of what's already
    // been entered. An unscheduled match has no ScheduledAt to check, so it
    // falls straight through to the original goal-based heuristic — same
    // behaviour as before this feature existed.
    getMatchStatus() {
      if (this.isScheduled && this.nowMs < Date.parse(this.match.ScheduledAt)) {
        return 'upcoming';
      }
      const totalGoals = this.getTotalGoals();
      if (totalGoals === 0) return 'upcoming';
      return 'completed';
    },

    getMatchStatusText() {
      const status = this.getMatchStatus();
      switch (status) {
        case 'upcoming': return 'Upcoming';
        case 'completed': return 'Completed';
        default: return 'Unknown';
      }
    }
  }
};
</script>

<style scoped>
/* Component-specific styles that couldn't be moved to global */

.match-detail-container {
  background-color: var(--bg-secondary);
}

/* Match Content */
.match-content {
  padding: 2rem 0;
}

/* Score Overview */
.score-overview {
  margin-bottom: 2rem;
}

.match-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.75rem 1rem;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.match-title-left,
.match-title-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.match-back-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  background: none;
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.match-back-btn:hover {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
}

.match-back-btn svg {
  width: 18px;
  height: 18px;
}

.match-date {
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.match-status-badge {
  padding: 0.3rem 0.65rem;
  border-radius: 20px;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.match-status-badge.upcoming {
  background-color: #fef3c7;
  color: #92400e;
}

.match-status-badge.completed {
  background-color: #d1fae5;
  color: #065f46;
}

.teams-score {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 1.5rem;
}

/* One row per team — name/colour and score side by side — at every
   breakpoint, rather than a large stacked block on desktop and a
   separately-tuned compact row on mobile. */
.team-score {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.85rem 1.1rem;
  background-color: var(--bg-secondary);
  border-radius: var(--border-radius);
  border: 1px solid var(--border-color);
  transition: all var(--transition-fast);
}

.team-score:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.team-info {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 0.6rem;
  text-align: left;
  min-width: 0;
}

.team-info .team-color {
  width: 18px;
  height: 18px;
  border-width: 2px;
  flex-shrink: 0;
}

.team-name {
  font-size: 1.05rem;
  font-weight: 700;
  color: var(--text-primary);
  text-transform: capitalize;
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.score {
  font-size: 1.75rem;
  font-weight: 900;
  color: var(--primary-color);
  text-shadow: 0 2px 4px rgba(16, 185, 129, 0.2);
  line-height: 1;
  flex-shrink: 0;
}

.vs-divider {
  display: flex;
  align-items: center;
  justify-content: center;
}

.vs-circle {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 800;
  font-size: 0.7rem;
  flex-shrink: 0;
  box-shadow: var(--shadow-lg);
}

/* Sign-up panel (scheduled matches only) */
.signup-panel {
  margin-bottom: 2rem;
}

.signup-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.75rem 1rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.signup-kickoff {
  display: flex;
  align-items: center;
  gap: 0.65rem;
}

.signup-kickoff svg {
  width: 22px;
  height: 22px;
  color: var(--primary-color);
  flex-shrink: 0;
}

.signup-kickoff-text {
  display: flex;
  flex-direction: column;
}

.signup-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
}

.signup-kickoff-value {
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-primary);
}

.signup-status-group {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.6rem;
}

.signup-count-inline {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

/* Same pill shape as .match-status-badge above, so the two badges on this page
   read as the same kind of thing. */
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

.signup-state-detail {
  margin: 1rem 0 0;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.signup-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-top: 1rem;
}

/* Marked out from Close/Reopen next to it: this is the one button in the panel
   that changes the page's own unsaved state rather than the server's. */
.fill-teams-btn {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.fill-teams-hint {
  margin: 0.75rem 0 0;
  color: var(--text-secondary);
  font-size: 0.8rem;
  line-height: 1.5;
}

/* The "nothing is saved" half of the sentence has to survive being skim-read. */
.fill-teams-hint strong {
  color: var(--text-primary);
}

.signup-loading {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 1.5rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.signup-lists {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
  margin-top: 1.5rem;
}

.signup-list-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0 0 0.75rem;
  font-size: 0.9rem;
}

.signup-entries {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  /* A 30-strong waiting list must not push the team management section off
     the bottom of the page. */
  max-height: 18rem;
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

/* The waiting list is deliberately muted: it is real information, but the
   confirmed roster is what a player scans for first. */
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

/* Man of the Match voting — deliberately its own small panel rather than a
   tab inside .team-management: it applies to every composed match, scheduled
   or not, so it can't live behind the sign-up-only gating above it. */
.motm-panel {
  margin-bottom: 2rem;
}

.motm-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem 1rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.motm-title {
  margin: 0;
  font-size: 1.1rem;
  color: var(--text-primary);
}

.motm-my-vote {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.motm-vote-form {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
  margin-top: 1rem;
}

.motm-candidate-select {
  flex: 1;
  min-width: 12rem;
  padding: 0.5rem 0.75rem;
  border-radius: var(--border-radius);
  border: 1px solid var(--border-color);
  background-color: var(--bg-secondary);
  color: var(--text-primary);
  font-size: 0.875rem;
}

.motm-loading {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 1.5rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.motm-tally {
  list-style: none;
  margin: 1.5rem 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  /* Same cap as .signup-entries, for the same reason: a big roster must not
     push the team management section off the bottom of the page. */
  max-height: 14rem;
  overflow-y: auto;
}

.motm-tally-entry {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.65rem;
  padding: 0.5rem 0.65rem;
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  font-size: 0.875rem;
}

.motm-tally-name {
  color: var(--text-primary);
  font-weight: 500;
}

.motm-tally-votes {
  color: var(--text-secondary);
  font-variant-numeric: tabular-nums;
}

.motm-empty {
  margin: 1.5rem 0 0;
  color: var(--text-secondary);
  font-size: 0.875rem;
  font-style: italic;
}

@media (max-width: 768px) {
  .motm-vote-form {
    flex-direction: column;
    align-items: stretch;
  }

  .motm-candidate-select {
    width: 100%;
  }
}

/* Team Management */
.team-management {
  margin-bottom: 2rem;
}

.management-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  margin-bottom: 2rem;
  padding: 1rem;
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  border: 1px solid var(--border-color);
}

.tabs-buttons {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.no-roster-hint {
  margin: 0;
  font-size: 0.875rem;
  color: var(--text-secondary);
  font-style: italic;
}

/* "Add player to this team" — sits right next to the team switcher it acts
   on, rather than among the global Save/Delete actions. Dashed border
   echoes the "+" add-match card used in the matches list, for a consistent
   "+" affordance across the app. */
.add-player-icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2.5rem;
  height: 2.5rem;
  flex-shrink: 0;
  border: 2px dashed var(--border-color);
  border-radius: var(--border-radius);
  background: none;
  color: var(--primary-color);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.add-player-icon-btn:hover {
  border-color: var(--primary-color);
  background-color: var(--bg-tertiary);
}

.add-player-icon-btn svg {
  width: 20px;
  height: 20px;
}

.tab-button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem 1.5rem;
  background-color: var(--bg-primary);
  border: 2px solid var(--border-color);
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-weight: 500;
  color: var(--text-secondary);
}

.tab-button:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.tab-button.active {
  background-color: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.action-buttons {
  display: flex;
  gap: 1rem;
  align-items: center;
}

/* Pushed to the far end of the action bar and separated with extra margin so
   a click can't land here by mistake while reaching for Save Changes. */
.delete-match-btn {
  margin-left: 1.5rem;
}

/* Players list — one compact row per player instead of a taller card with
   a header row and a separate goal row, so the list needs far less
   scrolling to see everyone on a team. Bounded to whatever's left below
   the score card and the Save/Delete/tabs header above it, with its own
   scrollbar — a roster with many players then scrolls inside this list
   instead of growing the whole page and pushing Save Changes/Delete
   Match out of easy reach (same pattern as MatchesPanel.vue's own
   two-column team list). The 560px offset estimates that chrome's
   height; max() keeps a handful of rows visible either way. */
.players-grid {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-height: max(9rem, calc(100vh - 560px));
  overflow-y: auto;
}

.player-card {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  padding: 0.6rem 0.85rem;
  border: 1px solid var(--border-color);
  transition: all var(--transition-fast);
}

.player-card:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.player-info {
  flex: 1 1 auto;
  min-width: 0;
}

.player-info .player-name {
  /* It's an <h4>, which carries its own default vertical margin (unlike
     the <span> used for a player name elsewhere in this file) — left in
     place, that margin box (not the visible text) is what .player-card's
     align-items:center actually centers, throwing off the row's height
     and vertical alignment for no visual benefit. */
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.btn-danger-icon {
  background-color: var(--danger-color);
  color: white;
  border: none;
  border-radius: 50%;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-danger-icon:hover {
  background-color: var(--danger-hover);
}

.btn-danger-icon svg {
  width: 16px;
  height: 16px;
}

/* Goal Management */
.goal-management {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.goal-btn {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 2px solid var(--primary-color);
  background-color: var(--bg-primary);
  color: var(--primary-color);
  cursor: pointer;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.goal-btn:hover:not(:disabled) {
  background-color: var(--primary-color);
  color: white;
}

.goal-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.goal-btn svg {
  width: 14px;
  height: 14px;
}

.goal-count {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--primary-color);
  min-width: 24px;
  text-align: center;
}

/* Compose-choice modal — .modal-overlay/.modal-container/.modal-header/
   .modal-close all come from global-styles.css; only the two option cards
   are specific to this modal. */
.compose-choice-body {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.compose-choice-option {
  display: flex;
  align-items: flex-start;
  gap: 0.85rem;
  padding: 1rem;
  background: none;
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
  cursor: pointer;
  text-align: left;
  transition: all var(--transition-fast);
  width: 100%;
}

.compose-choice-option:hover {
  background-color: var(--bg-tertiary);
  border-color: var(--primary-color);
}

.compose-choice-icon {
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  color: var(--primary-color);
  margin-top: 0.15rem;
}

.compose-choice-text {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.compose-choice-text strong {
  color: var(--text-primary);
  font-size: 0.95rem;
}

.compose-choice-text span {
  color: var(--text-secondary);
  font-size: 0.825rem;
  line-height: 1.5;
}

/* Enhanced Multi-Player Modal */
.enhanced-multi-player-modal {
  background-color: var(--bg-primary);
  border-radius: var(--border-radius-lg);
  box-shadow: var(--shadow-xl);
  max-width: 900px;
  width: 95%;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-color);
}

.enhanced-modal-header {
  background: linear-gradient(135deg, var(--primary-color), var(--secondary-color)) !important;
  color: white !important;
  border-radius: var(--border-radius-lg) var(--border-radius-lg) 0 0 !important;
  border-bottom: none !important;
}

.enhanced-modal-header h3 {
  color: white !important;
}

.enhanced-modal-header .modal-close {
  color: rgba(255, 255, 255, 0.8) !important;
}

.enhanced-modal-header .modal-close:hover {
  background-color: rgba(255, 255, 255, 0.1) !important;
  color: white !important;
}

.enhanced-modal-body {
  flex: 1;
  padding: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* Search Section */
.search-section {
  flex-shrink: 0;
  padding: 1.5rem;
  background-color: var(--bg-secondary);
  border-bottom: 2px solid var(--border-color);
}

.search-section .form-group {
  margin-bottom: 0;
}

.search-section label {
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 0.75rem;
}

.input-wrapper {
  position: relative;
}

.search-loading {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
}

/* Create Player Section */
.create-player-section {
  margin-top: 1rem;
  padding: 1rem;
  background-color: var(--bg-tertiary);
  border-radius: var(--border-radius);
  border: 1px solid var(--border-color);
}

.create-player-prompt {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.info-icon {
  width: 16px;
  height: 16px;
  color: var(--secondary-color);
}

.create-player-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background-color: var(--accent-color);
  color: white;
  border: none;
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-weight: 500;
  font-size: 0.875rem;
}

.create-player-btn:hover:not(:disabled) {
  background-color: #d97706;
  transform: translateY(-1px);
}

.create-player-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.create-player-btn svg {
  width: 16px;
  height: 16px;
}

/* Two-column layout */
.players-columns {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  min-height: 0;
}

.column {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.selected-column {
  border-right: 1px solid var(--border-color);
}

.column-header {
  flex-shrink: 0;
  padding: 1rem 1.5rem;
  background: linear-gradient(135deg, var(--bg-tertiary), var(--bg-secondary));
  border-bottom: 1px solid var(--border-color);
}

.column-header h4 {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.header-icon {
  width: 18px;
  height: 18px;
  color: var(--primary-color);
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

.column-content {
  flex: 1;
  padding: 1rem;
  overflow-y: auto;
  min-height: 0;
}

/* Selected Players List */
.selected-players-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.selected-player-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem;
  background-color: var(--bg-secondary);
  border-radius: var(--border-radius);
  border: 1px solid var(--border-color);
  transition: all var(--transition-fast);
}

.selected-player-item:hover {
  background-color: var(--bg-tertiary);
  transform: translateX(-2px);
}

.remove-selected-btn {
  background: none;
  border: none;
  color: var(--danger-color);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: 50%;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
}

.remove-selected-btn:hover {
  background-color: rgba(239, 68, 68, 0.1);
}

.remove-selected-btn svg {
  width: 16px;
  height: 16px;
}

/* Available Players List */
.available-players-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.available-player-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem;
  background: none;
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
  cursor: pointer;
  transition: all var(--transition-fast);
  text-align: left;
  width: 100%;
}

.available-player-item:hover:not(:disabled) {
  background-color: var(--bg-tertiary);
  border-color: var(--primary-color);
  transform: translateX(2px);
}

.available-player-item:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.player-status {
  display: flex;
  align-items: center;
}

.status-selected {
  color: var(--primary-color);
}

/* Same pill language as .count-badge/.signup-position elsewhere in this
   file — confirmed reads as the "normal" state (primary colour), waiting
   stays deliberately muted, same reasoning as .signup-list-waiting. */
.registration-badge {
  display: inline-block;
  margin-left: 0.5rem;
  padding: 0.05rem 0.45rem;
  border-radius: 12px;
  font-size: 0.7rem;
  font-weight: 600;
  background-color: var(--primary-color);
  color: white;
  vertical-align: middle;
}

.registration-badge.waiting {
  background-color: var(--bg-tertiary);
  color: var(--text-secondary);
}

.status-in-match {
  color: var(--danger-color);
}

.status-available {
  color: var(--text-light);
}

.player-status svg {
  width: 18px;
  height: 18px;
}

/* Enhanced Footer */
.enhanced-footer {
  flex-shrink: 0;
  background-color: var(--bg-secondary);
  border-top: 2px solid var(--border-color);
  padding: 1.5rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
}

.footer-info {
  flex: 1;
}

.selection-summary {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--text-secondary);
}

.summary-icon {
  width: 16px;
  height: 16px;
  color: var(--primary-color);
}

.selection-count {
  font-weight: 500;
}

.selection-count.empty {
  color: var(--text-light);
}

.footer-buttons {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.enhanced-confirm {
  min-width: 120px;
}

/* Error State */
.error-state {
  padding: 4rem 0;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 50vh;
}

/* Responsive Design */
@media (max-width: 768px) {
  .match-content {
    padding: 1rem 0;
  }

  /* Score overview: card-base/card-large's own padding is sized for
     desktop — override it directly here rather than fighting it. */
  .score-overview {
    padding: 1rem;
    margin-bottom: 1rem;
  }

  /* No more "Match" heading taking up half the row (see .match-title-left
     in the template) — the back button, date and status badge now fit on
     one row even on a narrow screen, so the wrap the base rule allows for
     is no longer needed. */
  .match-title {
    flex-wrap: nowrap;
    margin-bottom: 1rem;
    padding-bottom: 0.75rem;
  }

  /* The full "Sunday, August 23, 2026" format (see formatDate()) is long
     next to the back button and status badge sharing this row on a narrow
     screen — shrink it rather than truncate or reflow the row. */
  .match-date {
    font-size: 0.75rem;
  }

  /* Both teams already sit on one row at every breakpoint (see .team-score
     above) — mobile just gets a bit tighter to fit the narrower width. */
  .teams-score {
    gap: 0.5rem;
  }

  .team-score {
    padding: 0.6rem 0.75rem;
    gap: 0.5rem;
  }

  .team-name {
    font-size: 0.9rem;
  }

  .team-info .team-color {
    width: 20px;
    height: 20px;
  }

  .score {
    font-size: 1.65rem;
  }

  .vs-circle {
    width: 32px;
    height: 32px;
    font-size: 0.7rem;
  }

  .team-management {
    padding: 1rem;
    margin-bottom: 1rem;
  }

  .management-header {
    flex-direction: column;
    align-items: stretch;
    gap: 0.75rem;
    margin-bottom: 1rem;
    padding: 0.75rem;
  }

  .tabs-buttons {
    justify-content: center;
    gap: 0.5rem;
  }

  .tab-button {
    padding: 0.6rem 1rem;
    font-size: 0.9rem;
  }

  /* Side by side on one row instead of stacked full-width — each button
     shares the row equally. */
  .action-buttons {
    flex-wrap: nowrap;
    gap: 0.5rem;
  }

  .action-buttons .btn-base {
    flex: 1;
    justify-content: center;
  }

  /* They're already visually separated by sharing the row 50/50 with a
     gap, so the extra desktop-only margin meant to prevent a misclick
     would just throw the 50/50 split off here. */
  .delete-match-btn {
    margin-left: 0;
  }

  /* Roster row: player name rendered as an <h4> with no font-size of its
     own (see .player-name in global-styles.css), so it falls back to the
     browser's default ~1rem heading size. 0.85rem (an earlier pass) read
     as too small next to the rest of the row — 1rem is the actual target,
     just without the <h4>'s own oversized default margin (stripped in the
     base rule above, not this override). */
  .player-info .player-name {
    font-size: 1rem;
  }

  .enhanced-multi-player-modal {
    max-width: 95%;
    max-height: 95vh;
  }

  .search-section {
    padding: 1rem;
  }

  .column-header {
    padding: 0.75rem 1rem;
  }

  .column-content {
    padding: 0.75rem;
  }

  .players-columns {
    grid-template-columns: 1fr;
  }

  .selected-column {
    border-right: none;
    border-bottom: 1px solid var(--border-color);
  }

  .enhanced-footer {
    flex-direction: column;
    align-items: stretch;
    gap: 0.75rem;
    padding: 1rem;
  }

  .footer-buttons {
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .match-content {
    padding: 0.75rem 0;
  }

  .tab-button {
    padding: 0.6rem 0.85rem;
    font-size: 0.85rem;
  }

  .team-color-small {
    width: 12px;
    height: 12px;
  }
}
</style>