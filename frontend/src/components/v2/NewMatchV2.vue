<template>
  <main class="page">
    <section v-if="loading" class="state">
      <div class="skel md"></div>
      <div class="skel lg"></div>
    </section>

    <form v-else class="new-match" novalidate @submit.prevent="create">
      <section class="sec">
        <div class="sec-head"><h2>{{ $t('newMatch.kickoff') }}</h2></div>
        <div class="form-row">
          <div class="field">
            <label class="label" for="new-match-day">{{ $t('newMatch.day') }}</label>
            <input id="new-match-day" v-model="day" type="date" :min="today">
          </div>
          <div class="field">
            <label class="label" for="new-match-time">{{ $t('newMatch.time') }}</label>
            <input id="new-match-time" v-model="time" type="time">
          </div>
        </div>
        <p v-if="errorFor('day') || errorFor('time')" class="field-error">{{ errorFor('day') || errorFor('time') }}</p>
      </section>

      <section class="sec">
        <div class="sec-head"><h2>{{ $t('newMatch.signups') }}</h2></div>
        <div class="field">
          <label class="label" for="new-match-opens">{{ $t('newMatch.opensOn') }}</label>
          <input id="new-match-opens" v-model="opensAt" type="datetime-local">
          <p class="hint">{{ $t('newMatch.opensHint') }}</p>
          <p v-if="errorFor('opensAt')" class="field-error">{{ errorFor('opensAt') }}</p>
        </div>
        <div class="field">
          <label class="label" for="new-match-places">{{ $t('newMatch.places') }}</label>
          <input id="new-match-places" v-model="maxPlayers" type="number" min="1" step="1" inputmode="numeric">
          <p class="hint">{{ $t('newMatch.placesHint') }}</p>
          <p v-if="errorFor('maxPlayers')" class="field-error">{{ errorFor('maxPlayers') }}</p>
        </div>
        <div class="pref-row">
          <div>
            <div class="who">{{ $t('newMatch.signMeUp') }}</div>
            <div class="note">{{ $t('newMatch.signMeUpNote') }}</div>
          </div>
          <button class="switch" :class="{ 'is-on': registerCreator }" type="button" role="switch"
            :aria-checked="registerCreator.toString()" :aria-label="$t('newMatch.signMeUp')" @click="registerCreator = !registerCreator">
            <span class="track"></span>
          </button>
        </div>
      </section>

      <p v-if="formError" class="field-error">{{ formError }}</p>
      <div class="actions">
        <button class="btn" type="submit" :disabled="creating">{{ creating ? $t('newMatch.creating') : $t('newMatch.create') }}</button>
      </div>
    </form>
  </main>
</template>

<script>
import { createMatch } from '@/services/api';
import { resolveActiveGroup } from '@/services/activeGroup';
import { buildNewMatch, localDateTimeValue, DEFAULT_MAX_PLAYERS } from '@/services/v2NewMatch';

// Creating a match in design v2: a page of its own, reached from the home
// page's "New match" button, for an admin of the active group only (the
// backend's POST /matches is admin-gated too). Scheduled matches only: an
// already played match isn't recorded from here.
export default {
  name: 'NewMatchV2',
  inject: {
    v2Context: { default: () => ({}) },
  },
  data() {
    const now = new Date();
    return {
      loading: true,
      groupId: '',
      today: localDateTimeValue(now).slice(0, 10),
      day: '',
      time: '',
      opensAt: localDateTimeValue(now),
      maxPlayers: String(DEFAULT_MAX_PLAYERS),
      registerCreator: false,
      invalid: null,
      formError: '',
      creating: false,
    };
  },
  async created() {
    this.v2Context.hide = true;
    this.v2Context.pageTitle = this.$t('newMatch.title');
    this.v2Context.backTo = '/';
    try {
      const { groups, activeGroupId } = await resolveActiveGroup();
      const group = groups.find((item) => item.id === activeGroupId);
      if (!group || group.role !== 'admin') {
        this.$router.replace('/');
        return;
      }
      this.groupId = group.id;
    } catch (error) {
      console.error('Error resolving the active group:', error);
      this.$router.replace('/');
      return;
    }
    this.loading = false;
  },
  beforeUnmount() {
    this.v2Context.hide = false;
    this.v2Context.pageTitle = '';
    this.v2Context.backTo = '';
  },
  methods: {
    errorFor(field) {
      return this.invalid && this.invalid.field === field ? this.invalid.error : '';
    },
    async create() {
      if (this.creating) return;
      this.formError = '';
      const built = buildNewMatch({
        day: this.day,
        time: this.time,
        opensAt: this.opensAt,
        maxPlayers: this.maxPlayers,
        registerCreator: this.registerCreator,
      });
      if (built.error) {
        this.invalid = built;
        return;
      }
      this.invalid = null;
      this.creating = true;
      try {
        const id = await createMatch(built.matchData, this.groupId, built.scheduling);
        this.$router.push({ name: 'MatchView', params: { id } });
      } catch (error) {
        console.error('Error creating match:', error);
        this.formError = error?.response?.data?.error || this.$t('newMatch.createError');
      } finally {
        this.creating = false;
      }
    },
  },
};
</script>
