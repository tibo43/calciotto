<template>
  <main class="page" :class="{ 'match-upcoming': upcoming, 'has-sticky': betOpen }">

    <section v-if="loading" class="state">
      <div class="skel lg"></div>
      <div class="skel md"></div>
    </section>
    <section v-else-if="failed" class="state">
      <h1>{{ $t('match.loadError') }}</h1>
      <button class="btn" type="button" @click="load">{{ $t('common.tryAgain') }}</button>
    </section>

    <template v-else-if="upcoming">
      <section class="hero">
        <p class="label">{{ $t('home.nextMatch') }}</p>
        <div class="hero-date-row">
          <h1 class="display hero-date">{{ heroDateLabel(match.Date) }}</h1>
          <ManageMatchLink v-if="isAdmin" :id="match.ID" />
        </div>
        <p v-if="kickoffLabel(match.ScheduledAt)" class="display hero-time">{{ kickoffLabel(match.ScheduledAt) }}</p>
        <!-- Once the teams are set, the player's side replaces the places
             count, as on the home page. -->
        <div v-if="teamsSet" class="hero-team">
          <template v-if="myTeam">
            <!-- Same line as on the home: mono label, the team in the accent. -->
            <p class="label hero-team-line"><i18n-t keypath="home.youPlayFor" scope="global"><template #team><span class="hero-team-mine">{{ myTeam }}</span></template></i18n-t></p>
          </template>
          <p v-else class="label">{{ $t('home.teamsSet') }}</p>
        </div>
        <template v-else>
          <div class="hero-spots">
            <div class="num big">{{ places }}</div>
            <!-- Same three lines as on the home page. -->
            <div class="cap">
              <span class="cap-line">{{ $t('home.placesLeft') }}</span>
              <span class="cap-line">{{ $t('home.signedUpCount', { confirmed: confirmed.length, cap }) }}</span>
              <span v-if="full" class="cap-line">{{ $t('home.waitingCount', { n: waiting.length }) }}</span>
            </div>
          </div>
          <div class="bar" aria-hidden="true"><i :style="{ '--pct': barPct }"></i></div>
        </template>
        <p v-if="mine && !teamsSet" class="reg-you">{{ $t('home.youAreSignedUp', { status: mine.IsWaiting ? $t('home.statusWaiting') : $t('home.statusConfirmed') }) }}</p>
        <p v-else-if="full && open" class="reg-you"><i18n-t keypath="home.fullYoudBe" scope="global"><template #position><strong>{{ $t('home.position', { n: waitingPosition }) }}</strong></template></i18n-t></p>
        <div class="actions">
          <button v-if="!mine && open" class="btn" type="button" :disabled="busy" @click="signUp">{{ full ? $t('home.joinWaitingList') : $t('home.signUp') }}</button>
          <span v-else-if="mine && open" class="withdraw-wrap withdraw-slot">
            <button class="btn btn-ghost" type="button" :disabled="busy" @click="confirming = true">{{ $t('home.withdraw') }}</button>
            <!-- Wide screens: anchored under the button, as on the home page. -->
            <div class="confirm-pop" role="dialog" aria-labelledby="match-withdraw-pop-title">
              <h3 id="match-withdraw-pop-title">{{ $t('home.withdrawTitle') }}</h3>
              <p class="ref">{{ $t('home.withdrawText') }}</p>
              <div class="confirm-actions">
                <button class="btn btn-danger" type="button" :disabled="busy" @click="withdraw">{{ $t('home.withdraw') }}</button>
                <button class="btn btn-ghost" type="button" @click="confirming = false">{{ $t('common.cancel') }}</button>
              </div>
            </div>
          </span>
        </div>
      </section>
      <!-- The teams, once set, are what the sign-up list was for. -->
      <div v-if="teamsSet" class="lineups lineups-upcoming">
        <section v-for="team in teams" :key="team.ID" class="lineup">
          <h3>{{ team.Name }}</h3>
          <ul class="signup-ro">
            <li v-for="player in team.Players" :key="player.ID" :class="{ 'is-me': player.ID === playerId }">
              <!-- Group admins only: the player's preferred positions right
                   after the name, which gives way first (…). -->
              <span v-if="tagOf(player)" class="nm-line"><span class="nm">{{ player.Name }}</span><span class="pos-tag">{{ tagOf(player) }}</span></span>
              <span v-else class="nm">{{ player.Name }}</span>
              <span v-if="player.ID === playerId" class="pos">{{ $t('match.you') }}</span>
            </li>
          </ul>
        </section>
      </div>
      <section v-else class="sec">
        <div class="sec-head"><h2>{{ $t('match.confirmed') }}</h2><span class="label">{{ $t('match.confirmedCount', { confirmed: confirmed.length, cap }) }}</span></div>
        <ul class="signup-ro">
          <li v-for="entry in confirmed" :key="entry.PlayerID">
            <span class="nm">{{ entry.Name }}</span>
            <span class="pos">{{ $t('home.position', { n: entry.Position }) }}</span>
          </li>
        </ul>
      </section>
      <!-- The prediction, next to the teams it's about: pinned above the tab
           bar on a phone, in the flow on a desktop. -->
      <div v-if="betOpen" class="sticky-cta bet-bar">
        <p class="cta-msg">{{ myPick ? $t('match.yourPredictionPick', { pick: myPick }) : $t('match.yourPrediction') }}</p>
        <div class="picks">
          <button v-for="option in betOptions" :key="option.key" class="pick" type="button"
            :class="{ 'is-on': option.on }" :disabled="betBusy" @click="predict(option.teamId)">
            <span class="who">{{ option.label }}</span>
            <span v-if="option.odds" class="odd">{{ option.odds }}</span>
          </button>
        </div>
      </div>
      <section v-if="waiting.length" class="sec">
        <div class="sec-head"><h2>{{ $t('match.waitingList') }}</h2><span class="label">{{ waiting.length }}</span></div>
        <ul class="signup-ro">
          <li v-for="entry in waiting" :key="entry.PlayerID">
            <span class="nm">{{ entry.Name }}</span>
          </li>
        </ul>
      </section>
    </template>

    <template v-else>
      <section class="board">
        <div class="board-date-row">
          <p class="label board-date">{{ heroDateLabel(match.Date) }}</p>
          <ManageMatchLink v-if="isAdmin" :id="match.ID" />
        </div>
        <div v-fit-two-lines class="board-teams">
          <div class="board-team"><span class="name">{{ teamName(0) }}</span></div>
          <div class="board-score">{{ scoreText }}</div>
          <div class="board-team"><span class="name">{{ teamName(1) }}</span></div>
        </div>
      </section>
      <p class="motm-line">{{ voteCopy }}</p>
      <div class="lineups">
        <section v-for="team in teams" :key="team.ID" class="lineup">
          <h3>{{ team.Name }}</h3>
          <template v-for="player in team.Players" :key="player.ID">
          <button v-if="canVote(player)" class="player" :class="{ 'is-winner': isWinner(player) }" type="button" :aria-label="voteAria(player)" @click="castVote(player)">
            <span v-if="tagOf(player)" class="nm-line"><span class="nm">{{ player.Name }}</span><span class="pos-tag">{{ tagOf(player) }}</span></span>
            <span v-else class="nm">{{ player.Name }}</span>
            <span class="player-meta">
              <span v-if="goalLabel(player)" class="goals">{{ goalLabel(player) }}</span>
              <span class="votes"><span class="star" aria-hidden="true">{{ starFor(player) }}</span>{{ votesFor(player) }}</span>
            </span>
          </button>
          <div v-else class="player" :class="{ 'is-self': player.ID === playerId, 'is-winner': isWinner(player) }">
            <span v-if="tagOf(player)" class="nm-line"><span class="nm">{{ player.Name }}</span><span class="pos-tag">{{ tagOf(player) }}</span></span>
            <span v-else class="nm">{{ player.Name }}</span>
            <span class="player-meta">
              <span v-if="goalLabel(player)" class="goals">{{ goalLabel(player) }}</span>
              <span class="votes"><span class="star" aria-hidden="true">{{ starFor(player) }}</span>{{ votesFor(player) }}</span>
            </span>
          </div>
          </template>
        </section>
      </div>
    </template>

    <!-- Phones: the same bottom sheet as the home page, shown by the
         shell's is-confirming class (the pop-up above is for wide screens).
         Only while the match is ahead: once over, nothing is about sign-ups. -->
    <div v-if="upcoming" class="confirm-scrim" @click="confirming = false"></div>
    <div v-if="upcoming" class="confirm-sheet" role="dialog" aria-labelledby="match-withdraw-title">
      <h3 id="match-withdraw-title">{{ $t('home.withdrawTitle') }}</h3>
      <p class="ref">{{ $t('home.withdrawText') }}</p>
      <div class="confirm-actions">
        <button class="btn btn-danger" type="button" :disabled="busy" @click="withdraw">{{ $t('home.withdraw') }}</button>
        <button class="btn btn-ghost" type="button" @click="confirming = false">{{ $t('common.cancel') }}</button>
      </div>
    </div>

    <AppToast ref="toast" />
  </main>
