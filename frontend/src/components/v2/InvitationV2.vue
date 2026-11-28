<template>
  <section class="sec invitation-v2">
    <div class="sec-head"><h2>{{ $t('invite.title') }}</h2></div>
    <CollapsibleCardV2 :title="$t('invite.cardTitle')" :summary="code ? $t('invite.active') : $t('invite.none')">
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"
          stroke-linejoin="round"><path d="M10 14a4 4 0 0 0 6 0l3-3a4 4 0 0 0-6-6l-1 1M14 10a4 4 0 0 0-6 0l-3 3a4 4 0 0 0 6 6l1-1" /></svg>
      </template>
      <template v-if="code">
        <p class="invite-code">{{ code }}</p>
        <p class="label invite-note">{{ $t('invite.validUntil') }}</p>
        <div class="invite-tiles withdraw-wrap">
          <ShareButton ref="share" class="invite-tile" @click="$refs.toast?.dismiss()" icon :text="shareText" :label="$t('invite.share')" />
          <button class="invite-tile" type="button" @click="copy">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"
              stroke-linejoin="round" aria-hidden="true"><rect x="8" y="8" width="12" height="12" rx="2" /><path d="M16 8V4H4v12h4" /></svg>
            <span>{{ copied ? $t('invite.copied') : $t('invite.copy') }}</span>
          </button>
          <button class="invite-tile" type="button" :disabled="saving"
            :aria-expanded="asking.toString()" @click="ask">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"
              stroke-linejoin="round" aria-hidden="true"><path d="M20 12a8 8 0 1 1-2.3-5.7M20 4v5h-5" /></svg>
            <span>{{ $t('invite.renew') }}</span>
          </button>
          <div v-if="asking" class="confirm-pop" role="dialog" aria-labelledby="invite-confirm-title">
            <h3 id="invite-confirm-title">{{ $t('invite.renewTitle') }}</h3>
            <p class="ref">{{ $t('invite.renewNote') }}</p>
            <div class="confirm-actions actions">
              <button class="btn btn-ghost" type="button" @click="cancel">{{ $t('common.cancel') }}</button>
              <button class="btn btn-ghost" type="button" :disabled="saving" @click="renew">{{ $t('invite.renew') }}</button>
            </div>
          </div>
        </div>
      </template>
      <div v-else class="invite-empty">
        <p class="scope">{{ $t('invite.empty') }}</p>
        <button class="btn btn-ghost" type="button" :disabled="saving" @click="renew">{{ $t('invite.make') }}</button>
      </div>
    </CollapsibleCardV2>
  </section>
  <div v-if="asking" class="confirm-scrim" @click="cancel"></div>
  <div v-if="asking" class="confirm-sheet" role="dialog" aria-labelledby="invite-sheet-title">
    <h3 id="invite-sheet-title">{{ $t('invite.renewTitle') }}</h3>
    <p class="ref">{{ $t('invite.renewNote') }}</p>
    <div class="confirm-actions">
      <button class="btn btn-ghost" type="button" :disabled="saving" @click="renew">{{ $t('invite.renew') }}</button>
      <button class="btn btn-ghost" type="button" @click="cancel">{{ $t('common.cancel') }}</button>
    </div>
  </div>
  <AppToast ref="toast" />
</template>

<script>
import AppToast from '@/components/AppToast.vue';
import ShareButton from '@/components/ShareButton.vue';
import CollapsibleCardV2 from './CollapsibleCardV2.vue';
import { rotateGroupInvitation } from '@/services/api';
import { actionErrorMessage } from '@/services/v2Match';

const COPIED_MS = 2000;

// Design v2 invitation block for group settings. The v1 modal keeps
// InvitationActions.vue; v2 can only renew the code.
export default {
  name: 'InvitationV2',
  components: { AppToast, CollapsibleCardV2, ShareButton },
  props: {
    groupId: { type: String, required: true },
    code: { type: String, default: '' },
    shareText: { type: String, default: '' },
  },
  emits: ['updated', 'confirming'],
  data() {
    return { saving: false, asking: false, copied: false, copiedTimer: null };
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
    ask() {
      this.asking = true;
    },
    cancel() {
      this.asking = false;
    },
    resetCopied() {
      clearTimeout(this.copiedTimer);
      this.copiedTimer = null;
      this.copied = false;
    },
    // Results go to the shared toast. ShareButton has its own toast
    // ("Message copied"), so drop it first: two toasts never stack.
    notify(message, type) {
      this.$refs.share?.$refs?.toast?.dismiss();
      this.$refs.toast?.show(message, type);
    },
    async copy() {
      try {
        await navigator.clipboard.writeText(this.code);
        this.resetCopied();
        this.copied = true;
        this.copiedTimer = setTimeout(() => {
          this.copied = false;
          this.copiedTimer = null;
        }, COPIED_MS);
      } catch {
        this.notify(this.$t('invite.copyError'), 'error');
      }
    },
    async renew() {
      if (this.saving) return;
      const groupId = this.groupId;
      // "Make a code" (no code yet) just shows the new code; only a renewal
      // of an existing code gets a success toast.
      const renewing = Boolean(this.code);
      this.saving = true;
      try {
        const invitation = await rotateGroupInvitation(groupId);
        if (this.groupId !== groupId) return;
        this.$emit('updated', invitation);
        if (renewing) this.notify(this.$t('invite.renewed'), 'success');
      } catch (error) {
        const fallback = renewing ? this.$t('invite.renewError') : this.$t('invite.makeError');
        if (this.groupId === groupId) this.notify(actionErrorMessage(error, fallback), 'error');
      } finally {
        this.saving = false;
        this.asking = false;
      }
    },
  },
};
</script>

<style scoped>
.invite-code {
  padding: 14px 16px 2px;
  font-family: var(--v2-font-label);
  font-weight: 700;
  font-size: 26px;
  line-height: 1.25;
  letter-spacing: 0.14em;
  color: var(--v2-text);
  overflow-wrap: anywhere;
  user-select: all;
}

.invite-note {
  padding: 0 16px;
}

.invite-tiles {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  padding: 14px 16px 16px;
}

@media (min-width: 600px) {
  .invite-tiles {
    grid-template-columns: repeat(3, 120px);
  }
}

/* :deep because Share is ShareButton's own <button>. */
.invite-tiles :deep(.invite-tile) {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 0;
  height: 68px;
  padding: 0 4px;
  border-radius: 10px;
  background: var(--v2-surface-2);
  color: var(--v2-text);
  font-family: var(--v2-font-label);
  font-size: 9.5px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  line-height: 1.2;
}

.invite-tiles :deep(.invite-tile svg) {
  width: 20px;
  height: 20px;
  flex: none;
}

.invite-tiles :deep(.invite-tile:focus-visible) {
  outline: 2px solid var(--v2-text);
  outline-offset: 2px;
}

.invite-tiles :deep(.invite-tile[disabled]) {
  opacity: 0.45;
}

.invite-empty {
  padding: 14px 16px 16px;
}

.invite-empty .scope {
  margin: 0 0 12px;
}
</style>
