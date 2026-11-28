<template>
  <main class="page">
    <nav v-if="!isEmptyGroup" class="subnav" :aria-label="$t('predictions.label')">
      <router-link to="/predictions" :class="{ 'is-on': tab === 'matches' }">
        {{ $t('predictions.tabs.matches') }}
      </router-link>
      <router-link to="/predictions?tab=leaderboard" :class="{ 'is-on': tab === 'leaderboard' }">
        {{ $t('predictions.tabs.leaderboard') }}
      </router-link>
    </nav>

    <template v-if="tab === 'matches'">
      <section v-if="loading" class="preds">
        <div v-for="n in 3" :key="n" class="pred">
          <div class="skel md"></div>
          <div class="skel lg"></div>
        </div>
      </section>
      <section v-else-if="failed" class="state">
        <p class="label">{{ $t('common.error') }}</p>
        <h1>{{ $t('predictions.loadError') }}</h1>
        <p>{{ $t('predictions.loadErrorText') }}</p>
        <button class="btn" type="button" @click="load">{{ $t('common.tryAgain') }}</button>
      </section>
      <section v-else-if="isEmptyGroup" class="state">
        <p class="label">{{ $t('predictions.emptyGroup.label') }}</p>
        <h1>{{ $t('predictions.emptyGroup.title') }}</h1>
        <p>{{ $t('predictions.emptyGroup.text') }}</p>
      </section>
      <section v-else-if="eligible.length === 0" class="state">
        <h1>{{ $t('predictions.empty.title') }}</h1>
        <p>{{ $t('predictions.empty.matches') }}</p>
      </section>
      <template v-else>
        <article v-for="match in eligible" :key="match.ID" class="pred">
          <div class="pred-head">
            <div class="pred-title">
              <span class="team">{{ teamName(match, 0) }}</span>
              <span v-if="showScore(match)" class="pred-score">{{ scoreText(match) }}</span>
              <span v-else class="pred-vs">{{ ' \u00b7 ' }}</span>
              <span class="team">{{ teamName(match, 1) }}</span>
            </div>
            <div class="pred-when">{{ whenLabel(match) }}</div>
          </div>
          <p v-if="note(match)" class="note">{{ note(match) }}</p>
          <div class="picks">
            <button v-for="option in options(match)" :key="option.key" class="pick" type="button"
              :class="pickClass(match, option.teamId)"
              :aria-disabled="(!open(match)).toString()"
              @click="choose(match, option.teamId)">
              <span class="who">{{ option.label }}</span>
              <span v-if="option.odds != null" class="odd">{{ oddsLabel(option.odds) }}</span>
            </button>
          </div>
          <p v-if="resultLine(match)" class="pred-state" :class="resultKind(match)">{{ resultLine(match) }}</p>
          <!-- The line-ups live on the match page rather than folded in here. -->
          <router-link class="pred-teams" :to="{ name: 'MatchView', params: { id: match.ID } }">{{ $t('predictions.seeTeams') }}</router-link>
        </article>
      </template>
    </template>

    <template v-else>
      <section v-if="boardLoading" class="state">
        <div class="skel md"></div>
        <div class="skel"></div>
      </section>
      <section v-else-if="boardFailed" class="state">
        <p class="label">{{ $t('common.error') }}</p>
        <h1>{{ $t('predictions.loadError') }}</h1>
        <p>{{ $t('predictions.loadErrorText') }}</p>
        <button class="btn" type="button" @click="loadBoard">{{ $t('common.tryAgain') }}</button>
      </section>
      <section v-else-if="isEmptyGroup" class="state">
        <p class="label">{{ $t('predictions.emptyGroup.label') }}</p>
        <h1>{{ $t('predictions.emptyGroup.title') }}</h1>
        <p>{{ $t('predictions.emptyGroup.text') }}</p>
      </section>
      <section v-else-if="standings.length === 0" class="state">
        <h1>{{ $t('predictions.empty.title') }}</h1>
        <p>{{ $t('predictions.empty.leaderboard') }}</p>
      </section>
      <template v-else>
        <div class="table-wrap desk-only">
          <table class="stand">
            <thead>
              <tr><th>#</th><th>{{ $t('rankings.cols.player') }}</th><th>{{ $t('predictions.cols.last5') }}</th><th>{{ $t('rankings.cols.pts') }}</th></tr>
            </thead>
            <tbody>
              <tr v-for="(row, index) in standings" :key="row.PlayerID" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
                <td class="rk">{{ index + 1 }}</td>
                <td>{{ displayName(row) }}<span v-if="left(row)" class="left-note">{{ ' ' + $t('rankings.leftGroup') }}</span></td>
                <td><span class="form"><span class="form-k">{{ $t('predictions.cols.last5') }}</span><span v-for="(slot, i) in lastFive(row)" :key="i" class="form-cell" :class="{ 'is-pad': !slot }">{{ slot || '\u2013' }}</span></span></td>
                <td class="pts">{{ pointsLabel(row.Points) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="cards mob-only">
          <article v-for="(row, index) in standings" :key="row.PlayerID" class="pcard" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
            <div class="rk">{{ index + 1 }}</div>
            <div>
              <div class="nm">{{ displayName(row) }}<span v-if="left(row)" class="left-note">{{ ' ' + $t('rankings.leftGroup') }}</span></div>
              <div class="st"><span class="form"><span class="form-k">{{ $t('predictions.cols.last5') }}</span><span v-for="(slot, i) in lastFive(row)" :key="i" class="form-cell" :class="{ 'is-pad': !slot }">{{ slot || '\u2013' }}</span></span></div>
            </div>
            <div class="pts">{{ pointsLabel(row.Points) }}</div>
          </article>
        </div>
      </template>
    </template>

    <AppToast ref="toast" />
  </main>
</template>

<script>
import AppToast from '@/components/AppToast.vue';
import { actionErrorMessage } from '@/services/v2Match';
import { formatDecimal } from '@/services/intl';
import {
  getMatchesDetails, getMatchPredictions, predictForMatch, getPredictionStandings, getSeasons, getToken,
} from '@/services/api';
import { resolveActiveGroup, setActiveGroupId } from '@/services/activeGroup';
import { isScheduledMatch } from '@/services/matchRegistration';
import {
  rosterFullyComposed, predictionsAreOpen, predictionDisplayState, matchWinnerId, settlePrediction,
  PREDICTIONS_OPEN, PREDICTIONS_SETTLED, PREDICTIONS_AWAITING_RESULT,
} from '@/services/matchPrediction';
import { heroDateLabel, kickoffLabel } from '@/services/v2Home';

function currentPlayerId() {
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
  name: 'PredictionsV2',
  components: { AppToast },
  inject: {
    v2Context: { default: () => ({}) },
  },
  data() {
    return {
      loading: true,
      failed: false,
      boardLoading: false,
      boardFailed: false,
      matches: [],
      summaries: {},
      standings: [],
      playerId: '',
      groupId: '',
      season: '',
      seasons: [],
      nowMs: Date.now(),
    };
  },
  computed: {
    tab() {
      return this.$route.query.tab === 'leaderboard' ? 'leaderboard' : 'matches';
    },
    isEmptyGroup() {
      return this.seasons.length === 0;
    },
    eligible() {
      return this.matches.filter((match) => isScheduledMatch(match) && rosterFullyComposed(match));
    },
  },
  watch: {
    tab(value) {
      if (value === 'leaderboard' && !this.standings.length && !this.boardLoading) this.loadBoard();
    },
  },
  created() {
    this.playerId = currentPlayerId();
    this.load();
  },
  methods: {
    summary(match) {
      return this.summaries[match.ID] || {};
    },
    open(match) {
      return predictionsAreOpen(match, this.nowMs);
    },
    state(match) {
      return predictionDisplayState(match, this.nowMs);
    },
    teamName(match, index) {
      const team = match.Teams && match.Teams[index];
      return (team && team.Name) || this.$t('predictions.team');
    },
    showScore(match) {
      return this.state(match) === PREDICTIONS_SETTLED;
    },
    whenLabel(match) {
      const date = heroDateLabel(match.Date);
      if (this.state(match) === PREDICTIONS_OPEN) {
        const time = kickoffLabel(match.ScheduledAt);
        return time ? this.$t('predictions.whenTime', { date, time }) : date;
      }
      if (this.state(match) === PREDICTIONS_AWAITING_RESULT) return this.$t('predictions.whenAwaiting', { date });
      return this.$t('predictions.whenPlayed', { date });
    },
    mine(match) {
      return this.summary(match).MyPrediction || null;
    },
    note(match) {
      if (!this.open(match)) return '';
      return this.mine(match)
        ? this.$t('predictions.noteOpen')
        : this.$t('predictions.noteWaiting');
    },
    options(match) {
      const odds = this.summary(match).Odds || {};
      const [teamA, teamB] = match.Teams;
      return [
        { key: 'a', teamId: teamA.ID, label: teamA.Name, odds: odds.TeamAOdds },
        { key: 'draw', teamId: null, label: this.$t('match.draw'), odds: odds.DrawOdds },
        { key: 'b', teamId: teamB.ID, label: teamB.Name, odds: odds.TeamBOdds },
      ];
    },
    oddsLabel(value) {
      const number = Number(value);
      return Number.isFinite(number) ? formatDecimal(number) : '';
    },
    pickClass(match, teamId) {
      const classes = [];
      const mine = this.mine(match);
      if (mine && (mine.PredictedTeamID || null) === teamId) classes.push('is-on');
      if (this.state(match) === PREDICTIONS_SETTLED) {
        classes.push(matchWinnerId(match) === teamId ? 'is-win' : 'is-lose');
      }
      return classes;
    },
    outcome(match) {
      const mine = this.mine(match);
      if (!mine) return null;
      return settlePrediction(match, this.summary(match).Odds, mine.PredictedTeamID);
    },
    resultLine(match) {
      const state = this.state(match);
      if (state === PREDICTIONS_OPEN && this.mine(match)) {
        const picked = this.mine(match).PredictedTeamID;
        const option = this.options(match).find((item) => item.teamId === (picked || null));
        return option ? this.$t('predictions.yourPick', { label: option.label, odds: this.oddsLabel(option.odds) }) : '';
      }
      const outcome = this.outcome(match);
      if (!outcome) {
        return state === PREDICTIONS_AWAITING_RESULT ? this.$t('predictions.awaitingResult') : '';
      }
      const points = `${outcome.points > 0 ? '+' : '−'}${formatDecimal(Math.abs(outcome.points))}`;
      return outcome.correct ? this.$t('predictions.won', { points }) : this.$t('predictions.lost', { points });
    },
    resultKind(match) {
      const outcome = this.outcome(match);
      if (outcome) return outcome.correct ? 'ok' : 'ko';
      if (this.state(match) === PREDICTIONS_OPEN && this.mine(match)) return 'ok';
      return 'wait';
    },
    isMe(row) {
      return Boolean(this.playerId) && row.PlayerID === this.playerId;
    },
    left(row) {
      return row.IsMember === false;
    },
    displayName(row) {
      return String(row.Name || '').split(' ').filter(Boolean)
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
        .join(' ');
    },
    scoreText(match) {
      return ` ${match.Teams[0].Score}\u2013${match.Teams[1].Score} `;
    },
    lastFive(row) {
      const recent = (row.History || []).slice(-5).map((entry) => (entry.Correct ? this.$t('predictions.formWin') : this.$t('predictions.formLoss')));
      return [...Array(5 - recent.length).fill(null), ...recent];
    },
    pointsLabel(points) {
      const rounded = Math.round(Number(points) * 10) / 10;
      const text = formatDecimal(Math.abs(rounded), 1);
      if (rounded > 0) return `+${text}`;
      if (rounded < 0) return `−${text}`;
      return formatDecimal(0, 1);
    },
    async choose(match, teamId) {
      if (!this.open(match)) return;
      try {
        const summary = await predictForMatch(match.ID, teamId);
        this.summaries = { ...this.summaries, [match.ID]: summary };
      } catch (error) {
        // Same contract as today's Pronostic page: say why (kick-off has
        // passed, say) and re-read the match's summary.
        console.error(error);
        this.$refs.toast.show(actionErrorMessage(error, this.$t('predictions.saveError')), 'error');
        const summary = await getMatchPredictions(match.ID).catch(() => null);
        if (summary) this.summaries = { ...this.summaries, [match.ID]: summary };
      }
    },
    async load() {
      this.loading = true;
      this.failed = false;
      this.nowMs = Date.now();
      try {
        const { groups, activeGroupId } = await resolveActiveGroup();
        this.groupId = activeGroupId || '';
        const seasons = this.groupId ? await getSeasons(this.groupId).catch(() => []) : [];
        const seasonList = Array.isArray(seasons) ? seasons : [];
        this.seasons = seasonList;
        const pinned = sessionStorage.getItem('calciotto-v2-season');
        this.season = pinned && seasonList.includes(pinned)
          ? pinned
          : (seasonList[seasonList.length - 1] || '');
        if (this.v2Context) {
          this.v2Context.groups = groups;
          this.v2Context.seasons = seasonList;
          this.v2Context.groupId = this.groupId;
          this.v2Context.season = this.season;
          this.v2Context.hide = false;
          this.v2Context.onGroup = (id) => {
            setActiveGroupId(id);
            window.location.reload();
          };
          this.v2Context.onSeason = (next) => {
            sessionStorage.setItem('calciotto-v2-season', next);
            this.v2Context.season = next;
            this.season = next;
            this.loadMatches();
            this.loadBoard();
          };
        }
        await this.loadMatches();
        if (this.tab === 'leaderboard') await this.loadBoard();
      } catch (error) {
        console.error(error);
        this.failed = true;
        this.loading = false;
      }
    },
    async loadMatches() {
      this.loading = true;
      this.failed = false;
      try {
        const matches = await getMatchesDetails(this.groupId, this.season);
        this.matches = Array.isArray(matches) ? matches : [];
        const eligible = this.matches.filter((match) => isScheduledMatch(match) && rosterFullyComposed(match));
        const pairs = await Promise.all(eligible.map(async (match) => {
          const summary = await getMatchPredictions(match.ID).catch(() => null);
          return [match.ID, summary];
        }));
        this.summaries = Object.fromEntries(pairs.filter((pair) => pair[1]));
      } catch (error) {
        console.error(error);
        this.failed = true;
      } finally {
        this.loading = false;
      }
    },
    async loadBoard() {
      this.boardLoading = true;
      this.boardFailed = false;
      try {
        const rows = await getPredictionStandings(this.season, this.groupId);
        this.standings = Array.isArray(rows) ? rows : [];
      } catch (error) {
        console.error(error);
        this.boardFailed = true;
      } finally {
        this.boardLoading = false;
      }
    },
  },
};
</script>
