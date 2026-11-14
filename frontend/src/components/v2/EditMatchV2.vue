<template>
  <main class="page edit-page has-sticky" :class="{ 'is-choix': pickedId }">
    <router-link class="back" :to="step === 'signups' ? `/matches/${id}` : { path: `/matches/${id}/edit` }">
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M10 3 5 8l5 5"/></svg>
      {{ step === 'signups' ? 'Back to the match' : 'Back to editing' }}
    </router-link>

    <section v-if="loading" class="state"><div class="skel lg"></div><div class="skel md"></div></section>
    <section v-else-if="failed" class="state">
      <h1>Couldn't load the match</h1>
      <button class="btn" type="button" @click="load">Try again</button>
    </section>

    <template v-else>
      <p class="label">Editing · admin</p>
      <h1 class="display h-kick">{{ heading }}</h1>
      <p class="reg-status" v-html="statusHtml"></p>

      <div class="share-wa">
        <a class="btn btn-ghost" :href="shareUrl" target="_blank" rel="noopener">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M5 19l1.2-4A8 8 0 1 1 12 20a8 8 0 0 1-3.5-.8L5 19z"/></svg>
          Share on WhatsApp
        </a>
        <p class="share-sub">{{ shareSub }}</p>
      </div>

      <nav class="steps" aria-label="Steps">
        <button class="step-i" :class="{ 'is-on': step === 'signups' }" type="button" @click="goStep('signups')"><b>1</b> Sign-ups</button>
        <button class="step-i" :class="{ 'is-on': step === 'teams' }" type="button" @click="goStep('teams')"><b>2</b> Teams</button>
        <button class="step-i" :class="{ 'is-on': step === 'score' }" type="button" @click="goStep('score')"><b>3</b> Score and goals</button>
      </nav>

      <template v-if="step === 'signups'">
        <section class="block">
          <div class="sec-head"><h2>1. Sign-ups</h2></div>
          <p class="note">{{ signupNote }}</p>
          <div class="actions" v-if="canClose || canReopen">
            <button v-if="canClose" class="btn btn-ghost" type="button" @click="closeList">Close sign-ups</button>
            <button v-if="canReopen" class="btn btn-ghost" type="button" @click="reopenList">Reopen sign-ups</button>
          </div>
          <div v-if="isScheduled" class="places-row">
            <label class="label" for="confirmed-places">Confirmed places</label>
            <input id="confirmed-places" type="number" min="1" v-model="maxPlayersInput">
            <button class="btn btn-ghost" type="button" :disabled="!canUpdateCap" @click="updateCap">Update</button>
          </div>
          <p class="label">Confirmed</p>
          <div v-for="entry in confirmed" :key="entry.PlayerID" class="sign-row">
            <div class="who">{{ entry.Name }}</div>
            <span class="tog is-on">Confirmed</span>
          </div>
          <p v-if="!confirmed.length" class="note">No confirmed players.</p>
          <p class="label">Waiting list</p>
          <div v-for="entry in waiting" :key="entry.PlayerID" class="sign-row">
            <div class="who">{{ entry.Name }}</div>
            <span class="tog">Waiting list</span>
          </div>
          <p v-if="!waiting.length" class="note">No one is waiting.</p>
        </section>
        <section class="block">
          <div class="sec-head"><h2>2. Teams</h2></div>
          <p class="note">{{ teamsNote }}</p>
          <div class="actions">
            <button class="btn" type="button" @click="goStep('teams')">Compose the teams</button>
          </div>
        </section>
        <section class="block">
          <div class="sec-head"><h2>3. Score and goals</h2></div>
          <p class="note">The score is not typed in. It is the sum of each team's goals, plus the other team's own goals. It opens once the teams are made.</p>
          <div class="actions">
            <button v-if="bothSides" class="btn" type="button" @click="goStep('score')">Enter the score</button>
            <button v-else class="btn btn-ghost" type="button" @click="goStep('teams')">Make the teams first</button>
          </div>
        </section>
        <p v-if="message" class="note">{{ message }}</p>
        <div class="sticky-cta">
          <button class="btn btn-ghost btn-danger" type="button" @click="confirmDelete">Delete</button>
          <button class="btn" type="button" :disabled="saving" @click="saveDraft">Save</button>
        </div>
      </template>

      <template v-else-if="step === 'teams'">
        <div class="compose-toolbar">
          <button class="btn" type="button" :disabled="!canSplit" @click="split">Split automatically</button>
          <button class="btn btn-ghost" type="button" @click="clearTeams">Clear</button>
          <p class="hint">{{ splitHint }}</p>
        </div>
        <p class="tap-hint">Tap a player to move them onto a team, or back to the reserve.</p>
        <p class="size-count">{{ sizeCount }}</p>
        <div class="compose-board">
          <section class="compose-col" aria-label="Reserve">
            <div class="col-head"><h2>Reserve</h2></div>
            <p class="col-count">{{ reserve.length }}</p>
            <p v-if="!reserve.length" class="col-empty">Every confirmed player is placed.</p>
            <button v-for="player in reserve" :key="player.ID" class="p-row" :class="{ 'is-picked': pickedId === player.ID }" type="button" @click="pick(player)">
              <svg class="grip" viewBox="0 0 16 16" aria-hidden="true"><circle cx="6" cy="4" r="1.2" fill="currentColor"/><circle cx="10" cy="4" r="1.2" fill="currentColor"/><circle cx="6" cy="8" r="1.2" fill="currentColor"/><circle cx="10" cy="8" r="1.2" fill="currentColor"/><circle cx="6" cy="12" r="1.2" fill="currentColor"/><circle cx="10" cy="12" r="1.2" fill="currentColor"/></svg>
              <span class="pos">{{ posLabel(player) }}</span>
              <span class="nm">{{ player.Name }}</span>
            </button>
          </section>
          <section v-for="index in [0, 1]" :key="teamId(index)" class="compose-col" :class="{ 'is-drop': pickedId && teamOf(pickedId) === index }" :aria-label="teamName(index)">
            <div class="col-head">
              <span class="dot" :style="{ '--c': colour(index) }"></span>
              <h2>{{ teamName(index) }}</h2>
            </div>
            <p class="col-count">{{ teamPlayers[index].length }}</p>
            <p v-if="!teamPlayers[index].length" class="col-empty">No players yet.</p>
            <button v-for="player in teamPlayers[index]" :key="player.ID" class="p-row" :class="{ 'is-picked': pickedId === player.ID }" type="button" @click="pick(player)">
              <svg class="grip" viewBox="0 0 16 16" aria-hidden="true"><circle cx="6" cy="4" r="1.2" fill="currentColor"/><circle cx="10" cy="4" r="1.2" fill="currentColor"/><circle cx="6" cy="8" r="1.2" fill="currentColor"/><circle cx="10" cy="8" r="1.2" fill="currentColor"/><circle cx="6" cy="12" r="1.2" fill="currentColor"/><circle cx="10" cy="12" r="1.2" fill="currentColor"/></svg>
              <span class="pos">{{ posLabel(player) }}</span>
              <span class="nm">{{ player.Name }}</span>
            </button>
          </section>
        </div>
        <section class="sec">
          <div class="sec-head"><h2>Waiting list</h2></div>
          <p class="note">These players are not included in the automatic split. The first player on the waiting list moves up when a confirmed place opens.</p>
          <div v-for="entry in waiting" :key="entry.PlayerID" class="wait-row">
            <div>
              <div class="who">{{ entry.Name }}</div>
              <div class="meta">Waiting · #{{ entry.Position }}</div>
            </div>
          </div>
          <div class="actions">
            <button class="btn btn-ghost" type="button" @click="toggleGuests">Add a player who didn't sign up</button>
          </div>
          <div v-if="addingGuest" class="guest-list">
            <button v-for="member in addable" :key="member.id" class="csc-pick" type="button" @click="addGuest(member)">
              <span class="nm">{{ member.name }}</span>
            </button>
            <p v-if="!addable.length" class="note">Everyone in the group is already here.</p>
          </div>
        </section>
        <p v-if="message" class="note">{{ message }}</p>
        <div class="sticky-cta">
          <p v-if="!bothSides" class="cta-msg">Put at least one player on each team to confirm.</p>
          <button class="btn" :class="{ 'is-off': !bothSides }" type="button" :disabled="!bothSides || saving" @click="saveDraft">Confirm the teams</button>
        </div>
        <div v-if="picked" class="sheet" role="dialog" aria-label="Choose the team">
          <p class="label">Assign</p>
          <h3>{{ picked.Name }}</h3>
          <p class="ref">{{ pickedWhere }}</p>
          <button class="btn" type="button" @click="assign(0)">{{ teamName(0) }}</button>
          <button class="btn btn-ghost" type="button" @click="assign(1)">{{ teamName(1) }}</button>
          <button class="btn btn-ghost" type="button" @click="assign(null)">Reserve</button>
          <button class="btn btn-ghost" type="button" @click="pickedId = null">Cancel</button>
        </div>
      </template>

      <template v-else>
        <template v-if="!bothSides">
          <p class="note">The score opens once each team has at least one player.</p>
          <button class="btn" type="button" @click="goStep('teams')">Compose the teams</button>
        </template>
        <template v-else>
          <section class="score-board" aria-label="Computed score">
            <div class="board-teams">
              <div class="board-team"><span class="dot" :style="{ '--c': colour(0) }"></span><span class="name">{{ teamName(0) }}</span></div>
              <div class="board-score">{{ sides[0].score }}<span class="sep">–</span>{{ sides[1].score }}</div>
              <div class="board-team"><span class="dot" :style="{ '--c': colour(1) }"></span><span class="name">{{ teamName(1) }}</span></div>
            </div>
            <p class="balance">{{ teamName(0) }} {{ teamPlayers[0].length }} · {{ teamName(1) }} {{ teamPlayers[1].length }}</p>
            <p class="score-status">{{ scoreStatus }}</p>
            <ul class="scorers">
              <li v-for="(line, index) in recap" :key="index" :class="{ away: line.away }">{{ line.text }}</li>
            </ul>
            <p class="score-hint">The score is the sum of each team's goals, plus the other team's own goals. An own goal does not count in the scorers table.</p>
          </section>
          <div class="score-teams">
            <section v-for="index in [0, 1]" :key="teamId(index)" class="score-team" :aria-label="`Goals for ${teamName(index)}`">
              <h2><span class="dot" :style="{ '--c': colour(index) }"></span>{{ teamName(index) }}</h2>
              <p class="team-sub">{{ teamScoreLine(sides[index]) }}</p>
              <div v-for="player in teamPlayers[index]" :key="player.ID" class="score-row">
                <div class="who">
                  <span class="nm">{{ player.Name }}</span>
                  <span v-if="player.OwnGoals" class="og-plain">{{ player.OwnGoals }} OG</span>
                </div>
                <div class="stepper">
                  <button type="button" :disabled="!player.GoalNumber" :aria-label="`Remove a goal for ${player.Name}`" @click="changeGoals(player, -1)">−</button>
                  <span class="n">{{ player.GoalNumber || 0 }}</span>
                  <button type="button" :aria-label="`Add a goal for ${player.Name}`" @click="changeGoals(player, 1)">+</button>
                </div>
              </div>
              <div v-for="player in ownGoalsFor(index)" :key="`og-${player.ID}`" class="score-row">
                <div class="who">
                  <span class="nm">{{ player.Name }}</span>
                  <span class="og-plain">OG</span>
                </div>
                <div class="stepper">
                  <button type="button" :aria-label="`Remove an own goal for ${player.Name}`" @click="changeOwnGoals(player, -1)">−</button>
                  <span class="n">{{ player.OwnGoals }}</span>
                  <button type="button" :aria-label="`Add an own goal for ${player.Name}`" @click="changeOwnGoals(player, 1)">+</button>
                </div>
              </div>
              <div class="csc-add-wrap" :class="{ 'is-open': pickingFor === index }">
                <button class="btn btn-ghost btn-csc-add" type="button" @click="openOwnGoal(index)">+ Own goal</button>
                <div v-if="pickingFor === index" class="csc-pop" role="dialog" aria-labelledby="og-pop-title">
                  <h3 id="og-pop-title">Own goal</h3>
                  <p class="ref">{{ ownGoalHelp(index) }}</p>
                  <button v-for="player in teamPlayers[1 - index]" :key="player.ID" class="csc-pick" type="button" @click="pickOwnGoal(player)">
                    <span class="nm">{{ player.Name }}</span>
                  </button>
                  <button class="btn btn-ghost" type="button" @click="pickingFor = null">Cancel</button>
                </div>
              </div>
            </section>
          </div>
          <div v-if="pickingFor !== null" class="csc-scrim" @click="pickingFor = null"></div>
          <div v-if="pickingFor !== null" class="csc-sheet" role="dialog" aria-labelledby="og-sheet-title">
            <h3 id="og-sheet-title">Own goal</h3>
            <p class="ref">{{ ownGoalHelp(pickingFor) }}</p>
            <button v-for="player in teamPlayers[1 - pickingFor]" :key="player.ID" class="csc-pick" type="button" @click="pickOwnGoal(player)">
              <span class="nm">{{ player.Name }}</span>
            </button>
            <button class="btn btn-ghost" type="button" @click="pickingFor = null">Cancel</button>
          </div>
          <p v-if="message" class="note">{{ message }}</p>
          <div class="sticky-cta">
            <button class="btn" type="button" :disabled="saving" @click="saveDraft">Save changes</button>
          </div>
        </template>
      </template>
    </template>
  </main>
