<template>
  <!-- Square 44px ghost, same outline as "View match"; the accessible name is
       "Manage match". -->
  <GearLink class="btn btn-ghost hero-manage" :to="`/matches/${id}/edit`" :label="$t('match.manage')" />
</template>

<script>
import GearLink from '@/components/v2/GearLink.vue';

export default {
  name: 'ManageMatchLink',
  components: { GearLink },
  props: {
    id: { type: [String, Number], required: true },
  },
};
</script>
