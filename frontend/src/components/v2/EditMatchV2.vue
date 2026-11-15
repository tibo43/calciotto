<template>
  <main class="page edit-page has-sticky">
    <router-link class="back" :to="`/matches/${id}`">
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M10 3 5 8l5 5"/></svg>
      Back to the match
    </router-link>

    <section v-if="loading" class="state"><div class="skel lg"></div><div class="skel md"></div></section>
    <section v-else-if="failed" class="state">
      <h1>Couldn't load the match</h1>
      <button class="btn" type="button" @click="load">Try again</button>
    </section>

    <template v-else>
      <p class="label">Editing · admin</p>
      <h1 class="display h-kick">{{ heading }}</h1>
      <p class="reg-status">
        <template v-if="!isScheduled">No sign-up list</template>
        <template v-else>Sign-ups <strong>{{ registrationWord }}</strong> · <template v-if="step === 'score' && bothSides">teams made</template><template v-else><strong>{{ confirmed.length }} confirmed / {{ cap }} places</strong> · {{ waitingLabel }}</template></template>
      </p>

      <div class="share-wa">
        <a class="btn btn-ghost" :href="shareUrl" target="_blank" rel="noopener">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M5 19l1.2-4A8 8 0 1 1 12 20a8 8 0 0 1-3.5-.8L5 19z"/></svg>
          Share on WhatsApp
        </a>
        <p class="share-sub">{{ shareSub }}</p>
      </div>

      <!-- The only way between the steps. The one to do next is orange,
           the others can be reopened to correct them, and one that isn't
           possible yet stays in place, greyed and inert. -->
      <nav class="steps" aria-label="Steps">
        <button v-for="item in stepItems" :key="item.key" class="step-i"
          :class="{ 'is-on': step === item.key, 'is-todo': item.todo, 'is-inactive': item.inactive }" type="button"
          :aria-current="step === item.key ? 'step' : null" :aria-disabled="item.inactive ? 'true' : null"
          @click="item.inactive || goStep(item.key)"><b>{{ item.n }}</b> {{ item.label }}</button>
      </nav>

      <template v-if="step === 'signups'">
        <section class="block">
          <div class="sec-head"><h2>1. Sign-ups</h2></div>
          <p class="note">{{ signupNote }}</p>
          <!-- One row from 768px: the sign-up buttons, then the places. -->
          <div class="signup-controls">
            <div class="actions" v-if="canClose || canReopen">
              <button v-if="canClose" class="btn btn-ghost" type="button" @click="closeList">Close sign-ups</button>
              <button v-if="canReopen" class="btn btn-ghost" type="button" aria-label="Reopen sign-ups" @click="reopenList"><span class="when-roomy">Reopen sign-ups</span><span class="when-narrow" aria-hidden="true">Reopen</span></button>
            </div>
            <div v-if="isScheduled" class="places-row">
              <label class="label" for="confirmed-places">Places</label>
              <input id="confirmed-places" type="number" aria-label="Number of places" min="1" v-model="maxPlayersInput">
              <button class="btn btn-ghost" type="button" :disabled="!canUpdateCap" @click="updateCap">Update</button>
            </div>
          </div>
          <p class="label">Confirmed</p>
          <div v-for="entry in confirmed" :key="entry.PlayerID" class="sign-row">
            <div class="who">{{ entry.Name }}</div>
          </div>
          <p v-if="!confirmed.length" class="note">No confirmed players.</p>
          <p class="label">Waiting list</p>
          <div v-for="entry in waiting" :key="entry.PlayerID" class="sign-row">
            <div class="who">{{ entry.Name }}</div>
          </div>
          <p v-if="!waiting.length" class="note">No one is waiting.</p>
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
          <section class="compose-col" :class="{ 'is-drop': pickedId && teamOf(pickedId) !== null }" aria-label="Reserve">
            <div class="col-head"><h2>Reserve</h2></div>
            <p class="col-count">{{ reserve.length }}</p>
            <p v-if="!reserve.length" class="col-empty">Every confirmed player is placed.</p>
            <template v-for="player in reserve" :key="player.ID">
              <button class="p-row" :class="{ 'is-picked': pickedId === player.ID }" type="button" :data-player="player.ID"
                aria-haspopup="menu" :aria-expanded="pickedId === player.ID ? 'true' : 'false'" @click="pick(player)">
                <span class="pos">{{ posLabel(player) }}</span>
                <span class="nm">{{ player.Name }}</span>
              </button>
              <div v-if="pickedId === player.ID" class="move-menu" role="menu" :aria-label="`Move ${player.Name} to`" @keydown="onMenuKey">
                <button v-for="dest in destinations" :key="dest.key" class="move-opt" role="menuitem" type="button" @click="assign(dest.index)">{{ dest.label }}</button>
              </div>
            </template>
          </section>
          <section v-for="index in [0, 1]" :key="teamId(index)" class="compose-col" :class="{ 'is-drop': pickedId && teamOf(pickedId) !== index }" :aria-label="teamName(index)">
            <div class="col-head">
              
              <h2>{{ teamName(index) }}</h2>
            </div>
            <p class="col-count">{{ teamPlayers[index].length }}</p>
            <p v-if="!teamPlayers[index].length" class="col-empty">No players yet.</p>
            <template v-for="player in teamPlayers[index]" :key="player.ID">
              <button class="p-row" :class="{ 'is-picked': pickedId === player.ID }" type="button" :data-player="player.ID"
                aria-haspopup="menu" :aria-expanded="pickedId === player.ID ? 'true' : 'false'" @click="pick(player)">
                <span class="pos">{{ posLabel(player) }}</span>
                <span class="nm">{{ player.Name }}</span>
              </button>
              <div v-if="pickedId === player.ID" class="move-menu" role="menu" :aria-label="`Move ${player.Name} to`" @keydown="onMenuKey">
                <button v-for="dest in destinations" :key="dest.key" class="move-opt" role="menuitem" type="button" @click="assign(dest.index)">{{ dest.label }}</button>
              </div>
            </template>
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
      </template>

      <template v-else>
        <template v-if="!bothSides">
          <p class="note">The score opens once each team has at least one player.</p>
        </template>
        <template v-else>
          <section class="score-board" aria-label="Computed score">
            <div class="board-teams">
              <div class="board-team"><span class="name">{{ teamName(0) }}</span></div>
              <div class="board-score">{{ sides[0].score }}<span class="sep">–</span>{{ sides[1].score }}</div>
              <div class="board-team"><span class="name">{{ teamName(1) }}</span></div>
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
              <h2>{{ teamName(index) }}</h2>
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

    <AppToast ref="toast" />
  </main>