</template>

<script>
import {
  getMatchDetailsByID, getMatchRegistrations, closeMatchRegistrations, reopenMatchRegistrations,
  setMatchMaxPlayers, updateMatch, deleteMatch, getInviteCode, getGroupMembers, getSeasons,
} from '@/services/api';
import { resolveActiveGroup, setActiveGroupId } from '@/services/activeGroup';
import {
  deriveRegistrationState, REGISTRATION_OPEN, REGISTRATION_CLOSED_BY_ADMIN,
  fillTeamsFromRegistrations,
} from '@/services/matchRegistration';
import { formatDateTimeForDisplay, formatCalendarDayLong } from '@/services/datetime';
import { heroDateLabel, kickoffLabel } from '@/services/v2Home';
import { encodeMatchId } from '@/services/shortLink';
import { buildWhatsAppShareText, buildWhatsAppShareUrl, buildTeamsShareText } from '@/services/whatsappShare';
import { scoreSides, teamScoreLine, sizeCounter, recapEntries, lineupsReady, shareSubtitle } from '@/services/v2Edit';

const COLOURS = {
  red: '#c0392b', blue: '#2471a3', black: '#111', white: '#eef0f7',
  green: '#1e8449', yellow: '#f4d03f', orange: '#e8743b',
};

function copyPlayer(player, position) {
  return {
    ID: player.ID,
    Name: player.Name,
    GoalNumber: Number(player.GoalNumber) || 0,
    OwnGoals: Number(player.OwnGoals) || 0,
    Position: position || null,
  };
}

