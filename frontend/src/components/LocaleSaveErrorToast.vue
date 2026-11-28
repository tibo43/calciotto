<template>
  <AppToast ref="toast" />
</template>

<script>
import AppToast from '@/components/AppToast.vue';
import { onLocaleSaveError } from '@/services/playerLocale';

// The toast that says a language could not be saved. Mounted once by the app
// shell (App.vue), so it shows wherever the player is when the save fails,
// not only on the Account page where the selector lives. The locale service
// has already put the previous language back when this runs, so the text
// comes out in that language.
export default {
  name: 'LocaleSaveErrorToast',
  components: { AppToast },
  mounted() {
    this.stopSaveErrors = onLocaleSaveError(() => {
      this.$refs.toast?.show(this.$t('language.saveError'), 'error');
    });
  },
  beforeUnmount() {
    if (this.stopSaveErrors) this.stopSaveErrors();
  },
};
</script>
