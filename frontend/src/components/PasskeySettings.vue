<template>
  <details class="passkey-settings" @toggle="onToggle">
    <summary>Passkeys</summary>
    <div class="passkey-content">
      <p class="passkey-hint">Sign in with Face ID, your fingerprint, or your device’s screen lock.</p>
      <p v-if="error" class="passkey-error" role="alert">{{ error }}</p>
      <p v-if="notice" class="passkey-hint" role="status">{{ notice }}</p>
      <p v-if="loading" role="status">Loading passkeys…</p>
      <button v-else-if="!loaded" type="button" class="btn btn-ghost" @click="loadKeys">Try again</button>
      <template v-else>
        <p v-if="keys.length === 0" class="passkey-hint">No passkeys saved yet.</p>
        <ul v-else class="passkey-list">
          <li v-for="key in keys" :key="key.id" class="passkey-row">
            <div class="passkey-identity">
              <strong>{{ key.name }}</strong>
              <p class="passkey-hint">{{ key.last_used_at ? 'Last used' : 'Added' }} {{ formatDate(key.last_used_at || key.created_at) }}</p>
            </div>
            <button type="button" class="btn btn-ghost" :aria-label="'Remove ' + key.name"
              :disabled="busy" @click="remove(key.id)">Remove</button>
          </li>
        </ul>
        <form v-if="adding" class="passkey-form" @submit.prevent="create">
          <label for="passkey-name">Passkey name <span class="passkey-hint">(optional)</span></label>
          <input id="passkey-name" v-model="name" class="form-input" type="text" maxlength="60" placeholder="My phone" :disabled="busy">
          <label for="passkey-password">Current password</label>
          <input id="passkey-password" v-model="password" class="form-input" type="password" autocomplete="current-password"
            required :disabled="busy">
          <div class="passkey-actions">
            <button type="button" class="btn btn-ghost" :disabled="busy" @click="closeForm">Cancel</button>
            <button type="submit" class="btn" :disabled="busy || !password">
              {{ busy ? 'Waiting for your device…' : 'Create passkey' }}
            </button>
          </div>
        </form>
        <button v-else-if="supported" class="btn" type="button" :disabled="busy" @click="adding = true">Add passkey</button>
        <p v-else class="passkey-hint">Open Calciotto in a browser that supports passkeys to add one.</p>
      </template>
    </div>
  </details>
</template>

<script>
import { getMyPasskeys, removeMyPasskey } from '@/services/api';
import { addPasskey, supportsPasskeys } from '@/services/passkeys';

export default {
  name: 'PasskeySettings',
  data() {
    return {
      supported: supportsPasskeys(),
      loaded: false,
      loading: false,
      busy: false,
      adding: false,
      keys: [],
      name: '',
      password: '',
      error: '',
      notice: '',
    };
  },
  methods: {
    onToggle(event) {
      if (event.target.open && !this.loaded) this.loadKeys();
      if (!event.target.open) this.closeForm();
    },
    async loadKeys() {
      if (this.loading) return;
      this.loading = true;
      this.error = '';
      try {
        this.keys = await getMyPasskeys();
        this.loaded = true;
      } catch {
        this.error = 'Could not load your passkeys.';
      } finally {
        this.loading = false;
      }
    },
    closeForm() {
      this.adding = false;
      this.name = '';
      this.password = '';
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
      if (this.busy) return;
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
      }
    },
    formatDate(date) {
      return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(date));
    },
  },
};
</script>

<style scoped>
.passkey-settings {
  padding-block: 0.75rem;
  color: var(--v2-text);
}

.passkey-settings summary {
  padding-block: 0.5rem;
  min-height: 44px;
  cursor: pointer;
  font-weight: 700;
}

.passkey-content,
.passkey-form {
  display: grid;
  gap: 0.65rem;
  justify-items: start;
}

.passkey-content p {
  margin: 0;
}

.passkey-hint {
  color: var(--v2-text-muted);
  font-size: 0.875rem;
}

.passkey-error {
  color: var(--v2-danger);
}

.passkey-list {
  width: 100%;
  padding: 0;
  margin: 0;
  list-style: none;
}

.passkey-row,
.passkey-actions {
  display: flex;
  align-items: center;
  gap: 0.65rem;
}

.passkey-row {
  justify-content: space-between;
  border-bottom: 1px solid var(--v2-rule);
  padding: 0.5rem 0;
}

.passkey-identity {
  overflow-wrap: anywhere;
}

.passkey-form {
  width: min(100%, 390px);
}

.passkey-form input {
  width: 100%;
}

.passkey-actions {
  flex-wrap: wrap;
}
</style>
