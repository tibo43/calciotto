<template>
  <section class="sec">
    <div class="sec-head"><h2>{{ $t('group.details.title') }}</h2></div>
    <form class="group-details-form" @submit.prevent="save" @input="feedback = null">
      <div class="field">
        <label class="label" for="group-description">{{ $t('group.details.description') }}</label>
        <textarea id="group-description" v-model="draft.description" rows="3" maxlength="1000" :disabled="saving"></textarea>
      </div>
      <div class="field">
        <label class="label" for="group-website-url">{{ $t('group.details.linkAddress') }}</label>
        <input id="group-website-url" v-model.trim="draft.website_url" type="url"
          placeholder="https://example.com" maxlength="2048" :disabled="saving">
      </div>
      <div class="field">
        <label class="label" for="group-website-label">{{ $t('group.details.linkLabel') }}</label>
        <input id="group-website-label" v-model="draft.website_label" type="text"
          :placeholder="$t('group.details.linkLabelPlaceholder')" maxlength="100" :disabled="saving || !draft.website_url">
      </div>
      <div class="group-details-actions">
        <button class="btn" type="submit" :disabled="saving">{{ saving ? $t('common.saving') : $t('group.details.save') }}</button>
        <p v-if="feedback" :role="feedback.error ? 'alert' : 'status'"
          :class="{ 'field-error': feedback.error }">{{ feedback.message }}</p>
      </div>
    </form>
  </section>
</template>

<script>
import { updateGroupDetails } from '@/services/api';
import { apiErrorMessage } from '@/services/apiError';

function detailsOf(group) {
  return {
    description: group.description || '',
    website_url: group.website_url || '',
    website_label: group.website_label || '',
  };
}

export default {
  name: 'GroupDetailsForm',
  props: { group: { type: Object, required: true } },
  emits: ['saved'],
  data() {
    return { draft: detailsOf(this.group), saving: false, feedback: null };
  },
  methods: {
    async save() {
      if (this.saving) return;
      this.saving = true;
      this.feedback = null;
      try {
        const group = await updateGroupDetails(this.group.id, this.draft);
        this.draft = detailsOf(group);
        this.$emit('saved', group);
        this.feedback = { message: this.$t('group.details.saved') };
      } catch (error) {
        this.feedback = { error: true, message: apiErrorMessage(error, this.$t('group.details.saveError')) };
      } finally {
        this.saving = false;
      }
    },
  },
};
</script>

<style scoped>
.group-details-form textarea {
  display: block;
  width: 100%;
  min-height: 84px;
  padding: 10px 12px;
  border: 1px solid var(--v2-outline);
  background: var(--v2-surface);
  color: var(--v2-text);
  font: inherit;
  resize: vertical;
}

.group-details-form textarea:focus {
  border-color: var(--v2-accent);
}

.group-details-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
}

.group-details-actions p {
  margin: 0;
}
</style>
