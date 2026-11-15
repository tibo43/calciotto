<template>
  <main class="page" :class="{ 'has-fab': isAdmin }">
    <template v-if="loading">
      <p class="label">Next match</p>
      <div class="skel lg"></div>
      <div class="skel md"></div>
      <div class="skel sm"></div>
    </template>

    <section v-else-if="failed" class="state">
      <p class="label">Error</p>
      <h1>Couldn't load the matches</h1>
      <button class="btn" type="button" @click="load">Try again</button>
    </section>

    <section v-else-if="!next && recent.length === 0" class="state">
      <p class="label">No matches yet</p>
      <h1>Start the season</h1>
      <p v-if="isAdmin">Create the first match to open sign-ups.</p>
      <p v-else>An admin will create the first match.</p>
      <router-link v-if="isAdmin" class="btn" :to="{ name: 'NewMatch' }">Create match</router-link>
    </section>

    <template v-else>
      <section v-if="next" class="hero hero-home">
        <router-link class="hero-go" :to="matchTo(next)">
          <p class="label">Next match</p>
          <h1 class="display hero-date">{{ heroDateLabel(next.Date) }}</h1>
          <p v-if="kickoffLabel(next.ScheduledAt)" class="display hero-time">{{ kickoffLabel(next.ScheduledAt) }}</p>
        </router-link>
        <!-- The way into the match, on the right: its page for a member, its
             editing page for an admin. -->
        <div class="hero-status">
          <router-link v-if="isAdmin" class="btn btn-ghost hero-status-btn" :to="`/matches/${next.ID}/edit`">Manage match</router-link>
          <router-link v-else class="btn btn-ghost hero-status-btn" :to="matchTo(next)">View match</router-link>
        </div>
        <!-- Once the teams are set the places are history: what matters is
             which side the player is on. -->
        <div v-if="teamsSet" class="hero-team">
          <template v-if="myTeam">
            <!-- One line, one face: only the team takes the accent. -->
            <p class="label hero-team-line">You play for <span class="hero-team-mine">{{ myTeam }}</span></p>
          </template>
          <template v-else>
            <p class="label">Teams are set</p>
            <p class="hero-meta">{{ teamName(next, 0) }} vs {{ teamName(next, 1) }}</p>
          </template>
        </div>
        <template v-else>
          <div class="hero-spots">
            <div class="num big">{{ places }}</div>
            <div class="cap">places<br>left<br>{{ confirmedCount }} / {{ cap }} signed up</div>
          </div>
          <div class="bar" aria-hidden="true"><i :style="{ '--pct': barPct }"></i></div>
        </template>
        <p v-if="mine && registrationsAreOpen(regState)" class="reg-you">You're signed up · {{ mine.IsWaiting ? 'waiting list' : 'confirmed' }}</p>
        <p v-else-if="full && registrationsAreOpen(regState)" class="reg-you">Full · you'd be <strong>#{{ waitingPosition }}</strong> on the waiting list</p>
        <div class="actions" v-if="registrationsAreOpen(regState)">
          <button v-if="!mine" class="btn" type="button" :disabled="busy" @click="signUp">{{ full ? 'Join the waiting list' : 'Sign up' }}</button>
          <span v-else class="withdraw-wrap withdraw-slot">
            <button class="btn btn-ghost" type="button" @click="confirming = true">Withdraw</button>
            <!-- Wide screens: anchored under the button, as on the match page. -->
            <div class="confirm-pop" role="dialog" aria-labelledby="home-withdraw-pop-title">
              <h3 id="home-withdraw-pop-title">Withdraw from this match?</h3>
              <p class="ref">Your spot goes to the first player on the waiting list.</p>
              <div class="confirm-actions">
                <button class="btn btn-danger" type="button" :disabled="busy" @click="withdraw">Withdraw</button>
                <button class="btn btn-ghost" type="button" @click="confirming = false">Cancel</button>
              </div>
            </div>
          </span>
        </div>
      </section>

      <!-- Nothing ahead: the last match keeps the header, with its score,
           until the next one is created (then it heads Latest matches). -->
      <section v-else class="hero hero-home hero-last" :class="{ 'has-motm': voteLine }">
        <router-link class="hero-go" :to="matchTo(last)">
          <p class="label">Last match</p>
          <h1 class="display hero-date">{{ heroDateLabel(last.Date) }}</h1>
        </router-link>
        <div class="board-teams hero-board">
          <div class="board-team"><span class="name">{{ teamName(last, 0) }}</span></div>
          <div class="board-score">{{ scoreOf(last) }}</div>
          <div class="board-team"><span class="name">{{ teamName(last, 1) }}</span></div>
        </div>
        <div class="hero-status">
          <router-link v-if="isAdmin" class="btn btn-ghost hero-status-btn" :to="`/matches/${last.ID}/edit`">Manage match</router-link>
          <router-link v-else class="btn btn-ghost hero-status-btn" :to="matchTo(last)">View match</router-link>
        </div>
        <!-- Part of the header: its vote, while open, under the score. -->
        <router-link v-if="voteLine" class="motm-row hero-motm" :to="matchTo(voteMatch)">
          <span class="main">{{ voteLine }}</span>
        </router-link>
      </section>

      <section v-if="later.length" class="sec">
        <div class="sec-head"><h2>Later</h2></div>
        <div class="list">
          <router-link v-for="match in later" :key="match.ID" class="row" :to="matchTo(match)">
            <div class="row-date">{{ rowDateLabel(match.Date) }}</div>
            <div class="row-main">
              <div v-if="weekdayTimeLabel(match.ScheduledAt)" class="row-title">{{ weekdayTimeLabel(match.ScheduledAt) }}</div>
              <div class="row-sub">{{ metaOf(match) }}</div>
            </div>
            <span class="chip">Upcoming</span>
          </router-link>
        </div>
      </section>

      <section v-if="latest.length" class="sec">
        <div class="sec-head"><h2>Latest matches</h2></div>
        <div class="list">
          <!-- The score only on the right; the accessible name still reads it,
               with the day, and the sub-line describes the row. -->
          <router-link
            v-for="match in latest" :key="match.ID" class="row" :to="matchTo(match)"
            :aria-label="rowAriaLabel(match)" :aria-describedby="rowSubLine(match, nowMs) ? `row-sub-${match.ID}` : null"
          >
            <div class="row-date">{{ rowDateLabel(match.Date) }}</div>
            <div class="row-main">
              <div class="row-title row-teams"><span class="row-team">{{ teamName(match, 0) }}</span> <span class="row-vs">vs</span> <span class="row-team">{{ teamName(match, 1) }}</span></div>
              <!-- One grey line: « MOTM vote open » while its vote is open,
                   else its weekday and kick-off time; the row leads to it. -->
              <div v-if="rowSubLine(match, nowMs)" :id="`row-sub-${match.ID}`" class="row-sub" :class="{ 'row-sub-vote': rowSubLine(match, nowMs) === 'MOTM vote open' }">{{ rowSubLine(match, nowMs) }}</div>
            </div>
            <div class="score">{{ scoreOf(match) }}</div>
          </router-link>
        </div>
      </section>
    </template>

    <div class="confirm-scrim" @click="confirming = false"></div>
    <div class="confirm-sheet" role="dialog" aria-labelledby="withdraw-title">
      <h3 id="withdraw-title">Withdraw from this match?</h3>
      <p class="ref">Your spot goes to the first player on the waiting list.</p>
      <div class="confirm-actions">
        <button class="btn btn-danger" type="button" :disabled="busy" @click="withdraw">Withdraw</button>
        <button class="btn btn-ghost" type="button" @click="confirming = false">Cancel</button>
      </div>
    </div>

    <AppToast ref="toast" />

    <!-- Floating, so it's there whatever the page shows, including when a
         match is already scheduled. -->
    <router-link v-if="isAdmin && !loading" class="fab" :to="{ name: 'NewMatch' }" aria-label="New match" title="New match">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 4v16M4 12h16" /></svg>
    </router-link>
  </main>
