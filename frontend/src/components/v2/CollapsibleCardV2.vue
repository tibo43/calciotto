<template>
  <div class="cc-card" :class="{ 'is-open': open }">
    <button :id="headerId" class="cc-head" type="button" :aria-expanded="open.toString()" :aria-controls="panelId"
      @click="toggle">
      <span class="cc-chip" aria-hidden="true"><slot name="icon" /></span>
      <span class="cc-text">
        <span class="cc-title">{{ title }}</span>
        <span class="cc-summary">{{ summary }}</span>
      </span>
      <svg class="cc-chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"
        stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
    </button>
    <div :id="panelId" class="cc-panel" role="region" :aria-labelledby="headerId" :hidden="!open">
      <template v-if="open">
        <hr class="cc-rule">
        <slot />
      </template>
    </div>
  </div>
</template>

<script>
let nextId = 0;

// Design v2 card that folds its content under a header button (chip icon,
// title, one-line mono summary, chevron). Closed by default.
export default {
  name: 'CollapsibleCardV2',
  props: {
    title: { type: String, required: true },
    summary: { type: String, default: '' },
    initiallyOpen: { type: Boolean, default: false },
  },
  emits: ['toggle'],
  data() {
    nextId += 1;
    return {
      open: this.initiallyOpen,
      headerId: `cc-head-${nextId}`,
      panelId: `cc-panel-${nextId}`,
    };
  },
  methods: {
    toggle() {
      this.open = !this.open;
      this.$emit('toggle', this.open);
    },
  },
};
</script>

<style scoped>
.cc-card {
  background: var(--v2-surface);
  border-radius: var(--v2-radius-lg, 16px);
  min-width: 0;
}

.cc-head {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 64px;
  padding: 0 16px;
  color: var(--v2-text);
  text-align: left;
  border-radius: var(--v2-radius-lg, 16px);
}

.cc-head:focus-visible {
  outline: 2px solid var(--v2-text);
  outline-offset: -2px;
}

.cc-chip {
  display: grid;
  place-items: center;
  flex: none;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--v2-surface-2);
}

.cc-chip :deep(svg) {
  width: 20px;
  height: 20px;
}

.cc-text {
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-width: 0;
}

.cc-title {
  font-family: var(--v2-font-body);
  font-weight: 600;
  font-size: 15px;
  line-height: 1.3;
}

.cc-summary {
  font-family: var(--v2-font-label);
  font-size: 11px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--v2-text-muted);
  line-height: 1.4;
}

.cc-chevron {
  flex: none;
  width: 20px;
  height: 20px;
  transition: transform 0.15s ease;
}

.is-open .cc-chevron {
  transform: rotate(180deg);
}

.cc-rule {
  margin: 0;
  border: 0;
  border-top: 1px solid var(--v2-rule);
}
</style>