</template>

<script>
import { getMatchRegistrations, registerForMatch, unregisterFromMatch, getMatchVotes, voteForMotm, removeMotmVote, getSeasons, getToken, getMatchPredictions, predictForMatch, getMatchPositions } from '@/services/api';
import { positionsMap, positionsTag } from '@/services/v2Positions';
import { predictionsAreOpen } from '@/services/matchPrediction';
import { setActiveGroupId } from '@/services/activeGroup';
import { findMatchInMyGroups, actionErrorMessage } from '@/services/v2Match';
import AppToast from '@/components/AppToast.vue';
import ManageMatchLink from '@/components/v2/ManageMatchLink.vue';
import { deriveRegistrationState, registrationsAreOpen, isPlayerOnRoster, teamsAreComposed } from '@/services/matchRegistration';
import { heroDateLabel, kickoffLabel, placesLeft, isMatchOver, matchVoteCopy, motmWinners, goalLine, myTeam, nextWaitingPosition, voteErrorMessage } from '@/services/v2Home';
import { formatDecimal } from '@/services/intl';
import { isMotmVotingOpen } from '@/services/motmVoting';
import fitTwoLines from '@/directives/fitTwoLines';

function currentPlayerIdFromToken() {
  const token = getToken();
  if (!token) return '';
  try {
    const payload = token.split('.')[1];
    const decoded = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')));
    return decoded.player_id || '';
  } catch {
    return '';
  }
}

