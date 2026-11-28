<template>
  <div class="lang-selector">
    <button :id="labelId" class="pref-row lang-toggle" type="button" :aria-expanded="open.toString()"
      :aria-controls="listId" @click="toggle" @keydown.esc="close">
      <span class="who">{{ $t('language.label') }}</span>
      <span class="lang-current">
        <span :lang="current.code">{{ current.name }}</span>
        <svg class="lang-chevron" :class="{ 'is-open': open }" width="16" height="16" viewBox="0 0 24 24" fill="none"
          stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
      </span>
    </button>
    <!-- The phone's language first, a rule, then the other languages. Picking
         one applies it at once: no Save button, no toast. -->
    <div v-if="open" :id="listId" class="lang-list" role="radiogroup" :aria-labelledby="labelId" @keydown="onKey">
      <template v-for="(entry, index) in entries" :key="entry.code">
        <div v-if="index === 1" class="lang-rule" aria-hidden="true"></div>
        <button :ref="(el) => setRef(el, index)" class="lang-option" type="button" role="radio" :lang="entry.code"
          :aria-checked="(entry.code === current.code).toString()" :tabindex="entry.code === current.code ? 0 : -1"
          @click="pick(entry)">
          <span>{{ entry.name }}</span>
          <svg v-if="entry.code === current.code" class="lang-check" width="18" height="18" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12.5l4.5 4.5L19 7.5" /></svg>
        </button>
      </template>
    </div>
  </div>
</template>

<script>
import { LOCALES } from '@/locales';
import { availableLocales, detectLocale, followPhone, localeState, setLocale, SOURCE_PHONE } from '@/services/locale';

let nextId = 0;

// The Account language selector (SPEC: Account > Appearance, under Dark mode).
// The first line is the phone's language alone (English when the phone speaks
// something the app does not), choosing it means "follow the phone"
// (players.locale NULL); then a rule, then the other languages in the usual
// order (English, Français, Italiano, Deutsch, Español). Each name is written
// in its own language and carries its own lang attribute. The saving is not
// done here: the locale service announces the change and
// services/playerLocale.js sends it. When that fails, the service puts the
// previous language back (the checkmark and the texts follow it) and the app
// shell's LocaleSaveErrorToast says so, in that previous language. A saved
// pick shows none.
export default {
  name: 'LanguageSelectorV2',
  data() {
    nextId += 1;
    return { open: false, labelId: `lang-toggle-${nextId}`, listId: `lang-list-${nextId}`, refs: [] };
  },
  computed: {
    phoneCode() {
      // Read the active language so this is worked out again when it changes.
      return localeState.locale && detectLocale();
    },
    entries() {
      const byCode = Object.fromEntries(LOCALES.map((locale) => [locale.code, locale]));
      const others = availableLocales().filter((code) => code !== this.phoneCode);
      return [this.phoneCode, ...others].map((code) => ({ code, name: byCode[code].name }));
    },
    // The checked line: the phone's one when the active language is the
    // phone's, whatever was saved.
    current() {
      return this.entries.find((entry) => entry.code === localeState.locale) || this.entries[0];
    },
  },
  methods: {
    toggle() {
      this.open = !this.open;
    },
    close() {
      if (!this.open) return;
      this.open = false;
      this.$nextTick(() => document.getElementById(this.labelId)?.focus());
    },
    setRef(el, index) {
      if (el) this.refs[index] = el;
    },
    pick(entry) {
      if (entry.code === this.current.code && (entry.code !== this.phoneCode || localeState.source === SOURCE_PHONE)) return;
      if (entry === this.entries[0]) {
        if (localeState.source !== SOURCE_PHONE) followPhone();
      } else {
        setLocale(entry.code);
      }
    },
    // Arrow keys move through the group and pick as they go (a radio group),
    // wrapping at both ends; Home and End jump to the first and last.
    onKey(event) {
      const last = this.entries.length - 1;
      const at = Math.max(0, this.entries.findIndex((entry) => entry.code === this.current.code));
      let to = null;
      if (event.key === 'ArrowDown' || event.key === 'ArrowRight') to = at === last ? 0 : at + 1;
      else if (event.key === 'ArrowUp' || event.key === 'ArrowLeft') to = at === 0 ? last : at - 1;
      else if (event.key === 'Home') to = 0;
      else if (event.key === 'End') to = last;
      else if (event.key === 'Escape') { event.stopPropagation(); this.close(); return; }
      if (to === null) return;
      event.preventDefault();
      this.pick(this.entries[to]);
      this.$nextTick(() => this.refs[to]?.focus());
    },
  },
};
</script>

<style scoped>
.lang-toggle {
  width: 100%;
  color: inherit;
  text-align: left;
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--v2-rule);
}

.lang-current {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--v2-text-muted);
}

.lang-chevron {
  flex: none;
  transition: transform 0.15s ease;
}

.lang-chevron.is-open {
  transform: rotate(180deg);
}

.lang-list {
  border-bottom: 1px solid var(--v2-rule);
}

.lang-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  min-height: 44px;
  padding: 0 0 0 12px;
  color: inherit;
  text-align: left;
  background: transparent;
  border: 0;
}

.lang-option[aria-checked='true'] {
  font-weight: 600;
}

.lang-check {
  flex: none;
  margin-right: 2px;
  color: var(--v2-accent);
}

.lang-rule {
  height: 1px;
  margin: 4px 0 4px 12px;
  background: var(--v2-rule);
}

@media (prefers-reduced-motion: reduce) {
  .lang-chevron {
    transition: none;
  }
}
</style>
