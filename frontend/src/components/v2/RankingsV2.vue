<template>
  <main class="page">
    <nav class="subnav" aria-label="Rankings">
      <router-link to="/classements" :class="{ 'is-on': tab === 'points' }">
        <span class="outline" :class="{ 'is-on': tab === 'points' }">Points</span>
      </router-link>
      <router-link to="/classements?tab=scorers" :class="{ 'is-on': tab === 'scorers' }">
        <span class="outline" :class="{ 'is-on': tab === 'scorers' }">Scorers</span>
      </router-link>
      <router-link to="/classements?tab=motm" :class="{ 'is-on': tab === 'motm' }">
        <span class="outline" :class="{ 'is-on': tab === 'motm' }">MOTM</span>
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
    <section v-else-if="rows.length === 0" class="state">
      <h1>{{ emptyTitle }}</h1>
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
import { getPointsStandings, getScorers, getMotmStandings, getSeasons, getToken } from '@/services/api';
import { resolveActiveGroup, setActiveGroupId } from '@/services/activeGroup';

function currentPlayerId() {
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
      loading: true,
      failed: false,
      points: [],
      scorers: [],
      motm: [],
      playerId: '',
      groupId: '',
    };
  },
  computed: {
    tab() {
      const value = this.$route.query.tab;
      return value === 'scorers' || value === 'motm' ? value : 'points';
    },
    rows() {
      if (this.tab === 'scorers') return this.scorers;
      if (this.tab === 'motm') return this.motm;
      return this.points;
    },
    emptyTitle() {
      if (this.tab === 'scorers') return 'No goals yet';
      if (this.tab === 'motm') return 'No MOTM awards yet';
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
        const seasons = await getSeasons(this.groupId).catch(() => []);
        const seasonList = Array.isArray(seasons) ? seasons : [];
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
        const [points, scorers, motm] = await Promise.all([
          getPointsStandings(season, this.groupId),
          getScorers(season, this.groupId),
          getMotmStandings(season, this.groupId),
        ]);
        this.points = Array.isArray(points) ? points : [];
        this.scorers = Array.isArray(scorers) ? scorers : [];
        this.motm = Array.isArray(motm) ? motm : [];
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
