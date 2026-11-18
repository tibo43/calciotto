<template>
  <main class="player-guide" aria-labelledby="player-guide-title" @keydown.esc="dismiss">
    <div class="guide-card">
      <div class="guide-preview">
        <img :src="imageSrc" alt="" width="390" height="560" draggable="false">
      </div>
      <section class="guide-panel">
        <div class="guide-panel-heading">
          <p class="label">Step {{ index + 1 }} of {{ steps.length }}</p>
          <button v-if="!required" class="guide-close" type="button" @click="dismiss">Close guide</button>
        </div>
        <h1 id="player-guide-title" ref="title" class="guide-title" tabindex="-1">{{ step.title }}</h1>
        <p class="guide-description">{{ step.description }}</p>
        <p v-if="error" class="field-error" role="alert">{{ error }}</p>
        <div class="guide-actions">
          <button v-if="index > 0" class="btn btn-ghost" type="button" :disabled="saving" @click="changeStep(-1)">Back</button>
          <button class="btn" type="button" :disabled="saving" @click="advance">
            {{ saving ? 'Saving…' : actionLabel }}
          </button>
        </div>
      </section>
    </div>
  </main>
</template>

<script>
const steps = [
  {
    title: 'Welcome to Calciotto',
    description: 'Find your next match and keep up with your group’s teams, scores, and standings.',
    image: 'welcome.jpg',
  },
  {
    title: 'Matches',
    description: 'Tap Sign up when places are open. Come back to see your team and the final score.',
    image: 'matches.jpg',
  },
  {
    title: 'Rankings',
    description: 'Points tracks match results. Scorers lists goals. MOTM counts Man of the Match awards from post-match votes.',
    image: 'rankings.jpg',
  },
  {
    title: 'Predictions',
    description: 'Pick a result while predictions are open. Check the leaderboard for your points after the match.',
    image: 'predictions.jpg',
  },
  {
    title: 'Profile',
    description: 'See your record and groups, manage your settings, and replay this guide whenever you need it.',
    image: 'profile.jpg',
  },
];

export default {
  name: 'PlayerGuide',
  props: {
    dark: Boolean,
    required: Boolean,
    saving: Boolean,
    error: { type: String, default: '' },
  },
  emits: ['finish', 'dismiss'],
  data() {
    return { steps, index: 0 };
  },
  computed: {
    step() {
      return this.steps[this.index];
    },
    imageSrc() {
      const theme = this.dark ? 'dark' : 'light';
      return `/guide/${theme}/${this.step.image}`;
    },
    isLast() {
      return this.index === this.steps.length - 1;
    },
    actionLabel() {
      if (!this.isLast) return 'Next';
      return this.required ? 'Get started' : 'Done';
    },
  },
  mounted() {
    this.$refs.title.focus();
  },
  methods: {
    async changeStep(offset) {
      this.index += offset;
      await this.$nextTick();
      this.$refs.title?.focus();
    },
    advance() {
      if (this.isLast) {
        this.$emit('finish');
      } else {
        this.changeStep(1);
      }
    },
    dismiss() {
      if (!this.required) this.$emit('dismiss');
    },
  },
};
</script>

<style scoped>
.player-guide {
  min-height: 100dvh;
  display: grid;
  place-items: center;
  padding: 16px 12px;
  background: var(--v2-bg);
  color: var(--v2-text);
}

.guide-card {
  width: min(100%, 480px);
  overflow: hidden;
  border: 1px solid var(--v2-outline);
  border-radius: 18px;
  background: var(--v2-surface);
}

.guide-preview {
  height: clamp(230px, 54svh, 490px);
  overflow: hidden;
  padding-bottom: 14px;
  background: var(--v2-bg);
}

.guide-preview img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.guide-panel {
  position: relative;
  margin-top: -14px;
  padding: 18px 20px 20px;
  border-radius: 14px 14px 0 0;
  background: var(--v2-surface);
}

.guide-panel-heading,
.guide-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.guide-panel-heading {
  justify-content: space-between;
}

.guide-panel-heading p {
  margin: 0;
}

.guide-close {
  min-height: 44px;
  border: 0;
  padding: 0;
  background: transparent;
  color: inherit;
  text-decoration: underline;
}

.guide-title {
  margin: 7px 0 6px;
  outline: none;
  font-family: var(--v2-font-display);
  font-size: clamp(38px, 7vw, 60px);
  line-height: 1;
  font-weight: 800;
}

.guide-description {
  margin: 0;
  font-size: 16px;
  line-height: 1.5;
}

.guide-actions {
  justify-content: flex-end;
  margin-top: 18px;
}

.guide-actions .btn {
  min-width: 100px;
  min-height: 44px;
}

@media (min-width: 760px) {
  .player-guide {
    padding: 32px;
  }

  .guide-card {
    display: grid;
    grid-template-columns: minmax(0, 390px) minmax(0, 1fr);
    width: min(100%, 830px);
  }

  .guide-preview {
    height: clamp(430px, 74svh, 560px);
    padding-bottom: 0;
  }

  .guide-panel {
    align-self: center;
    margin-top: 0;
    padding: 36px;
    border-radius: 0;
  }
}
</style>
