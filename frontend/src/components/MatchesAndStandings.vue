<template>
  <!-- Do not mount HomeV2 until the active group is known, so it never fires
       GET /matches/details without a group_id (that request would fall back
       to the player's first group). -->
  <div v-if="isInitializing" class="page" aria-busy="true">
    <p class="label">{{ $t('home.nextMatch') }}</p>
    <div class="skel lg"></div>
    <div class="skel md"></div>
    <div class="skel sm"></div>
  </div>
  <HomeV2 v-else :active-group-id="activeGroupId" :group-name="activeGroupName" :season="selectedSeason" :is-admin="isAdmin" />
</template>

<script>
import { getSeasons } from '@/services/api';
import { resolveActiveGroup, setActiveGroupId } from '@/services/activeGroup';
import { seasonOf } from '@/services/datetime';
import { decodeMatchId } from '@/services/shortLink';
import HomeV2 from '@/components/v2/HomeV2.vue';

// The home page: resolves the active group and the season, publishes them to
// the shell's group and season selects (App.vue's v2Context), then hands
// them to HomeV2. The rankings live on their own page (/rankings), so this
// page loads no standings.
export default {
  name: 'MatchesAndStandings',
  components: { HomeV2 },
  inject: {
    v2Context: { default: null },
  },
  data() {
    return {
      activeGroupId: '',
      groups: [],
      isAdmin: false,
      seasons: [],
      selectedSeason: '',
      isInitializing: true,
    };
  },
  computed: {
    activeGroupName() {
      const group = this.groups.find((item) => item.id === this.activeGroupId);
      return group ? group.name : '';
    },
  },
  async created() {
    // A shared /m/:code link lands here as /?match=<code>: the match page
    // finds the match's own group by itself.
    if (this.openDeepLinkedMatch()) {
      return;
    }
    try {
      const { groups, activeGroupId } = await resolveActiveGroup();
      this.groups = groups;
      this.activeGroupId = activeGroupId;
      this.isAdmin = groups.find(g => g.id === activeGroupId)?.role === 'admin';
    } catch (error) {
      console.error('Error resolving the active group:', error);
    }
    await this.loadSeasons();
    this.isInitializing = false;
    this.publishContext();
  },
  methods: {
    switchGroup() {
      setActiveGroupId(this.activeGroupId);
      window.location.reload();
    },
    publishContext() {
      if (!this.v2Context) return;
      const pinned = sessionStorage.getItem('calciotto-v2-season');
      if (pinned && this.seasons.includes(pinned)) this.selectedSeason = pinned;
      this.v2Context.groups = this.groups;
      this.v2Context.seasons = this.seasons;
      this.v2Context.groupId = this.activeGroupId;
      this.v2Context.season = this.selectedSeason;
      this.v2Context.hide = false;
      this.v2Context.onGroup = (id) => {
        this.activeGroupId = id;
        this.switchGroup();
      };
      this.v2Context.onSeason = (season) => {
        sessionStorage.setItem('calciotto-v2-season', season);
        this.selectedSeason = season;
        this.v2Context.season = season;
      };
    },
    openDeepLinkedMatch() {
      const code = this.$route?.query?.match;
      if (!code) {
        return false;
      }
      let matchId;
      try {
        matchId = decodeMatchId(code);
      } catch {
        return false;
      }
      this.$router.replace({ name: 'MatchView', params: { id: matchId } });
      return true;
    },
    // Defaults to the season today falls in, not the last one in the list:
    // a match scheduled into next season must not hide every other match.
    // When today's season has no match yet, the last season on record.
    async loadSeasons() {
      try {
        const seasons = await getSeasons(this.activeGroupId);
        this.seasons = Array.isArray(seasons) ? seasons : [];
      } catch (error) {
        console.error('Error fetching seasons:', error);
        this.seasons = [];
      }
      const current = seasonOf(new Date());
      if (this.seasons.includes(current)) {
        this.selectedSeason = current;
      } else {
        this.selectedSeason = this.seasons.length ? this.seasons[this.seasons.length - 1] : '';
      }
    },
  },
};
</script>