</template>

<script>
import {
  getMatchRegistrations, closeMatchRegistrations, reopenMatchRegistrations,
  setMatchMaxPlayers, updateMatch, deleteMatch, getInviteCode, getGroupMembers, getSeasons,
} from '@/services/api';
import { setActiveGroupId } from '@/services/activeGroup';
import { findMatchInMyGroups, actionErrorMessage } from '@/services/v2Match';
import AppToast from '@/components/AppToast.vue';
import {
  deriveRegistrationState, REGISTRATION_OPEN, REGISTRATION_CLOSED_BY_ADMIN, REGISTRATION_CLOSED_AT_KICKOFF,
  fillTeamsFromRegistrations,
} from '@/services/matchRegistration';
import { formatDateTimeForDisplay, formatCalendarDayLong } from '@/services/datetime';
import { heroDateLabel, kickoffLabel } from '@/services/v2Home';
import { encodeMatchId } from '@/services/shortLink';
import { buildWhatsAppShareText, buildWhatsAppShareUrl, buildTeamsShareText } from '@/services/whatsappShare';
import { scoreSides, teamScoreLine, sizeCounter, recapEntries, lineupsReady, shareSubtitle } from '@/services/v2Edit';

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
  components: { AppToast },
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
    // The status line is built in the template, not as an HTML string: no
    // v-html anywhere in the app keeps player-supplied text from ever being
    // rendered as markup.
    registrationWord() {
      if (this.registrationState === REGISTRATION_OPEN) return 'open';
      return this.registrationState === 'not-open-yet' ? 'not open yet' : 'closed';
    },
    waitingLabel() {
      return this.waiting.length === 1 ? '1 waiting' : `${this.waiting.length} waiting`;
    },
    signupNote() {
      if (this.bothSides) return 'The teams are made. Sign-ups above stay as they are.';
      const names = this.waiting.map((entry) => entry.Name).join(', ');
      const bench = names ? `, ${this.waiting.length} waiting (${names})` : '';
      return `Not composed yet. ${this.confirmed.length} confirmed players are in the reserve${bench}. The waiting list is not split automatically.`;
    },
    // Where the match stands, from what the server has saved (not this
    // page's draft): sign-ups closed, then teams saved, then the score.
    progress() {
      const match = this.match || {};
      const saved = match.Teams || [];
      const scoreSaved = Boolean(match.ResultRecordedAt)
        || saved.some((team) => (team.Players || []).some((player) => player.GoalNumber || player.OwnGoals));
      const teamsSaved = [0, 1].every((index) => ((saved[index] || {}).Players || []).length > 0);
      const closed = !this.isScheduled || this.registrationState === REGISTRATION_CLOSED_BY_ADMIN
        || this.registrationState === REGISTRATION_CLOSED_AT_KICKOFF;
      if (scoreSaved) return null;
      if (teamsSaved) return 'score';
      return closed ? 'teams' : null;
    },
    stepItems() {
      return [
        { key: 'signups', n: 1, label: 'Sign-ups' },
        { key: 'teams', n: 2, label: 'Teams' },
        { key: 'score', n: 3, label: 'Score and goals', inactive: !this.bothSides },
      ].map((item) => ({ ...item, todo: this.progress === item.key && !item.inactive }));
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
    // Where the picked player can go: the two teams and the reserve, minus
    // where they already are. All alike; none is a default.
    destinations() {
      if (!this.picked) return [];
      const here = this.teamOf(this.picked.ID);
      return [
        { key: 'team-0', index: 0, label: this.teamName(0) },
        { key: 'team-1', index: 1, label: this.teamName(1) },
        { key: 'reserve', index: null, label: 'Reserve' },
      ].filter((dest) => dest.index !== here);
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
        // In the group the match belongs to, not the caller's first one: an
        // admin of several groups edits matches in all of them.
        const found = await findMatchInMyGroups(this.id);
        this.match = found.details;
        const regs = await getMatchRegistrations(this.id).catch(() => []);
        this.registrations = Array.isArray(regs) ? regs : [];
        this.guests = [];
        this.held = {};
        this.initDraft();
        try {
          const { groups, group } = found;
          this.groupId = group.id;
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
    // Action outcomes go to the shared toast, like everywhere else in the
    // app; a refusal shows the backend's own reason.
    async closeList() {
      try {
        await closeMatchRegistrations(this.match.ID);
        this.match = { ...this.match, RegistrationsClosedAt: new Date().toISOString() };
        this.$refs.toast.show('Sign-ups closed.', 'success');
      } catch (error) {
        console.error('Error closing sign-ups:', error);
        this.$refs.toast.show(actionErrorMessage(error, 'Could not close sign-ups.'), 'error');
      }
    },
    async reopenList() {
      try {
        await reopenMatchRegistrations(this.match.ID);
        this.match = { ...this.match, RegistrationsClosedAt: null };
        this.$refs.toast.show('Sign-ups reopened.', 'success');
      } catch (error) {
        console.error('Error reopening sign-ups:', error);
        this.$refs.toast.show(actionErrorMessage(error, 'Could not reopen sign-ups.'), 'error');
      }
    },
    async updateCap() {
      if (!this.canUpdateCap) return;
      const requested = Number(this.maxPlayersInput);
      try {
        const entries = await setMatchMaxPlayers(this.match.ID, requested);
        this.registrations = Array.isArray(entries) ? entries : this.registrations;
        this.match = { ...this.match, MaxPlayers: requested };
        this.$refs.toast.show(`Places set to ${requested}.`, 'success');
      } catch (error) {
        console.error('Error updating the places:', error);
        this.$refs.toast.show(actionErrorMessage(error, 'Could not change the places.'), 'error');
      }
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
    // A click, a tap or Enter on a player opens their menu (or closes it
    // again) and moves the focus onto its first choice.
    pick(player) {
      this.pickedId = this.pickedId === player.ID ? null : player.ID;
      if (this.pickedId) {
        this.$nextTick(() => {
          const first = this.$el.querySelector('.move-menu .move-opt');
          if (first) first.focus();
        });
      }
    },
    // Up/Down (and Home/End) walk the choices, Enter picks one, Escape
    // closes the menu and gives the focus back to the player.
    onMenuKey(event) {
      const items = Array.from(event.currentTarget.querySelectorAll('.move-opt'));
      const at = items.indexOf(document.activeElement);
      let next = null;
      if (event.key === 'ArrowDown') next = (at + 1) % items.length;
      else if (event.key === 'ArrowUp') next = (at - 1 + items.length) % items.length;
      else if (event.key === 'Home') next = 0;
      else if (event.key === 'End') next = items.length - 1;
      else if (event.key === 'Escape') {
        event.preventDefault();
        const row = event.currentTarget.previousElementSibling;
        this.pickedId = null;
        if (row) row.focus();
        return;
      }
      if (next === null) return;
      event.preventDefault();
      items[next].focus();
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
      // Back on the player, in their new place.
      this.$nextTick(() => {
        const row = this.$el.querySelector(`.p-row[data-player="${player.ID}"]`);
        if (row) row.focus();
      });
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
        this.$refs.toast.show(this.step === 'score' ? 'Changes saved.' : 'Saved.', 'success');
      } catch (error) {
        console.error(error);
        this.$refs.toast.show(actionErrorMessage(error, 'Could not save the match.'), 'error');
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
        this.$refs.toast.show(actionErrorMessage(error, 'Could not delete the match.'), 'error');
      }
    },
  },
};
</script>
