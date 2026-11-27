<template>
  <main class="page">
    <nav v-if="!isEmptyGroup" class="subnav rank-tabs" aria-label="Rankings">
      <router-link to="/rankings" :class="{ 'is-on': tab === 'points' }">
        Points
      </router-link>
      <router-link to="/rankings?tab=scorers" :class="{ 'is-on': tab === 'scorers' }">
        Scorers
      </router-link>
      <router-link to="/rankings?tab=motm" :class="{ 'is-on': tab === 'motm' }">
        {{ stars }}
      </router-link>
      <router-link to="/rankings?tab=agoranking" :class="{ 'is-on': tab === 'agoranking' }">
        Agoranking
      </router-link>
    </nav>

    <section v-if="loading" class="state">
      <div class="skel md"></div>
      <div class="skel"></div>
      <div class="skel"></div>
    </section>
    <section v-else-if="failed" class="state">
      <h1>Couldn't load the rankings</h1>
      <button class="btn" type="button" @click="load">Try again</button>
    </section>
    <section v-else-if="tab === 'agoranking' && agorankingFailed" class="state">
      <h1>Couldn't load the Agoranking</h1>
      <button class="btn" type="button" @click="load">Try again</button>
    </section>
    <section v-else-if="isEmptyGroup" class="state">
      <p class="label">No matches played yet</p>
      <h1>Rankings</h1>
      <p>Standings appear once the first score is recorded.</p>
    </section>
    <section v-else-if="rows.length === 0" class="state">
      <h1>{{ emptyTitle }}</h1>
    </section>

    <section v-else-if="tab === 'agoranking'" class="agr">
      <h1 class="agr-title">Agoranking</h1>
      <p class="agr-lead">
        Who wins the most points <b>per match</b>, not in total. A few lucky games aren't enough: the more you play, the more your own results count.
      </p>

      <div class="agr-fold agr-how" :class="{ 'is-open': howOpen }">
        <button id="agr-how-head" class="agr-fold-head" type="button" :aria-expanded="howOpen.toString()"
          aria-controls="agr-how-body" @click="howOpen = !howOpen">
          <span>How it's calculated</span>
        </button>
        <div v-show="howOpen" id="agr-how-body" class="agr-fold-body" role="region" aria-labelledby="agr-how-head">
          <p>A win is worth 3 points, a draw 1, a loss 0. Every player starts with a few <b>“average” matches</b> worth the group's average points per match. Your real matches are added on top, so a newcomer sits near the average and moves away from it as they play.</p>
          <div class="agr-formula">rating = (<b>points</b> + N × group avg) ÷ (<b>matches</b> + N)</div>
          <p class="agr-note">N is a third of the matches the group has played this season, rounded up. It's also the minimum number of matches you need to be ranked, so both grow as the season goes on.</p>
        </div>
      </div>

      <template v-if="agoranking.Ranked.length">
        <div class="table-wrap desk-only">
          <table class="stand agr-table">
            <thead>
              <tr><th>#</th><th>Player</th><th>M</th><th>W-D-L</th><th>Pts</th><th>Rating</th></tr>
            </thead>
            <tbody>
              <tr v-for="(row, index) in agoranking.Ranked" :key="row.PlayerID" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
                <td class="rk">{{ index + 1 }}</td>
                <td class="pl">{{ displayName(row) }}<span v-if="isMe(row)" class="agr-you">YOU</span><span v-if="left(row)" class="left-note"> (left the group)</span></td>
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
              <div class="nm">{{ displayName(row) }}<span v-if="isMe(row)" class="agr-you">YOU</span><span v-if="left(row)" class="left-note"> (left the group)</span></div>
              <div class="st">{{ rankedLine(row) }}</div>
            </div>
            <div class="pts">{{ rating(row.Rating) }}</div>
          </article>
        </div>
      </template>

      <div v-if="agoranking.Unranked.length" class="agr-fold agr-unranked" :class="{ 'is-open': unrankedOpen }">
        <button id="agr-unranked-head" class="agr-fold-head" type="button" :aria-expanded="unrankedOpen.toString()"
          aria-controls="agr-unranked-body" @click="unrankedOpen = !unrankedOpen">
          <span>Not ranked yet · {{ agoranking.Unranked.length }}</span>
        </button>
        <div v-show="unrankedOpen" id="agr-unranked-body" class="agr-fold-body" role="region" aria-labelledby="agr-unranked-head">
          <div class="table-wrap desk-only">
            <table class="stand agr-table agr-unranked-table">
              <tbody>
                <tr v-for="row in agoranking.Unranked" :key="row.PlayerID" :class="{ 'is-me': isMe(row) }">
                  <td class="rk">–</td>
                  <td class="pl">{{ displayName(row) }}<span v-if="isMe(row)" class="agr-you">YOU</span><span v-if="left(row)" class="left-note"> (left the group)</span></td>
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
                <div class="nm">{{ displayName(row) }}<span v-if="isMe(row)" class="agr-you">YOU</span><span v-if="left(row)" class="left-note"> (left the group)</span></div>
                <div class="st">{{ recordLine(row) }}</div>
              </div>
              <div class="m">{{ row.Played }} M</div>
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
              <th>#</th><th>Player</th>              <th>P</th><th>W</th><th>D</th><th>L</th><th>GF</th><th>OG</th><th>Pts</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in rows" :key="row.PlayerID" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
              <td class="rk">{{ index + 1 }}</td>
              <td>{{ displayName(row) }}<span v-if="left(row)" class="left-note"> (left the group)</span></td>
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
            <div class="nm">{{ displayName(row) }}<span v-if="left(row)" class="left-note"> (left the group)</span></div>
            <div class="st">P {{ row.Played }} · W {{ row.Won }} · D {{ row.Drawn }} · L {{ row.Lost }} · GF {{ row.GoalsFor }}</div>
          </div>
          <div class="pts">{{ row.Points }}</div>
        </article>
      </div>
    </template>

    <template v-else-if="tab === 'scorers'">
      <div class="table-wrap desk-only">
        <table class="stand">
          <thead>
            <tr><th>#</th><th>Player</th><th></th><th>Goals</th></tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in rows" :key="row.PlayerID" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
              <td class="rk">{{ index + 1 }}</td>
              <td>{{ displayName(row) }}<span v-if="left(row)" class="left-note"> (left the group)</span></td>
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
            <div class="nm">{{ displayName(row) }}<span v-if="left(row)" class="left-note"> (left the group)</span></div>
            <div class="st">{{ goalsLabel(row.Goals) }} · {{ playedLabel(row.Played) }}</div>
          </div>
          <div class="pts">{{ row.Goals }}</div>
        </article>
      </div>
    </template>

    <template v-else>
      <div class="table-wrap desk-only">
        <table class="stand">
          <thead>
            <tr><th>#</th><th>Player</th><th>Awards</th></tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in rows" :key="row.PlayerID" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
              <td class="rk">{{ index + 1 }}</td>
              <td>{{ displayName(row) }}<span v-if="left(row)" class="left-note"> (left the group)</span></td>
              <td class="pts">{{ row.Awards }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="cards mob-only">
        <article v-for="(row, index) in rows" :key="row.PlayerID" class="pcard" :class="{ 'is-me': isMe(row), 'is-left': left(row) }">
          <div class="rk">{{ index + 1 }}</div>
          <div>
            <div class="nm">{{ displayName(row) }}<span v-if="left(row)" class="left-note"> (left the group)</span></div>
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
import { formatRating, rankedLine, recordLine, normalizeAgoranking, emptyAgoranking } from '@/services/agoranking';
import { STARS } from '@/services/v2Home';
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

function countLabel(count, singular, plural) {
  return `${count} ${count === 1 ? singular : plural}`;
}

export default {
  name: 'RankingsV2',
  inject: {
    v2Context: { default: () => ({}) },
  },
  data() {
    return {
      stars: STARS,
      loading: true,
      failed: false,
      points: [],
      scorers: [],
      motm: [],
      agoranking: emptyAgoranking(),
      agorankingFailed: false,
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
      return ['scorers', 'motm', 'agoranking'].includes(value) ? value : 'points';
    },
    rows() {
      if (this.tab === 'scorers') return this.scorers;
      if (this.tab === 'motm') return this.motm;
      if (this.tab === 'agoranking') return [...this.agoranking.Ranked, ...this.agoranking.Unranked];
      return this.points;
    },
    emptyTitle() {
      if (this.tab === 'scorers') return 'No goals yet';
      if (this.tab === 'motm') return `No ${STARS.toLowerCase()} yet`;
      return 'No standings yet';
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
      return countLabel(count, 'match', 'matches');
    },
    goalsLabel(count) {
      return countLabel(count, 'goal', 'goals');
    },
    awardsLabel(count) {
      return countLabel(count, 'award', 'awards');
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
        const { groups, activeGroupId } = await resolveActiveGroup();
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
          getAgoranking(season, this.groupId).catch(() => null),
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
