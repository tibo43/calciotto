<template>
  <RankingsV2 v-if="on" />
</template>

<script>
import RankingsV2 from '@/components/v2/RankingsV2.vue';

// /classements is a v1 redirect unless the design-v2 cache is set (see the
// route). If the features response then says the flag is off, leave.
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