export default {
  name: 'MatchView',
  directives: { fitTwoLines },
  components: { AppToast, ManageMatchLink },
  inject: {
    v2Context: { default: () => ({ label: '' }) },
  },
  props: {
    id: { type: String, required: true },
  },
  data() {
    return {
      loading: true,
      failed: false,
      match: null,
      registrations: [],
      votes: null,
      playerId: '',
      nowMs: Date.now(),
      isAdmin: false,
      positionsById: {},
      busy: false,
      confirming: false,
      prediction: null,
      betBusy: false,
    };
  },
  computed: {
    upcoming() {
      return this.match && !isMatchOver(this.match, this.nowMs);
    },
    teams() {
      return (this.match && this.match.Teams) || [];
    },
    confirmed() {
      return this.registrations.filter((entry) => !entry.IsWaiting);
    },
    waiting() {
      return this.registrations.filter((entry) => entry.IsWaiting);
    },
    cap() {
      return Number(this.match && this.match.MaxPlayers) || 0;
    },
    places() {
      return placesLeft(this.match, this.confirmed.length);
    },
    barPct() {
      if (!this.cap) return '0%';
      return `${Math.min(100, Math.round((this.confirmed.length / this.cap) * 100))}%`;
    },
    mine() {
      return this.registrations.find((entry) => entry.PlayerID === this.playerId) || null;
    },
    teamsSet() {
      return Boolean(this.match) && teamsAreComposed(this.match);
    },
    myTeam() {
      return this.teamsSet ? myTeam(this.match, this.playerId) : null;
    },
    betOpen() {
      return Boolean(this.match) && this.upcoming && Boolean(this.prediction) && predictionsAreOpen(this.match, this.nowMs);
    },
    betOptions() {
      const odds = (this.prediction && this.prediction.Odds) || {};
      const mine = this.prediction && this.prediction.MyPrediction;
      const picked = mine ? (mine.PredictedTeamID || null) : undefined;
      const fmt = (value) => (Number.isFinite(Number(value)) && value != null ? formatDecimal(Number(value)) : '');
      const [a, b] = this.teams;
      return [
        { key: 'a', teamId: a.ID, label: a.Name, odds: fmt(odds.TeamAOdds) },
        { key: 'draw', teamId: null, label: this.$t('match.draw'), odds: fmt(odds.DrawOdds) },
        { key: 'b', teamId: b.ID, label: b.Name, odds: fmt(odds.TeamBOdds) },
      ].map((option) => ({ ...option, on: picked !== undefined && picked === option.teamId }));
    },
    myPick() {
      const chosen = this.betOptions.find((option) => option.on);
      return chosen ? this.$t('match.pick', { label: chosen.label, odds: chosen.odds }) : '';
    },
    full() {
      return Boolean(this.cap) && this.places === 0;
    },
    waitingPosition() {
      return nextWaitingPosition(this.registrations);
    },
    open() {
      return registrationsAreOpen(deriveRegistrationState(this.match, this.nowMs));
    },
    scoreText() {
      const a = this.teams[0] ? this.teams[0].Score : 0;
      const b = this.teams[1] ? this.teams[1].Score : 0;
      return `${a}–${b}`;
    },
    voteOpen() {
      return this.match && !this.upcoming && isMotmVotingOpen(this.match, this.nowMs);
    },
    winners() {
      return motmWinners(this.votes);
    },
    voteCopy() {
      return matchVoteCopy(this.match, this.votes, this.nowMs);
    },
  },
  watch: {
    id() { this.load(); },
    confirming(value) {
      this.v2Context.confirming = value;
    },
  },
  // The shell's confirmation overlay is driven by this flag; leaving the
  // page with it open must not leave the overlay on the next one.
  beforeUnmount() {
    this.v2Context.confirming = false;
  },
  created() {
    this.playerId = currentPlayerIdFromToken();
    this.load();
  },
  methods: {
    // "DEF · MID", or '' for a player with no position.
    tagOf(player) {
      if (!player) return '';
      return positionsTag(this.positionsById[player.ID]);
    },
    heroDateLabel,
    kickoffLabel,
    teamName(index) {
      const team = this.teams[index];
      return (team && team.Name) || '—';
    },
    goalLabel(player) {
      return goalLine(player);
    },
    votesFor(player) {
      const row = this.votes && (this.votes.Tally || []).find((entry) => entry.PlayerID === player.ID);
      return row ? row.Votes : 0;
    },
    starFor(player) {
      return this.votes && this.votes.MyVoteFor === player.ID ? '★' : '☆';
    },
    // The accent goes to the winner once the vote is closed. While it is
    // open, the one ahead is no winner yet: every count stays muted.
    isWinner(player) {
      if (this.voteOpen) return false;
      return this.winners.some((row) => row.PlayerID === player.ID);
    },
    canVote(player) {
      return this.voteOpen && isPlayerOnRoster(this.match, this.playerId) && player.ID !== this.playerId;
    },
    isMyVote(player) {
      return Boolean(this.votes && this.votes.MyVoteFor === player.ID);
    },
    voteAria(player) {
      const count = this.votesFor(player);
      return this.isMyVote(player) ? this.$t('match.votesAriaMine', count) : this.$t('match.votesAria', count);
    },
    // Tapping the player you voted for takes the vote back, as on today's
    // Matches tab; any other player casts or moves it.
    async castVote(player) {
      if (!this.canVote(player) || this.busy) return;
      this.busy = true;
      const removing = this.isMyVote(player);
      try {
        if (removing) {
          await removeMotmVote(this.match.ID);
        } else {
          await voteForMotm(this.match.ID, player.ID);
        }
        this.$refs.toast.show(removing
          ? this.$t('match.vote.removed')
          : this.$t('match.vote.saved'), 'success');
      } catch (error) {
        console.error('Error updating MVP vote:', error);
        this.$refs.toast.show(voteErrorMessage(error), 'error');
      } finally {
        this.votes = await getMatchVotes(this.match.ID).catch(() => this.votes);
        this.busy = false;
      }
    },
    async publishChrome(groups) {
      if (!this.v2Context) return;
      const seasons = await getSeasons(this.v2Context.groupId || (groups[0] && groups[0].id)).catch(() => []);
      this.v2Context.groups = groups;
      this.v2Context.seasons = Array.isArray(seasons) ? seasons : [];
      if (!this.v2Context.season && this.v2Context.seasons.length) {
        this.v2Context.season = this.v2Context.seasons[this.v2Context.seasons.length - 1];
      }
      this.v2Context.hide = false;
      this.v2Context.onGroup = (id) => {
        setActiveGroupId(id);
        window.location.href = '/';
      };
      this.v2Context.onSeason = (season) => {
        sessionStorage.setItem('calciotto-v2-season', season);
        window.location.href = '/';
      };
    },
    async load() {
      this.loading = true;
      this.failed = false;
      this.nowMs = Date.now();
      try {
        // Looked up in the group it belongs to, not the caller's first one,
        // so a player in several groups can open any of their matches.
        const { group, details, groups } = await findMatchInMyGroups(this.id);
        this.match = details;
        const [regs, votes] = await Promise.all([
          getMatchRegistrations(this.id).catch(() => []),
          getMatchVotes(this.id).catch(() => null),
        ]);
        this.registrations = Array.isArray(regs) ? regs : [];
        this.votes = votes;
        this.prediction = predictionsAreOpen(details, this.nowMs)
          ? await getMatchPredictions(this.id).catch(() => null)
          : null;
        this.isAdmin = group.role === 'admin';
        // Every member of the group sees the preferred positions; a failed
        // request just shows no tag.
        this.positionsById = positionsMap(await getMatchPositions(this.id).catch(() => []));
        this.v2Context.groupId = group.id;
        await this.publishChrome(groups).catch(() => {});
      } catch (error) {
        console.error(error);
        this.failed = true;
      } finally {
        this.loading = false;
      }
    },
    async predict(teamId) {
      if (this.betBusy) return;
      this.betBusy = true;
      try {
        this.prediction = await predictForMatch(this.id, teamId);
      } catch (error) {
        console.error(error);
        this.$refs.toast.show(actionErrorMessage(error, this.$t('match.predictionError')), 'error');
        this.prediction = await getMatchPredictions(this.id).catch(() => this.prediction);
      } finally {
        this.betBusy = false;
      }
    },
    // The registrations are re-read either way: a refusal usually means
    // this page's view of the list was stale.
    async signUp() {
      if (this.busy) return;
      this.busy = true;
      try {
        const entry = await registerForMatch(this.id);
        if (entry && entry.IsWaiting) {
          this.$refs.toast.show(this.$t('home.toast.waiting', { position: entry.Position }), 'success');
        } else {
          const message = entry && entry.Position
            ? this.$t('home.toast.inPosition', { position: entry.Position })
            : this.$t('home.toast.in');
          this.$refs.toast.show(message, 'success');
        }
      } catch (error) {
        console.error('Error signing up for match:', error);
        this.$refs.toast.show(actionErrorMessage(error, this.$t('home.toast.signUpError')), 'error');
      } finally {
        await this.reloadRegistrations();
        this.busy = false;
      }
    },
    async withdraw() {
      if (this.busy) return;
      this.busy = true;
      try {
        await unregisterFromMatch(this.id);
        this.$refs.toast.show(this.$t('home.toast.withdrawn'), 'success');
      } catch (error) {
        console.error('Error withdrawing from match:', error);
        this.$refs.toast.show(actionErrorMessage(error, this.$t('home.toast.withdrawError')), 'error');
      } finally {
        // Closed either way, so the toast saying what happened isn't hidden
        // behind it.
        this.confirming = false;
        await this.reloadRegistrations();
        this.busy = false;
      }
    },
    async reloadRegistrations() {
      const regs = await getMatchRegistrations(this.id).catch(() => null);
      if (Array.isArray(regs)) this.registrations = regs;
    },
  },
};
</script>
