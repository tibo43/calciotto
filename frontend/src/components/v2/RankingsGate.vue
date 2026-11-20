<template>
  <RankingsV2 v-if="on" />
</template>

<script>
import RankingsV2 from '@/components/v2/RankingsV2.vue';

// /rankings is a v1 redirect unless the server says design-v2 is on (see
// the route's guard). If the flag later turns out off, leave.
export default {
  name: 'RankingsGate',
  components: { RankingsV2 },
  inject: {
    designV2On: { default: false },
  },
  computed: {
    on() {
      return Boolean(this.designV2On);
    },
  },
  watch: {
    on(value) {
      if (!value) this.$router.replace('/');
    },
  },
};
</script>
