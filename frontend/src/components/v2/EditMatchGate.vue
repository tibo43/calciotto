<template>
  <EditMatchV2 v-if="on" :id="id" />
  <MatchDetails v-else :id="id" />
</template>

<script>
import MatchDetails from '@/components/MatchDetails.vue';
import EditMatchV2 from '@/components/v2/EditMatchV2.vue';

// The edit route stays MatchDetails while the feature is off, so today's
// page and its screenshots are unchanged. The new steps render only when
// design-v2 is on.
export default {
  name: 'EditMatchGate',
  components: { MatchDetails, EditMatchV2 },
  inject: {
    designV2On: { default: false },
  },
  props: {
    id: { type: String, required: true },
  },
  computed: {
    on() {
      return Boolean(this.designV2On);
    },
  },
};
</script>
