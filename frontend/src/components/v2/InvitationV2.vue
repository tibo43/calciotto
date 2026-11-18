<template>
  <section class="sec invitation-v2">
    <div class="sec-head"><h2>Invitation</h2></div>
    <div class="invite-card">
      <template v-if="code">
        <p class="label">Invite code</p>
        <p class="invite-code">{{ code }}</p>
        <p class="label invite-note">Works until you make a new one or revoke it</p>
        <div class="invite-buttons">
          <ShareButton class="btn btn-ghost" :text="shareText" label="Share invite" />
          <button class="btn btn-ghost" type="button" @click="copy">{{ copied ? 'Copied' : 'Copy' }}</button>
        </div>
      </template>
      <template v-else>
        <p class="label">No active invite</p>
        <p class="invite-empty">Make a code to invite players.</p>
        <div class="invite-buttons">
          <button class="btn btn-ghost" type="button" :disabled="saving" @click="update('rotate')">Make a code</button>
        </div>
      </template>
      <p v-if="error" class="scope invite-error" role="alert">{{ error }}</p>
    </div>
    <div v-if="code" class="invite-links withdraw-wrap">
      <button class="invite-link" type="button" :disabled="saving"
        :aria-expanded="(asking === 'rotate').toString()" @click="ask('rotate')">
        <span aria-hidden="true">↻</span> New code
      </button>
      <button class="invite-link" type="button" :disabled="saving"
        :aria-expanded="(asking === 'revoke').toString()" @click="ask('revoke')">Revoke</button>
      <div v-if="asking" class="confirm-pop" role="dialog" aria-labelledby="invite-confirm-title">
        <h3 id="invite-confirm-title">{{ prompt.title }}</h3>
        <p class="ref">{{ prompt.text }}</p>
        <div class="confirm-actions actions">
          <button class="btn btn-ghost" type="button" @click="cancel">Cancel</button>
          <button class="btn" :class="prompt.danger ? 'btn-danger' : 'btn-ghost'" type="button" :disabled="saving"
            @click="update(asking)">{{ prompt.action }}</button>
        </div>
      </div>
    </div>
  </section>
  <div v-if="asking" class="confirm-scrim" @click="cancel"></div>
  <div v-if="asking" class="confirm-sheet" role="dialog" aria-labelledby="invite-sheet-title">
    <h3 id="invite-sheet-title">{{ prompt.title }}</h3>
    <p class="ref">{{ prompt.text }}</p>
    <div class="confirm-actions">
      <button class="btn" :class="prompt.danger ? 'btn-danger' : 'btn-ghost'" type="button" :disabled="saving"
        @click="update(asking)">{{ prompt.action }}</button>
      <button class="btn btn-ghost" type="button" @click="cancel">Cancel</button>
    </div>
  </div>
</template>

<script>
import ShareButton from '@/components/ShareButton.vue';
import { rotateGroupInvitation, revokeGroupInvitation } from '@/services/api';

const PROMPTS = {
  rotate: { title: 'Make a new code?', text: 'The current code stops working.', action: 'New code', danger: false },
  revoke: { title: 'Revoke the invite?', text: 'Nobody can join with this code anymore.', action: 'Revoke', danger: true },
};

const COPIED_MS = 2000;

// Design v2 invitation block for group settings. The v1 modal keeps
// InvitationActions.vue; both call the same two endpoints.
export default {
  name: 'InvitationV2',
  components: { ShareButton },
  props: {
    groupId: { type: String, required: true },
    code: { type: String, default: '' },
    shareText: { type: String, default: '' },
  },
  emits: ['updated', 'confirming'],
  data() {
    return { saving: false, error: '', asking: '', copied: false, copiedTimer: null };
  },
  computed: {
    prompt() {
      return PROMPTS[this.asking] || PROMPTS.rotate;
    },
  },
  watch: {
    asking(value) {
      this.$emit('confirming', Boolean(value));
    },
    code() {
      this.resetCopied();
    },
  },
  beforeUnmount() {
    this.resetCopied();
    if (this.asking) this.$emit('confirming', false);
  },
  methods: {
    ask(kind) {
      this.error = '';
      this.asking = kind;
    },
    cancel() {
      this.asking = '';
    },
    resetCopied() {
      clearTimeout(this.copiedTimer);
      this.copiedTimer = null;
      this.copied = false;
    },
    async copy() {
      this.error = '';
      try {
        await navigator.clipboard.writeText(this.code);
        this.resetCopied();
        this.copied = true;
        this.copiedTimer = setTimeout(() => {
          this.copied = false;
          this.copiedTimer = null;
        }, COPIED_MS);
      } catch {
        this.error = "Couldn't copy the code. Select it and copy it by hand.";
      }
    },
    async update(kind) {
      if (this.saving) return;
      const groupId = this.groupId;
      this.saving = true;
      this.error = '';
      try {
        const change = kind === 'revoke' ? revokeGroupInvitation : rotateGroupInvitation;
        const invitation = await change(groupId);
        if (this.groupId === groupId) this.$emit('updated', invitation);
      } catch {
        this.error = 'Could not update the invitation.';
      } finally {
        this.saving = false;
        this.asking = '';
      }
    },
  },
};
</script>

<style scoped>
.invite-card {
  padding: 20px;
  background: var(--v2-surface);
  border-radius: var(--v2-radius-lg, 16px);
  min-width: 0;
}

.invite-code {
  margin: 6px 0 4px;
  font-family: var(--v2-font-label);
  font-size: 28px;
  line-height: 1.2;
  letter-spacing: 0.08em;
  color: var(--v2-text);
  overflow-wrap: anywhere;
  user-select: all;
}

.invite-empty {
  margin: 6px 0 0;
  color: var(--v2-text);
}

.invite-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 20px;
}

/* :deep because Share invite is ShareButton's own <button>. Equal halves
   on a phone, capped so they don't stretch across a wide card. */
.invite-buttons :deep(.btn) {
  flex: 1 1 0;
  min-width: 0;
  max-width: 200px;
  min-height: 44px;
  padding: 0 14px;
}

.invite-error {
  margin: 12px 0 0;
}

.invite-links {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 20px;
  margin-top: 4px;
}

.invite-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 44px;
  min-width: 44px;
  padding: 0;
  color: var(--v2-text-muted);
  font-size: 15px;
  font-weight: 500;
}

.invite-link:hover {
  color: var(--v2-text);
}

.invite-link[disabled] {
  opacity: 0.45;
}
</style>
