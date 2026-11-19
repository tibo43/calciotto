<template>
  <section class="sec passkeys-v2">
    <div class="sec-head"><h2>Passkeys</h2></div>
    <CollapsibleCardV2 title="Passkeys" :summary="summary" @toggle="onToggle">
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"
          stroke-linejoin="round"><circle cx="8" cy="15" r="4" /><path d="M11 12l9-9M17 6l3 3M14 9l2 2" /></svg>
      </template>
      <div v-if="loading" class="pk-loading" role="status" aria-label="Loading passkeys">
        <div class="skel pk-skel"></div>
        <div class="skel pk-skel"></div>
      </div>
      <div v-else-if="!loaded" class="pk-block">
        <p class="scope" role="alert">{{ error }}</p>
        <button class="btn btn-ghost" type="button" @click="loadKeys">Try again</button>
      </div>
      <template v-else>
        <p v-if="keys.length === 0" class="scope pk-empty">Sign in with Face ID, your fingerprint or your screen lock.</p>
        <ul v-else class="pk-list">
          <li v-for="key in keys" :key="key.id" class="pk-row">
            <div class="pk-id">
              <p class="pk-name">{{ key.name }}</p>
              <p class="label pk-meta">{{ usage(key) }}</p>
            </div>
            <div class="withdraw-wrap pk-remove">
              <button class="pk-trash" type="button" :aria-label="'Remove ' + key.name"
                :aria-expanded="(removing === key.id).toString()" :disabled="busy" @click="askRemove(key.id)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"
                  stroke-linejoin="round" aria-hidden="true"><path d="M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13" /></svg>
              </button>
              <div v-if="removing === key.id" class="confirm-pop" role="dialog" aria-labelledby="passkey-remove-title">
                <h3 id="passkey-remove-title">Remove this passkey?</h3>
                <p class="ref">You won't be able to sign in with it anymore.</p>
                <div class="confirm-actions actions">
                  <button class="btn btn-ghost" type="button" @click="removing = ''">Cancel</button>
                  <button class="btn btn-danger" type="button" :disabled="busy" @click="remove(key.id)">Remove</button>
                </div>
              </div>
            </div>
          </li>
        </ul>
        <p v-if="notice" class="scope pk-feedback" role="status">{{ notice }}</p>
        <p v-if="error" class="scope pk-feedback" role="alert">{{ error }}</p>
        <form v-if="adding" class="pk-form" @submit.prevent="create">
          <div class="field">
            <label class="label" for="passkey-name">Name (optional)</label>
            <input id="passkey-name" v-model="name" type="text" maxlength="60" placeholder="My phone" :disabled="busy">
          </div>
          <div class="field">
            <label class="label" for="passkey-password">Current password</label>
            <input id="passkey-password" v-model="password" type="password" autocomplete="current-password"
              required :disabled="busy">
          </div>
          <div class="pk-form-actions">
            <button class="btn btn-ghost" type="button" :disabled="busy" @click="closeForm">Cancel</button>
            <button class="btn btn-ghost" type="submit" :disabled="busy || !password">
              {{ busy ? 'Waiting for your device…' : 'Create passkey' }}
            </button>
          </div>
        </form>
        <button v-else-if="supported" class="pk-add" type="button" :disabled="busy" @click="openForm">
          <span class="pk-add-chip" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M12 5v14M5 12h14" /></svg>
          </span>
          <span>Add a passkey</span>
        </button>
        <p v-else class="scope pk-unsupported">Open Calciotto in a browser that supports passkeys to add one.</p>
      </template>
    </CollapsibleCardV2>
  </section>
  <div v-if="removing" class="confirm-scrim" @click="removing = ''"></div>
  <div v-if="removing" class="confirm-sheet" role="dialog" aria-labelledby="passkey-remove-sheet-title">
    <h3 id="passkey-remove-sheet-title">Remove this passkey?</h3>
    <p class="ref">You won't be able to sign in with it anymore.</p>
    <div class="confirm-actions">
      <button class="btn btn-danger" type="button" :disabled="busy" @click="remove(removing)">Remove</button>
      <button class="btn btn-ghost" type="button" @click="removing = ''">Cancel</button>
    </div>
  </div>
</template>

<script>
import { getMyPasskeys, removeMyPasskey } from '@/services/api';
import { addPasskey, supportsPasskeys } from '@/services/passkeys';
import CollapsibleCardV2 from './CollapsibleCardV2.vue';

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

// "18 Nov 2026" in the browser's own time zone (the label style uppercases it).
export function formatPasskeyDate(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return `${date.getDate()} ${MONTHS[date.getMonth()]} ${date.getFullYear()}`;
}