export default {
  name: 'EditMatchV2',
  inject: {
    v2Context: { default: () => ({}) },
  },
  props: {
    id: { type: String, required: true },
  },
  data() {
    return {
      loading: true,
      failed: false,
      saving: false,
      step: 'signups',
      match: null,
      registrations: [],
      teamPlayers: [[], []],
      held: {},
      guests: [],
      members: [],
      addingGuest: false,
      pickedId: null,
      pickingFor: null,
      maxPlayersInput: '',
      inviteCode: '',
      groupId: '',
      message: '',
      hadScore: false,
      nowMs: Date.now(),
    };
  },
  computed: {
    isScheduled() {
      return Boolean(this.match && this.match.ScheduledAt);
    },
    registrationState() {
      return deriveRegistrationState(this.match, this.nowMs);
    },
    canClose() {
      return this.isScheduled && this.registrationState === REGISTRATION_OPEN;
    },
    canReopen() {
      return this.isScheduled && this.registrationState === REGISTRATION_CLOSED_BY_ADMIN;
    },
    confirmed() {
      return this.registrations.filter((entry) => entry && !entry.IsWaiting);
    },
    waiting() {
      return this.registrations.filter((entry) => entry && entry.IsWaiting);
    },
    cap() {
      return Number(this.match && this.match.MaxPlayers) || this.confirmed.length;
    },
    canUpdateCap() {
      const parsed = Number(this.maxPlayersInput);
      return Number.isInteger(parsed) && parsed >= 1 && parsed !== this.cap;
    },
    reserve() {
      const placed = new Set(this.teamPlayers.flat().map((player) => player.ID));
      const rows = [];
      this.confirmed.forEach((entry) => {
        if (placed.has(entry.PlayerID)) return;
        rows.push(this.materialize({ ID: entry.PlayerID, Name: entry.Name, Position: entry.Position }));
      });
      this.guests.forEach((guest) => {
        if (!placed.has(guest.ID)) rows.push(this.materialize(guest));
      });
      return rows;
    },
    bothSides() {
      return this.teamPlayers[0].length > 0 && this.teamPlayers[1].length > 0;
    },
    canSplit() {
      return this.confirmed.some((entry) => !this.teamPlayers.flat().some((player) => player.ID === entry.PlayerID));
    },
    sides() {
      return scoreSides(this.teamPlayers);
    },
    recap() {
      return recapEntries(this.teamPlayers);
    },
    heading() {
      if (!this.match) return '';
      const date = heroDateLabel(this.match.Date);
      const time = kickoffLabel(this.match.ScheduledAt);
      return time ? `${date} · ${time}` : date;
    },
    statusHtml() {
      if (!this.isScheduled) return 'No sign-up list';
      const word = this.registrationState === REGISTRATION_OPEN ? 'open'
        : this.registrationState === 'not-open-yet' ? 'not open yet' : 'closed';
      if (this.step === 'score' && this.bothSides) return `Sign-ups <strong>${word}</strong> · teams made`;
      const wait = this.waiting.length === 1 ? '1 waiting' : `${this.waiting.length} waiting`;
      return `Sign-ups <strong>${word}</strong> · <strong>${this.confirmed.length} confirmed / ${this.cap} places</strong> · ${wait}`;
    },
    signupNote() {
      if (this.bothSides) return 'The teams are made. Sign-ups above stay as they are.';
      const names = this.waiting.map((entry) => entry.Name).join(', ');
      const bench = names ? `, ${this.waiting.length} waiting (${names})` : '';
      return `Not composed yet. ${this.confirmed.length} confirmed players are in the reserve${bench}. The waiting list is not split automatically.`;
    },
    teamsNote() {
      return this.signupNote;
    },
    sizeCount() {
      return sizeCounter(this.teamPlayers[0].length, this.teamPlayers[1].length);
    },
    splitHint() {
      return `All confirmed players are split, in sign-up order, onto the smaller team (a tie goes to ${this.teamName(0)}). An odd number leaves one extra player on one side. Players already placed stay where they are. The waiting list is not included.`;
    },
    picked() {
      if (!this.pickedId) return null;
      return this.teamPlayers.flat().find((player) => player.ID === this.pickedId)
        || this.reserve.find((player) => player.ID === this.pickedId)
        || null;
    },
    pickedWhere() {
      if (!this.picked) return '';
      const team = this.teamOf(this.picked.ID);
      const spot = this.picked.Position ? `Confirmed · #${this.picked.Position}` : 'Added by an admin';
      if (team === null) return `${spot} · in the reserve`;
      return `${spot} · currently on ${this.teamName(team)}`;
    },
    draftTeams() {
      return [0, 1].map((index) => ({
        ...(this.match.Teams[index] || {}),
        Name: this.teamName(index),
        Players: this.teamPlayers[index],
        Score: this.sides[index].score,
      }));
    },
    shareSub() {
      return shareSubtitle(this.draftTeams);
    },
    shareUrl() {
      if (!this.match) return '#';
      const when = this.isScheduled
        ? formatDateTimeForDisplay(this.match.ScheduledAt)
        : formatCalendarDayLong(this.match.Date);
      if (lineupsReady(this.draftTeams)) {
        const teams = this.draftTeams.map((team) => ({
          name: team.Name,
          players: (team.Players || []).map((player) => player.Name),
        }));
        return buildWhatsAppShareUrl(buildTeamsShareText({ dateTimeLabel: when, teams }));
      }
      const text = buildWhatsAppShareText({
        kickoffLabel: when,
        matchUrl: `${window.location.origin}/m/${encodeMatchId(this.match.ID)}`,
        groupInviteCode: this.inviteCode,
      });
      return buildWhatsAppShareUrl(text);
    },
    scoreStatus() {
      return this.hadScore ? 'Score saved · you can still change it.' : 'Score not saved yet.';
    },
    addable() {
      const taken = new Set([
        ...this.registrations.map((entry) => entry.PlayerID),
        ...this.teamPlayers.flat().map((player) => player.ID),
        ...this.guests.map((guest) => guest.ID),
      ]);
      return this.members.filter((member) => member && member.id && !taken.has(member.id));
    },
  },
  watch: {
    '$route.query.step': function onStep() {
      this.step = this.stepFromRoute();
    },
    pickingFor(value) {
      if (this.v2Context) this.v2Context.pickingCsc = value !== null;
    },
    id() { this.load(); },
  },
  created() {
    this.step = this.stepFromRoute();
    this.load();
  },
  beforeUnmount() {
    if (!this.v2Context) return;
    this.v2Context.pickingCsc = false;
    this.v2Context.confirming = false;
  },
  methods: {
    teamScoreLine,
    stepFromRoute() {
      const step = this.$route.query.step;
      return step === 'teams' || step === 'score' ? step : 'signups';
    },
    goStep(step) {
      this.pickedId = null;
      this.pickingFor = null;
      this.message = '';
      const query = step === 'signups' ? {} : { step };
      this.$router.replace({ path: `/matches/${this.id}/edit`, query });
    },
    teamName(index) {
      const team = this.match && this.match.Teams && this.match.Teams[index];
      return (team && team.Name) || 'Team';
    },
    teamId(index) {
      const team = this.match && this.match.Teams && this.match.Teams[index];
      return (team && team.ID) || index;
    },
    colour(index) {
      const team = this.match && this.match.Teams && this.match.Teams[index];
      const raw = (team && team.Colour) || '';
      if (raw.startsWith('#')) return raw;
      return COLOURS[raw.toLowerCase()] || '#8b93ad';
    },
    posLabel(player) {
      return player.Position ? `#${player.Position}` : 'Guest';
    },
    teamOf(playerId) {
      const index = this.teamPlayers.findIndex((players) => players.some((player) => player.ID === playerId));
      return index === -1 ? null : index;
    },
    materialize(row) {
      const prev = this.held[row.ID];
      if (prev) return { ...prev, Name: row.Name, Position: row.Position || prev.Position };
      return copyPlayer(row, row.Position);
    },
    remember(player) {
      this.held[player.ID] = { ...player };
    },
    ownGoalsFor(beneficiary) {
      return this.teamPlayers[1 - beneficiary].filter((player) => player.OwnGoals > 0);
    },
    ownGoalHelp(beneficiary) {
      return `This goal counts for ${this.teamName(beneficiary)}. Choose the ${this.teamName(1 - beneficiary)} player who scored it. It does not count in the scorers table.`;
    },
    initDraft() {
      const positionOf = {};
      this.registrations.forEach((entry) => { positionOf[entry.PlayerID] = entry.Position; });
      this.teamPlayers = [0, 1].map((index) => {
        const team = (this.match.Teams || [])[index] || {};
        return (team.Players || []).map((player) => {
          const copy = copyPlayer(player, positionOf[player.ID]);
          this.held[copy.ID] = { ...copy };
          return copy;
        });
      });
      const any = this.teamPlayers.flat().some((player) => player.GoalNumber || player.OwnGoals);
      this.hadScore = Boolean(this.match.ResultRecordedAt) || any;
      this.maxPlayersInput = this.match.MaxPlayers ? String(this.match.MaxPlayers) : '';
    },
    async load() {
      this.loading = true;
      this.failed = false;
      this.nowMs = Date.now();
      this.message = '';
      try {
        this.match = await getMatchDetailsByID(this.id);
        const regs = await getMatchRegistrations(this.id).catch(() => []);
        this.registrations = Array.isArray(regs) ? regs : [];
        this.guests = [];
        this.held = {};
        this.initDraft();
        try {
          const { groups } = await resolveActiveGroup();
          const group = groups.find((item) => item.id === this.match.GroupID) || groups[0];
          this.groupId = group ? group.id : '';
          if (this.v2Context && group) {
            this.v2Context.groupId = group.id;
            this.v2Context.hide = false;
            const seasons = await getSeasons(group.id).catch(() => []);
            this.v2Context.groups = groups;
            this.v2Context.seasons = Array.isArray(seasons) ? seasons : [];
            if (!this.v2Context.season && this.v2Context.seasons.length) {
              this.v2Context.season = this.v2Context.seasons[this.v2Context.seasons.length - 1];
            }
            this.v2Context.onGroup = (groupId) => {
              setActiveGroupId(groupId);
              window.location.href = '/';
            };
            this.v2Context.onSeason = (season) => {
              sessionStorage.setItem('calciotto-v2-season', season);
              window.location.href = '/';
            };
          }
          if (this.groupId) {
            const code = await getInviteCode(this.groupId).catch(() => null);
            this.inviteCode = (code && (code.invite_code || code.InviteCode)) || '';
          }
        } catch (error) {
          this.groupId = this.match.GroupID || '';
        }
      } catch (error) {
        console.error(error);
        this.failed = true;
      } finally {
        this.loading = false;
      }
    },
    async closeList() {
      await closeMatchRegistrations(this.match.ID);
      this.match = { ...this.match, RegistrationsClosedAt: new Date().toISOString() };
      this.message = 'Sign-ups closed.';
    },
    async reopenList() {
      await reopenMatchRegistrations(this.match.ID);
      this.match = { ...this.match, RegistrationsClosedAt: null };
      this.message = 'Sign-ups reopened.';
    },
    async updateCap() {
      if (!this.canUpdateCap) return;
      const requested = Number(this.maxPlayersInput);
      const entries = await setMatchMaxPlayers(this.match.ID, requested);
      this.registrations = Array.isArray(entries) ? entries : this.registrations;
      this.match = { ...this.match, MaxPlayers: requested };
      this.message = `Confirmed places set to ${requested}.`;
    },
    split() {
      const { rosters, addedCount } = fillTeamsFromRegistrations(
        this.registrations,
        this.teamPlayers
      );
      if (!addedCount) {
        this.message = 'Everyone confirmed is already on a team.';
        return;
      }
      this.teamPlayers = rosters.map((roster) => roster.map((player) => {
        const known = this.held[player.ID] || player;
        const entry = this.registrations.find((item) => item.PlayerID === player.ID);
        const next = copyPlayer(known, entry && entry.Position);
        this.remember(next);
        return next;
      }));
      this.message = '';
    },
    clearTeams() {
      this.teamPlayers.flat().forEach((player) => this.remember(player));
      this.teamPlayers = [[], []];
      this.pickedId = null;
    },
    pick(player) {
      this.pickedId = this.pickedId === player.ID ? null : player.ID;
    },
    assign(dest) {
      const player = this.picked;
      if (!player) return;
      this.remember(player);
      this.teamPlayers = this.teamPlayers.map((roster) => roster.filter((item) => item.ID !== player.ID));
      if (dest === 0 || dest === 1) {
        const next = this.materialize(player);
        this.teamPlayers[dest].push(next);
        this.remember(next);
      }
      this.pickedId = null;
    },
    async toggleGuests() {
      this.addingGuest = !this.addingGuest;
      if (this.addingGuest && !this.members.length && this.groupId) {
        const rows = await getGroupMembers(this.groupId).catch(() => []);
        this.members = Array.isArray(rows) ? rows : [];
      }
    },
    addGuest(member) {
      const guest = { ID: member.id, Name: member.name, Position: null };
      this.guests.push(guest);
      this.remember(copyPlayer(guest));
      this.addingGuest = false;
    },
    changeGoals(player, delta) {
      const next = (player.GoalNumber || 0) + delta;
      if (next < 0) return;
      player.GoalNumber = next;
    },
    changeOwnGoals(player, delta) {
      const next = (player.OwnGoals || 0) + delta;
      if (next < 0) return;
      player.OwnGoals = next;
    },
    openOwnGoal(index) {
      this.pickingFor = this.pickingFor === index ? null : index;
    },
    pickOwnGoal(player) {
      player.OwnGoals = (player.OwnGoals || 0) + 1;
      this.pickingFor = null;
    },
    payload() {
      return {
        ...this.match,
        Teams: this.draftTeams.map((team) => ({
          ...team,
          Players: team.Players.map((player) => ({
            ID: player.ID,
            Name: player.Name,
            GoalNumber: player.GoalNumber || 0,
            OwnGoals: player.OwnGoals || 0,
          })),
        })),
      };
    },
    async saveDraft() {
      if (this.step !== 'signups' && !this.bothSides) return;
      this.saving = true;
      this.message = '';
      try {
        await updateMatch(this.match.ID, this.payload());
        this.hadScore = this.hadScore || this.teamPlayers.flat().some((player) => player.GoalNumber || player.OwnGoals);
        this.message = this.step === 'score' ? 'Changes saved.' : 'Saved.';
      } catch (error) {
        console.error(error);
        this.message = 'Could not save the match.';
      } finally {
        this.saving = false;
      }
    },
    confirmDelete() {
      if (!window.confirm('Delete this match? This cannot be undone.')) return;
      this.remove();
    },
    async remove() {
      try {
        await deleteMatch(this.match.ID, this.groupId || this.match.GroupID);
        this.$router.push('/');
      } catch (error) {
        console.error(error);
        this.message = 'Could not delete the match.';
      }
    },
  },
};
</script>