</template>

<script>
import AppToast from '@/components/AppToast.vue';
import { actionErrorMessage } from '@/services/v2Match';
import { getMatchesDetails, getMatchRegistrations, registerForMatch, unregisterFromMatch, getToken } from '@/services/api';
import { deriveRegistrationState, registrationsAreOpen, teamsAreComposed } from '@/services/matchRegistration';
import {
  splitHomeMatches, heroDateLabel, rowDateLabel, dayMonthLabel, kickoffLabel, weekdayTimeLabel,
  registrationMeta, placesLeft, homeVoteLine, rowSubLine, myTeam, nextWaitingPosition,
} from '@/services/v2Home';

function currentPlayerIdFromToken() {
  const token = getToken();
  if (!token) return '';
  try {
    const payload = token.split('.')[1];
    const decoded = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')));
    return decoded.player_id || '';
  } catch (error) {
    return '';
  }
}

export default {
  name: 'HomeV2',
  components: { AppToast },
  inject: {
    v2Context: { default: () => ({ label: '', confirming: false }) },
  },
  props: {
    activeGroupId: { type: String, default: '' },
    groupName: { type: String, default: '' },
    season: { type: String, default: '' },
    isAdmin: { type: Boolean, default: false },
  },
  data() {
    return {
      loading: true,
      failed: false,
      matches: [],
      registrations: [],
      confirming: false,
      busy: false,
      nowMs: Date.now(),
      playerId: '',
      // Incremented on every load() so a slower, earlier request cannot
      // overwrite a newer result (in particular an unscoped first-group
      // fetch arriving after the empty group has already loaded).
      loadGeneration: 0,
    };
  },
  computed: {
    seasonLabel() {
      return (this.season || '').split('-')[0] || '';
    },
    split() {
      return splitHomeMatches(this.matches, this.nowMs);
    },
    next() { return this.split.next; },
    later() { return this.split.later; },
    recent() { return this.split.recent; },
    // The last match is in the header until the next one is created: it
    // is never listed twice.
    latest() { return this.next ? this.recent : this.recent.slice(1); },
    regState() {
      return this.next ? deriveRegistrationState(this.next, this.nowMs) : '';
    },
    confirmed() {
      return this.registrations.filter((entry) => !entry.IsWaiting);
    },
    confirmedCount() {
      return this.confirmed.length || Number(this.next && this.next.RegistrationCount) || 0;
    },
    cap() {
      return Number(this.next && this.next.MaxPlayers) || 0;
    },
    places() {
      return placesLeft(this.next, this.confirmedCount);
    },
    barPct() {
      if (!this.cap) return '0%';
      return `${Math.min(100, Math.round((this.confirmedCount / this.cap) * 100))}%`;
    },
    full() {
      return Boolean(this.cap) && this.places === 0;
    },
    waitingPosition() {
      return nextWaitingPosition(this.registrations);
    },
    teamsSet() {
      return Boolean(this.next) && teamsAreComposed(this.next);
    },
    myTeam() {
      return this.teamsSet ? myTeam(this.next, this.playerId) : null;
    },
    mine() {
      return this.registrations.find((entry) => entry.PlayerID === this.playerId) || null;
    },
    // The header match: the next one, or else the last one played.
    last() {
      return this.next ? null : (this.recent[0] || null);
    },
    voteMatch() {
      return this.next || this.last;
    },
    // Only about the header match, and only while its vote is open.
    voteLine() {
      return homeVoteLine(this.voteMatch, this.nowMs);
    },
  },
  watch: {
    activeGroupId() { this.load(); },
    season() { this.load(); },
    confirming(value) {
      this.v2Context.confirming = value;
    },
    groupName() { this.publishContext(); },
    seasonLabel() { this.publishContext(); },
  },
  created() {
    this.playerId = currentPlayerIdFromToken();
    this.publishContext();
    this.load();
  },
  methods: {
    heroDateLabel, rowDateLabel, kickoffLabel, weekdayTimeLabel, rowSubLine, registrationsAreOpen,
    publishContext() {
      this.v2Context.hide = false;
    },
    matchTo(match) {
      return { name: 'MatchView', params: { id: match.ID } };
    },
    metaOf(match) {
      return registrationMeta(match, this.nowMs);
    },
    teamName(match, index) {
      const team = match && match.Teams && match.Teams[index];
      return (team && team.Name) || '—';
    },
    scorePair(match) {
      const teams = (match && match.Teams) || [];
      const a = teams[0] ? teams[0].Score : 0;
      const b = teams[1] ? teams[1].Score : 0;
      return `${a}–${b}`;
    },
    // « Les Bleus 4–6 Les Rouges, 4 Sep »: the score the row shows on the
    // right, read with the teams and the day.
    rowAriaLabel(match) {
      return `${this.teamName(match, 0)} ${this.scorePair(match)} ${this.teamName(match, 1)}, ${dayMonthLabel(match.Date)}`;
    },
    scoreOf(match) {
      const text = this.scorePair(match);
      const [a, b] = text.split('–');
      return `${a}–${b}`;
    },
    async load() {
      // No group_id means the backend falls back to the player's first
      // group. Wait until the parent has resolved the active one, and
      // keep the skeleton up rather than painting another group's data.
      if (!this.activeGroupId) {
        return;
      }
      const generation = ++this.loadGeneration;
      this.loading = true;
      this.failed = false;
      this.nowMs = Date.now();
      try {
        const matches = await getMatchesDetails(this.activeGroupId, this.season);
        if (generation !== this.loadGeneration) return;
        this.matches = Array.isArray(matches) ? matches : [];
        const heroSplit = splitHomeMatches(this.matches, this.nowMs);
        this.registrations = [];
        if (heroSplit.next) {
          const list = await getMatchRegistrations(heroSplit.next.ID);
          if (generation !== this.loadGeneration) return;
          this.registrations = Array.isArray(list) ? list : [];
        }
      } catch (error) {
        if (generation !== this.loadGeneration) return;
        console.error('Error fetching matches:', error);
        this.failed = true;
        this.matches = [];
      } finally {
        if (generation === this.loadGeneration) this.loading = false;
      }
    },
    async signUp() {
      if (!this.next) return;
      this.busy = true;
      try {
        const entry = await registerForMatch(this.next.ID);
        if (entry && entry.IsWaiting) {
          this.$refs.toast.show(`Signed up — you are #${entry.Position}, on the waiting list.`, 'success');
        } else {
          const position = entry && entry.Position ? ` You are #${entry.Position}.` : '';
          this.$refs.toast.show(`You're in for this match.${position}`, 'success');
        }
        await this.load();
      } catch (error) {
        console.error(error);
        this.$refs.toast.show(actionErrorMessage(error, 'Error signing up for this match.'), 'error');
        await this.load();
      } finally {
        this.busy = false;
      }
    },
    async withdraw() {
      if (!this.next) return;
      this.busy = true;
      try {
        await unregisterFromMatch(this.next.ID);
        this.$refs.toast.show('You have withdrawn from this match.', 'success');
        await this.load();
      } catch (error) {
        console.error(error);
        this.$refs.toast.show(actionErrorMessage(error, 'Error withdrawing from this match.'), 'error');
        await this.load();
      } finally {
        // Closed either way, so the toast isn't hidden behind it.
        this.confirming = false;
        this.busy = false;
      }
    },
  },
};
</script>