// Design v2 passkey list for the Account page, shown only with the
// design-v2 and passkeys features (ProfileV2 checks both).
export default {
  name: 'PasskeysV2',
  components: { CollapsibleCardV2 },
  emits: ['confirming'],
  data() {
    return {
      supported: supportsPasskeys(),
      loaded: false,
      loading: false,
      busy: false,
      adding: false,
      removing: '',
      keys: [],
      name: '',
      password: '',
      error: '',
      notice: '',
    };
  },
  computed: {
    summary() {
      if (this.loading) return 'Loading…';
      if (!this.loaded) return this.error ? "Couldn't load" : '';
      if (this.keys.length === 0) return 'None yet';
      return this.keys.length === 1 ? '1 device' : `${this.keys.length} devices`;
    },
  },
  watch: {
    removing(value) {
      this.$emit('confirming', Boolean(value));
    },
  },
  created() {
    this.loadKeys();
  },
  beforeUnmount() {
    if (this.removing) this.$emit('confirming', false);
  },
  methods: {
    // One date: the last use, or when it was added if never used.
    usage(key) {
      return key.last_used_at
        ? `Last used ${formatPasskeyDate(key.last_used_at)}`
        : `Added ${formatPasskeyDate(key.created_at)}`;
    },
    onToggle(open) {
      if (open && !this.loaded && !this.loading) this.loadKeys();
      if (!open) {
        this.closeForm();
        this.removing = '';
      }
    },
    async loadKeys() {
      if (this.loading) return;
      this.loading = true;
      this.error = '';
      try {
        const keys = await getMyPasskeys();
        this.keys = Array.isArray(keys) ? keys : [];
        this.loaded = true;
      } catch {
        this.error = 'Could not load your passkeys.';
      } finally {
        this.loading = false;
      }
    },
    openForm() {
      this.notice = '';
      this.error = '';
      this.adding = true;
    },
    closeForm() {
      this.adding = false;
      this.name = '';
      this.password = '';
    },
    askRemove(id) {
      this.notice = '';
      this.error = '';
      this.removing = id;
    },
    async create() {
      if (this.busy || !this.password) return;
      const password = this.password;
      this.password = '';
      this.busy = true;
      this.error = '';
      this.notice = '';
      try {
        const key = await addPasskey(this.name, password);
        this.keys.unshift(key);
        this.closeForm();
        this.notice = 'Passkey added.';
      } catch (error) {
        this.error = error.response?.data?.error || 'Could not add a passkey. Try again.';
      } finally {
        this.busy = false;
      }
    },
    async remove(id) {
      if (this.busy || !id) return;
      this.busy = true;
      this.error = '';
      this.notice = '';
      try {
        await removeMyPasskey(id);
        this.keys = this.keys.filter((key) => key.id !== id);
        this.notice = 'Passkey removed.';
      } catch {
        this.error = 'Could not remove the passkey. Try again.';
      } finally {
        this.busy = false;
        this.removing = '';
      }
    },
  },
};
</script>

<style scoped>
.pk-loading {
  padding: 12px 16px 4px;
}

.pk-skel {
  height: 56px;
  margin: 0 0 8px;
  border-radius: 10px;
}

.pk-block {
  padding: 14px 16px 16px;
}

.pk-block .scope {
  margin: 0 0 12px;
}

.pk-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  min-height: 60px;
  padding: 0 8px 0 16px;
}

.pk-row + .pk-row {
  border-top: 1px solid var(--v2-rule);
}

.pk-id {
  min-width: 0;
}

.pk-name {
  font-weight: 600;
  overflow-wrap: anywhere;
}

.pk-meta {
  margin-top: 2px;
}

.pk-trash {
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  color: var(--v2-text-muted);
  border-radius: 10px;
}

.pk-trash svg {
  width: 20px;
  height: 20px;
}

.pk-trash:hover {
  color: var(--v2-text);
}

/* The shared pop-up opens from the left edge of its anchor. The trash
   button sits on the right of the row, so this one opens leftwards. */
.design-v2 .wrap.is-confirming .pk-remove .confirm-pop {
  left: auto;
  right: 0;
  width: 380px;
  max-width: calc(100vw - 64px);
}

.pk-empty {
  margin: 0;
  padding: 14px 16px 10px;
}

.pk-feedback {
  margin: 0;
  padding: 10px 16px;
  border-top: 1px solid var(--v2-rule);
}

.pk-add {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 56px;
  padding: 0 16px;
  color: var(--v2-text);
  font-family: var(--v2-font-body);
  font-weight: 600;
  font-size: 15px;
  text-align: left;
  border-top: 1px solid var(--v2-rule);
  border-radius: 0 0 var(--v2-radius-lg, 16px) var(--v2-radius-lg, 16px);
}

.pk-add:focus-visible,
.pk-trash:focus-visible {
  outline: 2px solid var(--v2-text);
  outline-offset: -2px;
}

.pk-add-chip {
  display: grid;
  place-items: center;
  flex: none;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: var(--v2-surface-2);
}

.pk-add-chip svg {
  width: 20px;
  height: 20px;
}

.pk-form {
  padding: 16px;
  border-top: 1px solid var(--v2-rule);
}

.pk-form .field input {
  background: var(--v2-bg);
}

.pk-form-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.pk-form-actions .btn {
  flex: 1 1 120px;
  padding: 0 14px;
}

.pk-unsupported {
  margin: 0;
  padding: 14px 16px 16px;
  border-top: 1px solid var(--v2-rule);
}
</style>
