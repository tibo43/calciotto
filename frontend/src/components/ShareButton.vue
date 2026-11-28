<template>
  <button v-bind="$attrs" class="share-btn" :class="{ 'compact-on-mobile': compactOnMobile }"
    type="button" :disabled="!text || sharing" :aria-label="label" @click="share">
    <svg v-if="icon || compactOnMobile" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
      stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d="M12 16V3m-4 4 4-4 4 4M7 10H5a1 1 0 0 0-1 1v9a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-9a1 1 0 0 0-1-1h-2" />
    </svg>
    <span class="share-label">{{ label }}</span>
  </button>
  <AppToast ref="toast" />
  <dialog ref="copyDialog" class="share-dialog" :aria-label="$t('share.dialogLabel')" @close="manualText = ''">
    <form method="dialog">
      <label>
        {{ $t('share.copyThis') }}
        <textarea readonly :value="manualText" rows="6" @focus="$event.target.select()"></textarea>
      </label>
      <button class="btn-base btn-cancel" type="submit">{{ $t('common.done') }}</button>
    </form>
  </dialog>
</template>

<script>
import AppToast from './AppToast.vue';

export default {
  name: 'ShareButton',
  inheritAttrs: false,
  components: { AppToast },
  props: {
    label: { type: String, required: true },
    text: { type: String, required: true },
    icon: Boolean,
    compactOnMobile: Boolean,
  },
  data() {
    return { sharing: false, manualText: '' };
  },
  methods: {
    async share() {
      if (!this.text || this.sharing) return;
      this.sharing = true;
      const text = this.text;
      try {
        if (navigator.share && (!navigator.canShare || navigator.canShare({ text }))) {
          try {
            await navigator.share({ text });
            return;
          } catch (error) {
            if (error?.name === 'AbortError') return;
          }
        }
        try {
          await navigator.clipboard.writeText(text);
          this.$refs.toast?.show(this.$t('share.copied'));
        } catch {
          this.manualText = text;
          await this.$nextTick();
          this.$refs.copyDialog?.showModal();
        }
      } finally {
        this.sharing = false;
      }
    },
  },
};
</script>

<style scoped>
@media (max-width: 768px) {
  .share-btn.compact-on-mobile {
    flex: none;
    width: 3rem;
    height: 3rem;
    padding: 0;
    justify-content: center;
  }

  .compact-on-mobile .share-label {
    display: none;
  }

  .compact-on-mobile svg {
    width: 20px;
    height: 20px;
  }
}

.share-dialog {
  width: min(420px, calc(100% - 32px));
  padding: 16px;
  border: 1px solid var(--v2-outline, var(--border-color));
  background: var(--v2-bg, var(--bg-primary));
  color: var(--v2-text, var(--text-primary));
}

.share-dialog::backdrop {
  background: rgb(0 0 0 / 45%);
}

.share-dialog label,
.share-dialog textarea {
  display: block;
  width: 100%;
}

.share-dialog textarea {
  margin: 8px 0 12px;
  padding: 8px;
  color: inherit;
  background: var(--v2-surface, var(--bg-secondary));
  border: 1px solid var(--v2-outline, var(--border-color));
  resize: vertical;
}
</style>
