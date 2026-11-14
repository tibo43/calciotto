<template>
  <main class="page">
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
      <p class="label">Season {{ seasonLabel }}</p>
      <h1>No matches yet</h1>
    </section>

    <template v-else>
      <section v-if="next" class="hero">
        <router-link class="hero-go" :to="matchTo(next)">
          <p class="label">Next match</p>
          <h1 class="display hero-date">{{ heroDateLabel(next.Date) }}</h1>
          <p v-if="kickoffLabel(next.ScheduledAt)" class="display hero-time">{{ kickoffLabel(next.ScheduledAt) }}</p>
          <p v-if="metaOf(next)" class="hero-meta">{{ metaOf(next) }}</p>
        </router-link>
        <div class="hero-spots">
          <div class="num big">{{ places }}</div>
          <div class="cap">places<br>left<br>{{ confirmedCount }} / {{ cap }} signed up</div>
        </div>
        <div class="bar" aria-hidden="true"><i :style="{ '--pct': barPct }"></i></div>
        <p v-if="mine && registrationsAreOpen(regState)" class="reg-you">You're signed up · {{ mine.IsWaiting ? 'waiting list' : 'confirmed' }}</p>
        <div class="actions" v-if="registrationsAreOpen(regState)">
          <button v-if="!mine" class="btn" type="button" :disabled="busy" @click="signUp">Sign up</button>
          <div v-else class="withdraw-wrap">
            <button class="btn btn-ghost" type="button" @click="confirming = true">Withdraw</button>
          </div>
        </div>
      </section>

      <section v-else class="hero hero-last">
        <p class="label">No upcoming match yet</p>
        <p class="hero-meta">{{ heroDateLabel(recent[0].Date) }}</p>
        <div class="board-teams">
          <div class="board-team"><span class="name">{{ teamName(recent[0], 0) }}</span></div>
          <div class="board-score">{{ scoreOf(recent[0]) }}</div>
          <div class="board-team"><span class="name">{{ teamName(recent[0], 1) }}</span></div>
        </div>
        <div class="actions">
          <router-link class="btn" :to="matchTo(recent[0])">See match</router-link>
        </div>
      </section>

      <router-link v-if="voteLine" class="motm-row" :to="matchTo(voteMatch)">
        <span class="main">{{ voteLine }}</span>
      </router-link>

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

      <section v-if="recent.length" class="sec">
        <div class="sec-head"><h2>Latest matches</h2></div>
        <div class="list">
          <router-link v-for="match in recent" :key="match.ID" class="row" :to="matchTo(match)">
            <div class="row-date">{{ rowDateLabel(match.Date) }}</div>
            <div class="row-main">
              <div class="row-title">{{ teamName(match, 0) }} {{ scorePair(match) }} {{ teamName(match, 1) }}</div>
              <div v-if="weekdayTimeLabel(match.ScheduledAt)" class="row-sub">{{ weekdayTimeLabel(match.ScheduledAt) }}</div>
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
    <div class="confirm-pop" role="dialog">
      <h3>Withdraw from this match?</h3>
      <p class="ref">Your spot goes to the first player on the waiting list.</p>
      <div class="confirm-actions">
        <button class="btn btn-danger" type="button" :disabled="busy" @click="withdraw">Withdraw</button>
        <button class="btn btn-ghost" type="button" @click="confirming = false">Cancel</button>
      </div>
    </div>
  </main>
</template>

<script>
import { getMatchesDetails, getMatchRegistrations, getMatchVotes, registerForMatch, unregisterFromMatch, getToken } from '@/services/api';
import { deriveRegistrationState, registrationsAreOpen } from '@/services/matchRegistration';
import {
  splitHomeMatches, heroDateLabel, rowDateLabel, kickoffLabel, weekdayTimeLabel,
  registrationMeta, placesLeft, homeVoteLine,
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
      recentVotes: null,
      confirming: false,
      busy: false,
      nowMs: Date.now(),
      playerId: '',
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
    mine() {
      return this.registrations.find((entry) => entry.PlayerID === this.playerId) || null;
    },
    voteMatch() {
      return this.recent[0] || null;
    },
    voteLine() {
      return homeVoteLine(this.voteMatch, this.recentVotes, this.nowMs);
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
    heroDateLabel, rowDateLabel, kickoffLabel, weekdayTimeLabel, registrationsAreOpen,
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
    scoreOf(match) {
      const text = this.scorePair(match);
      const [a, b] = text.split('–');
      return `${a}–${b}`;
    },
    async load() {
      this.loading = true;
      this.failed = false;
      this.nowMs = Date.now();
      try {
        const matches = await getMatchesDetails(this.activeGroupId, this.season);
        this.matches = Array.isArray(matches) ? matches : [];
        const heroSplit = splitHomeMatches(this.matches, this.nowMs);
        this.registrations = [];
        this.recentVotes = null;
        if (heroSplit.next) {
          const list = await getMatchRegistrations(heroSplit.next.ID);
          this.registrations = Array.isArray(list) ? list : [];
        }
        if (heroSplit.recent[0]) {
          this.recentVotes = await getMatchVotes(heroSplit.recent[0].ID).catch(() => null);
        }
      } catch (error) {
        console.error('Error fetching matches:', error);
        this.failed = true;
        this.matches = [];
      } finally {
        this.loading = false;
      }
    },
    async signUp() {
      if (!this.next) return;
      this.busy = true;
      try {
        await registerForMatch(this.next.ID);
        await this.load();
      } catch (error) {
        console.error(error);
      } finally {
        this.busy = false;
      }
    },
    async withdraw() {
      if (!this.next) return;
      this.busy = true;
      try {
        await unregisterFromMatch(this.next.ID);
        this.confirming = false;
        await this.load();
      } catch (error) {
        console.error(error);
      } finally {
        this.busy = false;
      }
    },
  },
};
</script>
