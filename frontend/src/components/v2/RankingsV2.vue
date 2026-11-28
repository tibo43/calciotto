<template>
  <main class="page">
    <nav v-if="!isEmptyGroup" class="subnav rank-tabs" :aria-label="$t('rankings.label')">
      <router-link to="/rankings" :class="{ 'is-on': tab === 'points' }">
        {{ $t('rankings.tabs.points') }}
      </router-link>
      <router-link to="/rankings?tab=scorers" :class="{ 'is-on': tab === 'scorers' }">
        {{ $t('rankings.tabs.scorers') }}
      </router-link>
      <router-link to="/rankings?tab=motm" :class="{ 'is-on': tab === 'motm' }">
        {{ $t('glossary.mvp') }}
      </router-link>
      <router-link v-if="agorankingEnabled" to="/rankings?tab=agoranking" :class="{ 'is-on': tab === 'agoranking' }">
        {{ $t('glossary.agoranking') }}
      </router-link>
    </nav>

    <section v-if="loading" class="state">
      <div class="skel md"></div>
      <div class="skel"></div>
      <div class="skel"></div>
    </section>
    <section v-else-if="failed" class="state">
      <h1>{{ $t('rankings.loadError') }}</h1>
      <button class="btn" type="button" @click="load">{{ $t('common.tryAgain') }}</button>
    </section>
    <section v-else-if="tab === 'agoranking' && agorankingFailed" class="state">
      <h1>{{ $t('rankings.agorankingLoadError') }}</h1>
      <button class="btn" type="button" @click="load">{{ $t('common.tryAgain') }}</button>
    </section>
    <section v-else-if="isEmptyGroup" class="state">
      <p class="label">{{ $t('rankings.emptyGroup.label') }}</p>
      <h1>{{ $t('rankings.emptyGroup.title') }}</h1>
      <p>{{ $t('rankings.emptyGroup.text') }}</p>
    </section>
    <section v-else-if="rows.length === 0" class="state">
      <h1>{{ emptyTitle }}</h1>
      <p v-if="emptyText">{{ emptyText }}</p>
    </section>

    <section v-else-if="tab === 'agoranking'" class="agr">
      <h1 class="agr-title">{{ $t('glossary.agoranking') }}</h1>
      <p class="agr-lead">
        <i18n-t keypath="rankings.agr.lead" scope="global"><template #perMatch><b>{{ $t('rankings.agr.perMatch') }}</b></template></i18n-t>
      </p>

      <div class="agr-fold agr-how" :class="{ 'is-open': howOpen }">
        <button id="agr-how-head" class="agr-fold-head" type="button" :aria-expanded="howOpen.toString()"
          aria-controls="agr-how-body" @click="howOpen = !howOpen">
          <span>{{ $t('rankings.agr.how') }}</span>
        </button>
        <div v-show="howOpen" id="agr-how-body" class="agr-fold-body" role="region" aria-labelledby="agr-how-head">
          <p><i18n-t keypath="rankings.agr.howText" scope="global"><template #average><b>{{ $t('rankings.agr.average') }}</b></template></i18n-t></p>
          <div class="agr-formula"><i18n-t keypath="rankings.agr.formula" scope="global"><template #points><b>{{ $t('rankings.agr.formulaPoints') }}</b></template><template #matches><b>{{ $t('rankings.agr.formulaMatches') }}</b></template></i18n-t></div>
          <p class="agr-note">{{ $t('rankings.agr.note') }}</p>
        </div>
      </div>

      <template v-if="agoranking.Ranked.length">
        <div class="table-wrap desk-only">
          <table class="stand agr-table">
            <thead>
              <tr><th>{{ $t('rankings.cols.rank') }}</th><th>{{ $t('rankings.cols.player') }}</th><th>{{ $t('rankings.cols.matchesShort') }}</th><th>{{ $t('rankings.cols.wdl') }}</th><th>{{ $t('rankings.cols.pts') }}</th><th>{{ $t('rankings.cols.rating') }}</th></tr>
            </thead>
            <tbody>
              <tr v-for="(row, index) in agoranking.Ranked" :key="row.PlayerID" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
                <td class="rk">{{ index + 1 }}</td>
                <td class="pl">{{ displayName(row) }}<span v-if="isMe(row)" class="agr-you">{{ $t('rankings.you') }}</span><span v-if="left(row)" class="left-note">{{ ' ' + $t('rankings.leftGroup') }}</span></td>
                <td class="n">{{ row.Played }}</td>
                <td class="n">{{ row.Won }}-{{ row.Drawn }}-{{ row.Lost }}</td>
                <td class="n">{{ row.Points }}</td>
                <td class="pts">{{ rating(row.Rating) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="cards mob-only">
          <article v-for="(row, index) in agoranking.Ranked" :key="row.PlayerID" class="pcard agr-card" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
            <div class="rk">{{ index + 1 }}</div>
            <div>
              <div class="nm">{{ displayName(row) }}<span v-if="isMe(row)" class="agr-you">{{ $t('rankings.you') }}</span><span v-if="left(row)" class="left-note">{{ ' ' + $t('rankings.leftGroup') }}</span></div>
              <div class="st">{{ rankedLine(row) }}</div>
            </div>
            <div class="pts">{{ rating(row.Rating) }}</div>
          </article>
        </div>
      </template>

      <div v-if="agoranking.Unranked.length" class="agr-fold agr-unranked" :class="{ 'is-open': unrankedOpen }">
        <button id="agr-unranked-head" class="agr-fold-head" type="button" :aria-expanded="unrankedOpen.toString()"
          aria-controls="agr-unranked-body" @click="unrankedOpen = !unrankedOpen">
          <span>{{ $t('rankings.agr.notRanked', { n: agoranking.Unranked.length }) }}</span>
        </button>
        <div v-show="unrankedOpen" id="agr-unranked-body" class="agr-fold-body" role="region" aria-labelledby="agr-unranked-head">
          <div class="table-wrap desk-only">
            <table class="stand agr-table agr-unranked-table">
              <tbody>
                <tr v-for="row in agoranking.Unranked" :key="row.PlayerID" :class="{ 'is-me': isMe(row) }">
                  <td class="rk">–</td>
                  <td class="pl">{{ displayName(row) }}<span v-if="isMe(row)" class="agr-you">{{ $t('rankings.you') }}</span><span v-if="left(row)" class="left-note">{{ ' ' + $t('rankings.leftGroup') }}</span></td>
                  <td class="n">{{ row.Played }}</td>
                  <td class="n">{{ row.Won }}-{{ row.Drawn }}-{{ row.Lost }}</td>
                  <td class="n">{{ row.Points }}</td>
                  <td class="n">–</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="cards mob-only">
            <article v-for="row in agoranking.Unranked" :key="row.PlayerID" class="agr-ucard" :class="{ 'is-me': isMe(row) }">
              <div>
                <div class="nm">{{ displayName(row) }}<span v-if="isMe(row)" class="agr-you">{{ $t('rankings.you') }}</span><span v-if="left(row)" class="left-note">{{ ' ' + $t('rankings.leftGroup') }}</span></div>
                <div class="st">{{ recordLine(row) }}</div>
              </div>
              <div class="m">{{ $t('rankings.agr.playedShort', { n: row.Played }) }}</div>
            </article>
          </div>
        </div>
      </div>
    </section>

    <template v-else-if="tab === 'points'">
      <div class="table-wrap desk-only">
        <table class="stand">
          <thead>
            <tr>
              <th>{{ $t('rankings.cols.rank') }}</th><th>{{ $t('rankings.cols.player') }}</th>              <th>{{ $t('rankings.cols.played') }}</th><th>{{ $t('rankings.cols.won') }}</th><th>{{ $t('rankings.cols.drawn') }}</th><th>{{ $t('rankings.cols.lost') }}</th><th>{{ $t('rankings.cols.goalsFor') }}</th><th>{{ $t('rankings.cols.ownGoals') }}</th><th>{{ $t('rankings.cols.pts') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in rows" :key="row.PlayerID" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
              <td class="rk">{{ index + 1 }}</td>
              <td>{{ displayName(row) }}<span v-if="left(row)" class="left-note">{{ ' ' + $t('rankings.leftGroup') }}</span></td>
              <td>{{ row.Played }}</td>
              <td>{{ row.Won }}</td>
              <td>{{ row.Drawn }}</td>
              <td>{{ row.Lost }}</td>
              <td>{{ row.GoalsFor }}</td>
              <td>{{ row.OwnGoals }}</td>
              <td class="pts">{{ row.Points }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="cards mob-only">
        <article v-for="(row, index) in rows" :key="row.PlayerID" class="pcard" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
          <div class="rk">{{ index + 1 }}</div>
          <div>
            <div class="nm">{{ displayName(row) }}<span v-if="left(row)" class="left-note">{{ ' ' + $t('rankings.leftGroup') }}</span></div>
            <div class="st">{{ $t('rankings.pointsLine', { played: row.Played, won: row.Won, drawn: row.Drawn, lost: row.Lost, goalsFor: row.GoalsFor }) }}</div>
          </div>
          <div class="pts">{{ row.Points }}</div>
        </article>
      </div>
    </template>

    <template v-else-if="tab === 'scorers'">
      <div class="table-wrap desk-only">
        <table class="stand">
          <thead>
            <tr><th>{{ $t('rankings.cols.rank') }}</th><th>{{ $t('rankings.cols.player') }}</th><th></th><th>{{ $t('rankings.cols.goals') }}</th></tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in rows" :key="row.PlayerID" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
              <td class="rk">{{ index + 1 }}</td>
              <td>{{ displayName(row) }}<span v-if="left(row)" class="left-note">{{ ' ' + $t('rankings.leftGroup') }}</span></td>
              <td>{{ playedLabel(row.Played) }}</td>
              <td class="pts">{{ row.Goals }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="cards mob-only">
        <article v-for="(row, index) in rows" :key="row.PlayerID" class="pcard" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
          <div class="rk">{{ index + 1 }}</div>
          <div>
            <div class="nm">{{ displayName(row) }}<span v-if="left(row)" class="left-note">{{ ' ' + $t('rankings.leftGroup') }}</span></div>
            <div class="st">{{ $t('rankings.scorersLine', { goals: goalsLabel(row.Goals), played: playedLabel(row.Played) }) }}</div>
          </div>
          <div class="pts">{{ row.Goals }}</div>
        </article>
      </div>
    </template>

    <template v-else>
      <div class="table-wrap desk-only">
        <table class="stand">
          <thead>
            <tr><th>{{ $t('rankings.cols.rank') }}</th><th>{{ $t('rankings.cols.player') }}</th><th>{{ $t('rankings.cols.awards') }}</th></tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in rows" :key="row.PlayerID" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
              <td class="rk">{{ index + 1 }}</td>
              <td>{{ displayName(row) }}<span v-if="left(row)" class="left-note">{{ ' ' + $t('rankings.leftGroup') }}</span></td>
              <td class="pts">{{ row.Awards }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="cards mob-only">
        <article v-for="(row, index) in rows" :key="row.PlayerID" class="pcard" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
          <div class="rk">{{ index + 1 }}</div>
          <div>
            <div class="nm">{{ displayName(row) }}<span v-if="left(row)" class="left-note">{{ ' ' + $t('rankings.leftGroup') }}</span></div>
            <div class="st">{{ awardsLabel(row.Awards) }}</div>
          </div>
          <div class="pts">{{ row.Awards }}</div>
        </article>
      </div>
    </template>
  </main>
</template>

<script>
import { getPointsStandings, getScorers, getMotmStandings, getAgoranking, getSeasons, getToken } from '@/services/api';
import { formatRating, rankedLine, recordLine, normalizeAgoranking, emptyAgoranking, AGORANKING_FEATURE } from '@/services/agoranking';
import { knownFeatureStatus, loadFeatureStatus } from '@/services/featureStatus';
import { resolveActiveGroup, setActiveGroupId } from '@/services/activeGroup';

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
  name: 'RankingsV2',
  inject: {
    v2Context: { default: () => ({}) },
  },
  data() {
    return {
      loading: true,
      failed: false,
      points: [],
      scorers: [],
      motm: [],
      agoranking: emptyAgoranking(),
      agorankingFailed: false,
      // Hidden feature: the tab and its request only exist with it.
      agorankingEnabled: false,
      howOpen: false,
      unrankedOpen: false,
      playerId: '',
      groupId: '',
      seasons: [],
    };
  },
  computed: {
    isEmptyGroup() {
      return this.seasons.length === 0;
    },
    tab() {
      const value = this.$route.query.tab;
      if (value === 'agoranking') return this.agorankingEnabled ? value : 'points';
      return value === 'scorers' || value === 'motm' ? value : 'points';
    },
    rows() {
      if (this.tab === 'scorers') return this.scorers;
      if (this.tab === 'motm') return this.motm;
      if (this.tab === 'agoranking') return [...this.agoranking.Ranked, ...this.agoranking.Unranked];
      return this.points;
    },
    emptyTitle() {
      if (this.tab === 'scorers') return this.$t('rankings.empty.scorers');
      if (this.tab === 'motm') return this.$t('rankings.empty.motm');
      if (this.tab === 'agoranking') return this.$t('rankings.empty.agoranking');
      return this.$t('rankings.empty.points');
    },
    // Points, scorers and the Agoranking fill up from played matches; MVP
    // awards need votes too, so their tab keeps the bare title.
    emptyText() {
      return this.tab === 'motm' ? '' : this.$t('rankings.empty.text');
    },
  },
  watch: {
    '$route.query.tab'() {
      this.failed = false;
    },
  },
  created() {
    this.playerId = currentPlayerId();
    this.load();
  },
  methods: {
    rating(value) {
      return formatRating(value);
    },
    rankedLine(row) {
      return rankedLine(row);
    },
    recordLine(row) {
      return recordLine(row);
    },
    playedLabel(count) {
      return this.$t('rankings.matches', count);
    },
    goalsLabel(count) {
      return this.$t('rankings.goals', count);
    },
    awardsLabel(count) {
      return this.$t('rankings.awards', count);
    },
    isMe(row) {
      return Boolean(this.playerId) && row.PlayerID === this.playerId;
    },
    left(row) {
      return row.IsMember === false;
    },
    displayName(row) {
      return String(row.Name || '').split(' ')
        .filter(Boolean)
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
        .join(' ');
    },
    async load() {
      this.loading = true;
      this.failed = false;
      try {
        const [{ groups, activeGroupId }, featureStatus] = await Promise.all([
          resolveActiveGroup(),
          Promise.resolve(knownFeatureStatus() ?? loadFeatureStatus()).catch(() => null),
        ]);
        const features = Array.isArray(featureStatus?.features) ? featureStatus.features : [];
        this.agorankingEnabled = features.includes(AGORANKING_FEATURE);
        this.groupId = activeGroupId || '';
        const seasons = this.groupId ? await getSeasons(this.groupId).catch(() => []) : [];
        const seasonList = Array.isArray(seasons) ? seasons : [];
        this.seasons = seasonList;
        const pinned = sessionStorage.getItem('calciotto-v2-season');
        const season = pinned && seasonList.includes(pinned)
          ? pinned
          : (seasonList[seasonList.length - 1] || '');
        if (this.v2Context) {
          this.v2Context.groups = groups;
          this.v2Context.seasons = seasonList;
          this.v2Context.groupId = this.groupId;
          this.v2Context.season = season;
          this.v2Context.hide = false;
          this.v2Context.onGroup = (id) => {
            setActiveGroupId(id);
            window.location.reload();
          };
          this.v2Context.onSeason = (next) => {
            sessionStorage.setItem('calciotto-v2-season', next);
            this.v2Context.season = next;
            this.loadTables(next);
          };
        }
        await this.loadTables(season);
      } catch (error) {
        console.error(error);
        this.failed = true;
        this.loading = false;
      }
    },
    async loadTables(season) {
      this.loading = true;
      this.failed = false;
      try {
        // A failing Agoranking only breaks its own tab, not the other three.
        const [points, scorers, motm, agoranking] = await Promise.all([
          getPointsStandings(season, this.groupId),
          getScorers(season, this.groupId),
          getMotmStandings(season, this.groupId),
          this.agorankingEnabled ? getAgoranking(season, this.groupId).catch(() => null) : emptyAgoranking(),
        ]);
        this.points = Array.isArray(points) ? points : [];
        this.scorers = Array.isArray(scorers) ? scorers : [];
        this.motm = Array.isArray(motm) ? motm : [];
        this.agorankingFailed = agoranking === null;
        this.agoranking = normalizeAgoranking(agoranking);
      } catch (error) {
        console.error(error);
        this.failed = true;
      } finally {
        this.loading = false;
      }
    },
  },
};
</script>
