<template>
  <LoginV2 v-if="on" />
  <LoginPage v-else />
</template>

<script>
import LoginPage from '@/components/Login.vue';
import LoginV2 from '@/components/v2/LoginV2.vue';

// /login renders today's Login.vue unless App.vue turned design v2 on for
// this page (see shouldUseLoginV2 in services/loginV2.js). Following the
// shell's answer keeps the page and the top bar in step.
export default {
  name: 'LoginGate',
  components: { LoginPage, LoginV2 },
  inject: {
    designV2On: { default: false },
  },
  computed: {
    on() {
      return Boolean(this.designV2On);
    },
  },
};
</script>
