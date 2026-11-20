<template>
  <!-- Renders nothing once installed, or where the browser can't install
       (see pwaInstall.js): there is nothing to offer there. -->
  <div v-if="isOffered" class="install-app-card card-base">
    <div class="install-app-icon" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="6" y="2" width="12" height="20" rx="2" ry="2" />
        <line x1="11" y1="18" x2="13" y2="18" />
      </svg>
    </div>
    <div class="install-app-text">
      <h3 class="install-app-title">Install Calciotto</h3>
      <p v-if="state === 'prompt'" class="install-app-description">
        Open it from your home screen, like any other app.
      </p>
      <p v-else class="install-app-description">
        Tap
        <svg class="install-app-share" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
          aria-label="Share">
          <path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8" />
          <polyline points="16 6 12 2 8 6" />
          <line x1="12" y1="2" x2="12" y2="15" />
        </svg>
        <strong>Share</strong>, then <strong>Add to Home Screen</strong>.
      </p>
    </div>
    <button v-if="state === 'prompt'" class="btn-base btn-small btn-primary" @click="install">
      Install
    </button>
  </div>
</template>

<script>
import { getInstallTracker, INSTALL_STATE } from '@/services/pwaInstall';

export default {
  name: 'InstallAppCard',
  data() {
    return {
      state: INSTALL_STATE.UNAVAILABLE
    };
  },
  computed: {
    isOffered() {
      return this.state === INSTALL_STATE.PROMPT || this.state === INSTALL_STATE.IOS;
    }
  },
  created() {
    const tracker = getInstallTracker();
    this.state = tracker.state();
    this.unsubscribe = tracker.subscribe((state) => {
      this.state = state;
    });
  },
  beforeUnmount() {
    this.unsubscribe();
  },
  methods: {
    install() {
      getInstallTracker().prompt();
    }
  }
};
</script>

<style scoped>
/* Icon, text, then a small button on the right — the same shape as the
   danger zone below it, so the bottom of the profile reads as one list of
   account-level actions. */
.install-app-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.25rem;
}

.install-app-icon {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background-color: rgba(16, 185, 129, 0.12);
  color: var(--primary-color);
}

.install-app-icon svg {
  width: 22px;
  height: 22px;
}

.install-app-text {
  flex: 1;
  min-width: 0;
}

.install-app-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 0.15rem;
}

.install-app-description {
  color: var(--text-secondary);
  font-size: 0.85rem;
  margin: 0;
}

.install-app-share {
  width: 15px;
  height: 15px;
  vertical-align: -2px;
  margin: 0 0.1rem;
}

.install-app-card .btn-base {
  flex-shrink: 0;
  /* Undo global-styles.css's mobile `.btn-base { width: 100% }` and its
     padding, like the danger zone's button. */
  width: auto;
  padding: 0.5rem 0.9rem;
  white-space: nowrap;
}
</style>
