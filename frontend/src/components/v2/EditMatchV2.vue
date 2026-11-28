<template>
  <main class="page edit-page has-sticky">
    <router-link class="back" :to="`/matches/${id}`">
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M10 3 5 8l5 5"/></svg>
      {{ $t('edit.backToMatch') }}
    </router-link>

    <section v-if="loading" class="state"><div class="skel lg"></div><div class="skel md"></div></section>
    <section v-else-if="failed" class="state">
      <h1>{{ $t('match.loadError') }}</h1>
      <button class="btn" type="button" @click="load">{{ $t('common.tryAgain') }}</button>
    </section>

    <template v-else>
      <p class="label">{{ $t('edit.editing') }}</p>
      <h1 class="display h-kick">{{ heading }}</h1>
      <p class="reg-status">
        <template v-if="!isScheduled">{{ $t('edit.noSignupList') }}</template>
        <template v-else><i18n-t keypath="edit.regStatus" scope="global"><template #state><strong>{{ registrationWord }}</strong></template><template #detail><template v-if="step === 'score' && bothSides">{{ $t('edit.teamsMade') }}</template><i18n-t v-else keypath="edit.regCounts" scope="global"><template #counts><strong>{{ confirmedPlacesLabel }}</strong></template><template #waiting>{{ waitingLabel }}</template></i18n-t></template></i18n-t></template>
      </p>

      <div class="match-sharing">
        <ShareButton class="btn btn-ghost" :text="shareText" :label="shareLabel" icon />
        <p class="share-sub">{{ shareSub }}</p>
      </div>

      <!-- The only way between the steps. The one to do next is orange,
           the others can be reopened to correct them, and one that isn't
           possible yet stays in place, greyed and inert. -->
      <nav class="steps" :aria-label="$t('edit.steps.label')">
        <button v-for="item in stepItems" :key="item.key" class="step-i"
          :class="{ 'is-on': step === item.key, 'is-todo': item.todo, 'is-inactive': item.inactive }" type="button"
          :aria-current="step === item.key ? 'step' : null" :aria-disabled="item.inactive ? 'true' : null"
          @click="item.inactive || goStep(item.key)"><b>{{ item.n }}</b> {{ item.label }}</button>
      </nav>

      <template v-if="step === 'signups'">
        <section class="block">
          <div class="sec-head"><h2>{{ $t('edit.signups.title') }}</h2></div>
          <p class="note">{{ signupNote }}</p>
          <!-- One row from 768px: the sign-up buttons, then the places. -->
          <div class="signup-controls">
            <div class="actions" v-if="canClose || canReopen">
              <button v-if="canClose" class="btn btn-ghost" type="button" @click="closeList">{{ $t('edit.signups.close') }}</button>
              <button v-if="canReopen" class="btn btn-ghost" type="button" :aria-label="$t('edit.signups.reopen')" @click="reopenList"><span class="when-roomy">{{ $t('edit.signups.reopen') }}</span><span class="when-narrow" aria-hidden="true">{{ $t('edit.signups.reopenShort') }}</span></button>
            </div>
            <div v-if="isScheduled" class="places-row">
              <label class="label" for="confirmed-places">{{ $t('edit.signups.places') }}</label>
              <input id="confirmed-places" type="number" :aria-label="$t('edit.signups.placesAria')" min="1" v-model="maxPlayersInput">
              <button class="btn btn-ghost" type="button" :disabled="!canUpdateCap" @click="updateCap">{{ $t('edit.signups.update') }}</button>
            </div>
          </div>
          <p class="label">{{ $t('edit.signups.confirmed') }}</p>
          <div v-for="entry in confirmed" :key="entry.PlayerID" class="sign-row">
            <div class="who">{{ entry.Name }}</div>
          </div>
          <p v-if="!confirmed.length" class="note">{{ $t('edit.signups.noConfirmed') }}</p>
          <p class="label">{{ $t('edit.signups.waitingList') }}</p>
          <div v-for="entry in waiting" :key="entry.PlayerID" class="sign-row">
            <div class="who">{{ entry.Name }}</div>
          </div>
          <p v-if="!waiting.length" class="note">{{ $t('edit.signups.noWaiting') }}</p>
        </section>
        <p v-if="message" class="note">{{ message }}</p>
        <div class="sticky-cta">
          <button class="btn btn-ghost btn-danger" type="button" @click="confirmDelete">{{ $t('edit.delete') }}</button>
          <button class="btn" type="button" :disabled="saving" @click="saveDraft">{{ $t('edit.save') }}</button>
        </div>
      </template>

      <template v-else-if="step === 'teams'">
        <div class="compose-toolbar">
          <button class="btn" type="button" :disabled="!canSplit" @click="split">{{ $t('edit.teams.split') }}</button>
          <button class="btn btn-ghost" type="button" @click="clearTeams">{{ $t('edit.teams.clear') }}</button>
          <p class="hint">{{ splitHint }}</p>
        </div>
        <p class="tap-hint">{{ $t('edit.teams.tapHint') }}</p>
        <p class="size-count">{{ sizeCount }}</p>
        <div class="compose-board">
          <section class="compose-col" :class="{ 'is-drop': pickedId && teamOf(pickedId) !== null }" :aria-label="$t('edit.teams.reserve')">
            <div class="col-head"><h2>{{ $t('edit.teams.reserve') }}</h2></div>
            <p class="col-count">{{ reserve.length }}</p>
            <p v-if="!reserve.length" class="col-empty">{{ $t('edit.teams.allPlaced') }}</p>
            <template v-for="player in reserve" :key="player.ID">
              <button class="p-row" :class="{ 'is-picked': pickedId === player.ID, 'has-tag': tagOf(player) }" type="button" :data-player="player.ID"
                aria-haspopup="menu" :aria-expanded="pickedId === player.ID ? 'true' : 'false'" @click="pick(player)">
                <span class="pos">{{ posLabel(player) }}</span>
                <span class="nm">{{ player.Name }}</span>
                <span v-if="tagOf(player)" class="pos-tag">{{ tagOf(player) }}</span>
              </button>
              <div v-if="pickedId === player.ID" class="move-menu" role="menu" :aria-label="$t('edit.teams.moveTo', { name: player.Name })" @keydown="onMenuKey">
                <button v-for="dest in destinations" :key="dest.key" class="move-opt" role="menuitem" type="button" @click="assign(dest.index)">{{ dest.label }}</button>
              </div>
            </template>
          </section>
          <section v-for="index in [0, 1]" :key="teamId(index)" class="compose-col" :class="{ 'is-drop': pickedId && teamOf(pickedId) !== index }" :aria-label="teamName(index)">
            <div class="col-head">
              
              <h2>{{ teamName(index) }}</h2>
            </div>
            <p class="col-count">{{ teamPlayers[index].length }}</p>
            <p v-if="teamPlayers[index].length" class="pos-summary">{{ positionSummary(teamPlayers[index], positionsById) }}</p>
            <p v-if="!teamPlayers[index].length" class="col-empty">{{ $t('edit.teams.noPlayers') }}</p>
            <template v-for="player in teamPlayers[index]" :key="player.ID">
              <button class="p-row" :class="{ 'is-picked': pickedId === player.ID, 'has-tag': tagOf(player) }" type="button" :data-player="player.ID"
                aria-haspopup="menu" :aria-expanded="pickedId === player.ID ? 'true' : 'false'" @click="pick(player)">
                <span class="pos">{{ posLabel(player) }}</span>
                <span class="nm">{{ player.Name }}</span>
                <span v-if="tagOf(player)" class="pos-tag">{{ tagOf(player) }}</span>
              </button>
              <div v-if="pickedId === player.ID" class="move-menu" role="menu" :aria-label="$t('edit.teams.moveTo', { name: player.Name })" @keydown="onMenuKey">
                <button v-for="dest in destinations" :key="dest.key" class="move-opt" role="menuitem" type="button" @click="assign(dest.index)">{{ dest.label }}</button>
              </div>
            </template>
          </section>
        </div>
        <section class="sec">
          <div class="sec-head"><h2>{{ $t('edit.signups.waitingList') }}</h2></div>
          <p class="note">{{ $t('edit.teams.waitingNote') }}</p>
          <div v-for="entry in waiting" :key="entry.PlayerID" class="wait-row">
            <div>
              <div class="who">{{ entry.Name }}</div>
              <div class="meta">{{ $t('edit.teams.waitingMeta', { n: entry.Position }) }}</div>
            </div>
          </div>
          <div class="actions">
            <button class="btn btn-ghost" type="button" @click="toggleGuests">{{ $t('edit.teams.addGuest') }}</button>
          </div>
          <div v-if="addingGuest" class="guest-list">
            <button v-for="member in addable" :key="member.id" class="csc-pick" type="button" @click="addGuest(member)">
              <span class="nm">{{ member.name }}</span>
            </button>
            <p v-if="!addable.length" class="note">{{ $t('edit.teams.everyoneHere') }}</p>
          </div>
        </section>
        <p v-if="message" class="note">{{ message }}</p>
        <div class="sticky-cta">
          <p v-if="!bothSides" class="cta-msg">{{ $t('edit.teams.needBoth') }}</p>
          <button class="btn" :class="{ 'is-off': !bothSides }" type="button" :disabled="!bothSides || saving" @click="saveDraft">{{ $t('edit.teams.confirm') }}</button>
        </div>
      </template>

      <template v-else>
        <template v-if="!bothSides">
          <p class="note">{{ $t('edit.score.opensLater') }}</p>
        </template>
        <template v-else>
          <section class="score-board" :aria-label="$t('edit.score.computed')">
            <div v-fit-two-lines class="board-teams">
              <div class="board-team"><span class="name">{{ teamName(0) }}</span></div>
              <div class="board-score">{{ sides[0].score }}<span class="sep">–</span>{{ sides[1].score }}</div>
              <div class="board-team"><span class="name">{{ teamName(1) }}</span></div>
            </div>
            <p class="balance">{{ teamName(0) }} {{ teamPlayers[0].length }} · {{ teamName(1) }} {{ teamPlayers[1].length }}</p>
            <p class="score-status">{{ scoreStatus }}</p>
            <ul class="scorers">
              <li v-for="(line, index) in recap" :key="index" :class="{ away: line.away }">{{ line.text }}</li>
            </ul>
            <p class="score-hint">{{ $t('edit.score.hint') }}</p>
          </section>
          <div class="score-teams">
            <section v-for="index in [0, 1]" :key="teamId(index)" class="score-team" :aria-label="$t('edit.score.goalsFor', { team: teamName(index) })">
              <h2>{{ teamName(index) }}</h2>
              <p class="team-sub">{{ teamScoreLine(sides[index]) }}</p>
              <div v-for="player in teamPlayers[index]" :key="player.ID" class="score-row">
                <div class="who">
                  <span class="nm">{{ player.Name }}</span>
                  <span v-if="player.OwnGoals" class="og-plain">{{ $t('match.ownGoals', player.OwnGoals) }}</span>
                </div>
                <div class="stepper">
                  <button type="button" :disabled="!player.GoalNumber" :aria-label="$t('edit.score.removeGoal', { name: player.Name })" @click="changeGoals(player, -1)">−</button>
                  <span class="n">{{ player.GoalNumber || 0 }}</span>
                  <button type="button" :aria-label="$t('edit.score.addGoal', { name: player.Name })" @click="changeGoals(player, 1)">+</button>
                </div>
              </div>
              <div v-for="player in ownGoalsFor(index)" :key="`og-${player.ID}`" class="score-row">
                <div class="who">
                  <span class="nm">{{ player.Name }}</span>
                  <span class="og-plain">{{ $t('edit.score.og') }}</span>
                </div>
                <div class="stepper">
                  <button type="button" :aria-label="$t('edit.score.removeOwnGoal', { name: player.Name })" @click="changeOwnGoals(player, -1)">−</button>
                  <span class="n">{{ player.OwnGoals }}</span>
                  <button type="button" :aria-label="$t('edit.score.addOwnGoal', { name: player.Name })" @click="changeOwnGoals(player, 1)">+</button>
                </div>
              </div>
              <div class="csc-add-wrap" :class="{ 'is-open': pickingFor === index }">
                <button class="btn btn-ghost btn-csc-add" type="button" @click="openOwnGoal(index)">{{ $t('edit.score.ownGoalAdd') }}</button>
                <div v-if="pickingFor === index" class="csc-pop" role="dialog" aria-labelledby="og-pop-title">
                  <h3 id="og-pop-title">{{ $t('edit.score.ownGoal') }}</h3>
                  <p class="ref">{{ ownGoalHelp(index) }}</p>
                  <button v-for="player in teamPlayers[1 - index]" :key="player.ID" class="csc-pick" type="button" @click="pickOwnGoal(player)">
                    <span class="nm">{{ player.Name }}</span>
                  </button>
                  <button class="btn btn-ghost" type="button" @click="pickingFor = null">{{ $t('common.cancel') }}</button>
                </div>
              </div>
            </section>
          </div>
          <div v-if="pickingFor !== null" class="csc-scrim" @click="pickingFor = null"></div>
          <div v-if="pickingFor !== null" class="csc-sheet" role="dialog" aria-labelledby="og-sheet-title">
            <h3 id="og-sheet-title">{{ $t('edit.score.ownGoal') }}</h3>
            <p class="ref">{{ ownGoalHelp(pickingFor) }}</p>
            <button v-for="player in teamPlayers[1 - pickingFor]" :key="player.ID" class="csc-pick" type="button" @click="pickOwnGoal(player)">
              <span class="nm">{{ player.Name }}</span>
            </button>
            <button class="btn btn-ghost" type="button" @click="pickingFor = null">{{ $t('common.cancel') }}</button>
          </div>
          <p v-if="message" class="note">{{ message }}</p>
          <div class="sticky-cta">
            <button class="btn" type="button" :disabled="saving" @click="saveDraft">{{ $t('edit.score.saveChanges') }}</button>
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
  setMatchMaxPlayers, updateMatch, deleteMatch, getInviteCode, getGroupMembers, getSeasons, getMatchPositions,
} from '@/services/api';
import { setActiveGroupId } from '@/services/activeGroup';
import { findMatchInMyGroups, actionErrorMessage } from '@/services/v2Match';
import AppToast from '@/components/AppToast.vue';
import {
  deriveRegistrationState, REGISTRATION_OPEN, REGISTRATION_CLOSED_BY_ADMIN, REGISTRATION_CLOSED_AT_KICKOFF,
} from '@/services/matchRegistration';
import { splitByPositions, positionsMap, positionsTag, positionSummary } from '@/services/v2Positions';
import { formatDateTimeForDisplay, formatCalendarDayLong } from '@/services/datetime';
import { heroDateLabel, kickoffLabel } from '@/services/v2Home';
import { encodeMatchId } from '@/services/shortLink';
import { buildMatchShareText, buildTeamsShareText } from '@/services/shareMessages';
import ShareButton from '@/components/ShareButton.vue';
import { scoreSides, teamScoreLine, sizeCounter, recapEntries, lineupsReady, shareSubtitle } from '@/services/v2Edit';
import fitTwoLines from '@/directives/fitTwoLines';

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
  directives: { fitTwoLines },
  components: { AppToast, ShareButton },
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
      positionsById: {},
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
      return time ? this.$t('edit.heading', { date, time }) : date;
    },
    // The status line is built in the template, not as an HTML string: no
    // v-html anywhere in the app keeps player-supplied text from ever being
    // rendered as markup.
    registrationWord() {
      if (this.registrationState === REGISTRATION_OPEN) return this.$t('edit.state.open');
      return this.registrationState === 'not-open-yet' ? this.$t('edit.state.notOpenYet') : this.$t('edit.state.closed');
    },
    waitingLabel() {
      return this.$t('edit.waiting', this.waiting.length);
    },
    // « 11 confirmed / 12 places »: each count agrees with its own noun.
    confirmedPlacesLabel() {
      return this.$t('edit.confirmedPlaces', {
        confirmed: this.$t('edit.confirmedCount', this.confirmed.length),
        cap: this.$t('edit.placesCount', this.cap),
      });
    },
    signupNote() {
      if (this.bothSides) return this.$t('edit.signups.teamsMadeNote');
      const names = this.waiting.map((entry) => entry.Name).join(', ');
      const bench = names ? this.$t('edit.signups.bench', { n: this.waiting.length, names }) : '';
      return this.$t('edit.signups.notComposed', { confirmed: this.confirmed.length, bench }, { plural: this.confirmed.length });
    },
    // Where the match stands, from what the server has saved (not this
    // page's draft): sign-ups closed, then teams saved, then the score.
    signupsClosed() {
      return !this.isScheduled || this.registrationState === REGISTRATION_CLOSED_BY_ADMIN
        || this.registrationState === REGISTRATION_CLOSED_AT_KICKOFF;
    },
    teamsSaved() {
      const saved = (this.match && this.match.Teams) || [];
      return [0, 1].every((index) => ((saved[index] || {}).Players || []).length > 0);
    },
    progress() {
      const match = this.match || {};
      const saved = match.Teams || [];
      const scoreSaved = Boolean(match.ResultRecordedAt)
        || saved.some((team) => (team.Players || []).some((player) => player.GoalNumber || player.OwnGoals));
      if (scoreSaved) return null;
      if (this.teamsSaved) return 'score';
      return this.signupsClosed ? 'teams' : null;
    },
    // The same rule as the next step's colour: Teams opens once sign-ups are
    // closed, Score and goals once the teams are saved. Until then a step
    // stays in place, greyed and inert.
    stepItems() {
      return [
        { key: 'signups', n: 1, label: this.$t('edit.steps.signups') },
        { key: 'teams', n: 2, label: this.$t('edit.steps.teams'), inactive: !this.signupsClosed },
        { key: 'score', n: 3, label: this.$t('edit.steps.score'), inactive: !this.signupsClosed || !this.teamsSaved },
      ].map((item) => ({ ...item, todo: this.progress === item.key && !item.inactive }));
    },
    sizeCount() {
      return sizeCounter(this.teamPlayers[0].length, this.teamPlayers[1].length);
    },
    splitHint() {
      return this.$t('edit.teams.splitHint');
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
        { key: 'reserve', index: null, label: this.$t('edit.teams.reserve') },
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
    shareLabel() {
      return lineupsReady(this.draftTeams) ? this.$t('edit.share.teams') : this.$t('edit.share.match');
    },
    shareText() {
      if (!this.match) return '';
      const when = this.isScheduled
        ? formatDateTimeForDisplay(this.match.ScheduledAt)
        : formatCalendarDayLong(this.match.Date);
      if (lineupsReady(this.draftTeams)) {
        const teams = this.draftTeams.map((team) => ({
          name: team.Name,
          players: (team.Players || []).map((player) => player.Name),
        }));
        return buildTeamsShareText({ dateTimeLabel: when, teams });
      }
      return buildMatchShareText({
        kickoffLabel: when,
        matchUrl: `${window.location.origin}/m/${encodeMatchId(this.match.ID)}`,
        groupInviteCode: this.inviteCode,
        signupsOpen: this.registrationState === REGISTRATION_OPEN,
      });
    },
    scoreStatus() {
      return this.hadScore ? this.$t('edit.score.saved') : this.$t('edit.score.notSaved');
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
    // A link to a step that isn't open yet shows the last one that is.
    stepItems(items) {
      if (!this.match) return;
      const current = items.find((item) => item.key === this.step);
      if (current && current.inactive) {
        const open = items.filter((item) => !item.inactive);
        this.goStep(open[open.length - 1].key);
      }
    },
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
    positionSummary,
    tagOf(player) {
      return positionsTag(this.positionsById[player.ID]);
    },
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
      return (team && team.Name) || this.$t('edit.team');
    },
    teamId(index) {
      const team = this.match && this.match.Teams && this.match.Teams[index];
      return (team && team.ID) || index;
    },
    posLabel(player) {
      return player.Position ? this.$t('home.position', { n: player.Position }) : this.$t('edit.teams.guest');
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
      return this.$t('edit.score.ownGoalHelp', { team: this.teamName(beneficiary), other: this.teamName(1 - beneficiary) });
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
        // This page is admin-only, and so are the preferred positions. Without
        // them (an older backend, a failed request) the split is still random.
        this.positionsById = positionsMap(await getMatchPositions(this.id).catch(() => []));
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
        } catch {
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
        this.$refs.toast.show(this.$t('edit.toast.closed'), 'success');
      } catch (error) {
        console.error('Error closing sign-ups:', error);
        this.$refs.toast.show(actionErrorMessage(error, this.$t('edit.toast.closeError')), 'error');
      }
    },
    async reopenList() {
      try {
        await reopenMatchRegistrations(this.match.ID);
        this.match = { ...this.match, RegistrationsClosedAt: null };
        this.$refs.toast.show(this.$t('edit.toast.reopened'), 'success');
      } catch (error) {
        console.error('Error reopening sign-ups:', error);
        this.$refs.toast.show(actionErrorMessage(error, this.$t('edit.toast.reopenError')), 'error');
      }
    },
    async updateCap() {
      if (!this.canUpdateCap) return;
      const requested = Number(this.maxPlayersInput);
      try {
        const entries = await setMatchMaxPlayers(this.match.ID, requested);
        this.registrations = Array.isArray(entries) ? entries : this.registrations;
        this.match = { ...this.match, MaxPlayers: requested };
        this.$refs.toast.show(this.$t('edit.toast.placesSet'), 'success');
      } catch (error) {
        console.error('Error updating the places:', error);
        this.$refs.toast.show(actionErrorMessage(error, this.$t('edit.toast.placesError')), 'error');
      }
    },
    split() {
      const { rosters, addedCount } = splitByPositions(
        this.registrations,
        this.teamPlayers,
        this.positionsById,
      );
      if (!addedCount) {
        this.message = this.$t('edit.toast.allOnTeam');
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
        // What was just saved is what the server now holds: the steps read it.
        this.match = {
          ...this.match,
          Teams: (this.match.Teams || []).map((team, index) => (index < 2
            ? { ...team, Players: this.teamPlayers[index].map((player) => ({ ...player })) }
            : team)),
        };
        this.hadScore = this.hadScore || this.teamPlayers.flat().some((player) => player.GoalNumber || player.OwnGoals);
        this.$refs.toast.show(this.step === 'score' ? this.$t('edit.toast.changesSaved') : this.$t('edit.toast.saved'), 'success');
      } catch (error) {
        console.error(error);
        this.$refs.toast.show(actionErrorMessage(error, this.$t('edit.toast.saveError')), 'error');
      } finally {
        this.saving = false;
      }
    },
    confirmDelete() {
      if (!window.confirm(this.$t('edit.toast.deleteConfirm'))) return;
      this.remove();
    },
    async remove() {
      try {
        await deleteMatch(this.match.ID, this.groupId || this.match.GroupID);
        this.$router.push('/');
      } catch (error) {
        console.error(error);
        this.$refs.toast.show(actionErrorMessage(error, this.$t('edit.toast.deleteError')), 'error');
      }
    },
  },
};
</script>
