<template>
  <section class="sec passkeys-v2">
    <div class="sec-head"><h2>Passkeys</h2></div>
    <p class="scope">Sign in with Face ID, your fingerprint or your screen lock.</p>
    <div v-if="loading" class="pk-loading" role="status" aria-label="Loading passkeys">
      <div class="skel pk-skel"></div>
      <div class="skel pk-skel"></div>
    </div>
    <template v-else-if="!loaded">
      <p class="scope" role="alert">{{ error }}</p>
      <p><button class="btn btn-ghost" type="button" @click="loadKeys">Try again</button></p>
    </template>
    <template v-else>
      <p v-if="keys.length === 0" class="scope pk-empty">No passkeys yet.</p>
      <ul v-else class="pk-list">
        <li v-for="key in keys" :key="key.id" class="pk-row">
          <div class="pk-id">
            <p class="pk-name">{{ key.name }}</p>
            <p class="label pk-meta">
              <span class="pk-date">Added {{ formatDate(key.created_at) }}</span><template v-if="key.last_used_at"><span class="pk-sep"> · </span><span class="pk-date">Last used {{ formatDate(key.last_used_at) }}</span></template>
            </p>
          </div>
          <div class="withdraw-wrap pk-remove">
            <button class="btn btn-ghost" type="button" :aria-label="'Remove ' + key.name"
              :aria-expanded="(removing === key.id).toString()" :disabled="busy" @click="askRemove(key.id)">Remove</button>
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
      <p v-else-if="supported"><button class="btn btn-ghost" type="button" :disabled="busy" @click="openForm">Add a passkey</button></p>
      <p v-else class="scope pk-unsupported">Open Calciotto in a browser that supports passkeys to add one.</p>
    </template>
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
    formatDate(value) {
      return formatPasskeyDate(value);
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
.pk-skel {
  height: 56px;
  margin: 0 0 8px;
}

.pk-list {
  margin: 0 0 16px;
}

.pk-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  min-height: 64px;
  padding: 10px 0;
  border-bottom: 1px solid var(--v2-rule);
}

.pk-id {
  min-width: 0;
}

.pk-name {
  font-weight: 600;
  overflow-wrap: anywhere;
}

.pk-meta {
  margin-top: 4px;
}

/* Wraps between "Added …" and "Last used …", never inside a date. */
.pk-meta .pk-date {
  white-space: nowrap;
}

/* Phones: "Added …" and "Last used …" on two lines, no separator. */
@media (max-width: 599px) {
  .pk-meta .pk-date {
    display: block;
  }

  .pk-meta .pk-sep {
    display: none;
  }

  .pk-meta .pk-date + .pk-date,
  .pk-meta .pk-sep + .pk-date {
    margin-top: 2px;
  }
}

.pk-remove .btn {
  padding: 0 14px;
}

/* The shared pop-up opens from the left edge of its anchor. The Remove
   button sits on the right of the row, so this one opens leftwards. */
.design-v2 .wrap.is-confirming .pk-remove .confirm-pop {
  left: auto;
  right: 0;
  width: 380px;
  max-width: calc(100vw - 64px);
}

.pk-feedback {
  margin: 0 0 16px;
}

.pk-form {
  padding: 20px;
  background: var(--v2-surface);
  border-radius: var(--v2-radius-lg, 16px);
  max-width: 440px;
}

.pk-form .field input {
  background: var(--v2-bg);
}

.pk-form-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 4px;
}

.pk-form-actions .btn {
  flex: 1 1 120px;
  padding: 0 14px;
}
</style>
