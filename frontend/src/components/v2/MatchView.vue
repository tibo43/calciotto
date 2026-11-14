<template>
  <main class="page" :class="{ 'match-upcoming': upcoming }">
    <router-link class="back" to="/">
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M10 3 5 8l5 5"/></svg>
      Back to home
    </router-link>

    <section v-if="loading" class="state">
      <div class="skel lg"></div>
      <div class="skel md"></div>
    </section>
    <section v-else-if="failed" class="state">
      <h1>Couldn't load the match</h1>
      <button class="btn" type="button" @click="load">Try again</button>
    </section>

    <template v-else-if="upcoming">
      <section class="hero">
        <p class="label">Next match</p>
        <h1 class="display hero-date">{{ heroDateLabel(match.Date) }}</h1>
        <p v-if="kickoffLabel(match.ScheduledAt)" class="display hero-time">{{ kickoffLabel(match.ScheduledAt) }}</p>
        <p v-if="meta" class="hero-meta">{{ meta }}</p>
        <div class="hero-spots">
          <div class="num big">{{ places }}</div>
          <div class="cap">places<br>left<br>{{ confirmed.length }} / {{ cap }} signed up</div>
        </div>
        <div class="bar" aria-hidden="true"><i :style="{ '--pct': barPct }"></i></div>
        <div class="actions">
          <button v-if="!mine && open" class="btn" type="button" @click="signUp">Sign up</button>
          <p v-else-if="mine" class="reg-you">You're signed up · {{ mine.IsWaiting ? 'waiting list' : 'confirmed' }}</p>
          <router-link v-if="isAdmin" class="btn btn-ghost" :to="`/matches/${match.ID}/edit`">Manage match</router-link>
        </div>
      </section>
      <section class="sec">
        <div class="sec-head"><h2>Confirmed</h2><span class="label">{{ confirmed.length }} / {{ cap }}</span></div>
        <ul class="signup-ro">
          <li v-for="entry in confirmed" :key="entry.PlayerID">
            <span class="nm">{{ entry.Name }}</span>
            <span class="pos">#{{ entry.Position }}</span>
          </li>
        </ul>
      </section>
      <section v-if="waiting.length" class="sec">
        <div class="sec-head"><h2>Waiting list</h2><span class="label">{{ waiting.length }}</span></div>
        <ul class="signup-ro">
          <li v-for="entry in waiting" :key="entry.PlayerID">
            <span class="nm">{{ entry.Name }}</span>
            <span class="wait">Waiting</span>
          </li>
        </ul>
      </section>
    </template>

    <template v-else>
      <section class="board">
        <p class="label board-date">{{ heroDateLabel(match.Date) }}</p>
        <div class="board-teams">
          <div class="board-team"><span class="dot" :style="{ '--c': colour(0) }"></span><span class="name">{{ teamName(0) }}</span></div>
          <div class="board-score">{{ scoreText }}</div>
          <div class="board-team"><span class="dot" :style="{ '--c': colour(1) }"></span><span class="name">{{ teamName(1) }}</span></div>
        </div>
      </section>
      <p class="motm-line">{{ voteCopy }}</p>
      <div class="lineups">
        <section v-for="(team, index) in teams" :key="team.ID" class="lineup">
          <h3><span class="dot" :style="{ '--c': colour(index) }"></span>{{ team.Name }}</h3>
          <template v-for="player in team.Players" :key="player.ID">
          <button v-if="canVote(player)" class="player" :class="{ 'is-winner': isWinner(player) }" type="button" :aria-label="voteAria(player)" @click="castVote(player)">
            <span class="nm">{{ player.Name }}</span>
            <span class="player-meta">
              <span v-if="goalLabel(player)" class="goals">{{ goalLabel(player) }}</span>
              <span class="votes"><span class="star" aria-hidden="true">{{ starFor(player) }}</span>{{ votesFor(player) }}</span>
            </span>
          </button>
          <div v-else class="player" :class="{ 'is-self': player.ID === playerId, 'is-winner': isWinner(player) }">
            <span class="nm">{{ player.Name }}</span>
            <span class="player-meta">
              <span v-if="goalLabel(player)" class="goals">{{ goalLabel(player) }}</span>
              <span class="votes"><span class="star" aria-hidden="true">{{ starFor(player) }}</span>{{ votesFor(player) }}</span>
            </span>
          </div>
          </template>
        </section>
      </div>
      <div v-if="isAdmin" class="actions" style="margin-top: 24px">
        <router-link class="btn btn-ghost" :to="`/matches/${match.ID}/edit`">Manage match</router-link>
      </div>
    </template>
  </main>
</template>

<script>
import { getMatchDetailsByID, getMatchRegistrations, registerForMatch, getMatchVotes, voteForMotm, getSeasons, getToken } from '@/services/api';
import { resolveActiveGroup, setActiveGroupId } from '@/services/activeGroup';
import { deriveRegistrationState, registrationsAreOpen, isPlayerOnRoster } from '@/services/matchRegistration';
import { heroDateLabel, kickoffLabel, registrationMeta, placesLeft, isMatchOver, matchVoteCopy, motmWinners, goalLine } from '@/services/v2Home';
import { isMotmVotingOpen } from '@/services/motmVoting';

const COLOURS = { red: '#c0392b', blue: '#2471a3', black: '#111', white: '#eef0f7', green: '#1e8449', yellow: '#f4d03f', orange: '#e8743b' };

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
  name: 'MatchView',
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
    open() {
      return registrationsAreOpen(deriveRegistrationState(this.match, this.nowMs));
    },
    meta() {
      return registrationMeta(this.match, this.nowMs);
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
  },
  created() {
    this.playerId = currentPlayerIdFromToken();
    this.load();
  },
  methods: {
    heroDateLabel,
    kickoffLabel,
    colour(index) {
      const team = this.teams[index];
      return COLOURS[(team && team.Colour) || ''] || '#8b93ad';
    },
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
    isWinner(player) {
      return this.winners.some((row) => row.PlayerID === player.ID);
    },
    canVote(player) {
      return this.voteOpen && isPlayerOnRoster(this.match, this.playerId) && player.ID !== this.playerId;
    },
    voteAria(player) {
      const count = this.votesFor(player);
      const mine = this.votes && this.votes.MyVoteFor === player.ID;
      return mine ? `${count} votes, your vote` : `${count} votes`;
    },
    async castVote(player) {
      if (!this.canVote(player)) return;
      try {
        await voteForMotm(this.match.ID, player.ID);
        this.votes = await getMatchVotes(this.match.ID);
      } catch (error) {
        console.error(error);
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
        this.match = await getMatchDetailsByID(this.id);
        const [regs, votes] = await Promise.all([
          getMatchRegistrations(this.id).catch(() => []),
          getMatchVotes(this.id).catch(() => null),
        ]);
        this.registrations = Array.isArray(regs) ? regs : [];
        this.votes = votes;
        try {
          const { groups } = await resolveActiveGroup();
          const group = groups.find((item) => item.id === this.match.GroupID);
          this.isAdmin = Boolean(group && group.role === 'admin');
          this.v2Context.groupId = group ? group.id : this.v2Context.groupId;
          await this.publishChrome(groups);
        } catch (error) {
          this.isAdmin = false;
        }
      } catch (error) {
        console.error(error);
        this.failed = true;
      } finally {
        this.loading = false;
      }
    },
    async signUp() {
      await registerForMatch(this.id);
      await this.load();
    },
  },
};
</script>
