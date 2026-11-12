<template>
  <!-- Push notifications on this device. A hidden feature: Profile.vue only
       renders this for players with PUSH_FEATURE. -->
  <div class="notifications-card card-base">
    <div class="notifications-icon" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
        <path d="M13.73 21a2 2 0 0 1-3.46 0" />
      </svg>
    </div>
    <div class="notifications-text">
      <h3 class="notifications-title">Notifications</h3>
      <p class="notifications-description">{{ description }}</p>
      <p v-if="error" class="error-message">{{ error }}</p>
    </div>
    <button v-if="canToggle" type="button" role="switch" class="notifications-switch"
      :class="{ 'is-on': state === 'on' }" :aria-checked="(state === 'on').toString()" :disabled="isBusy"
      aria-label="Notifications on this device" @click="toggle">
      <span class="notifications-switch-knob" />
    </button>
  </div>
</template>

<script>
import {
  PUSH_STATE, getPushState, enablePush, disablePush, resyncPushSubscription
} from '@/services/pushNotifications';

const DESCRIPTIONS = {
  [PUSH_STATE.ON]: 'On for this device: sign-ups, your team, kick-off and the result.',
  [PUSH_STATE.OFF]: 'Get told about sign-ups, your team, kick-off and the result.',
  [PUSH_STATE.NEEDS_INSTALL]: 'On iPhone, install the app first (Share, then Add to Home Screen) and turn them on from there.',
  [PUSH_STATE.DENIED]: 'Blocked for Calciotto. Allow notifications in your phone or browser settings, then come back here.',
  [PUSH_STATE.UNSUPPORTED]: "This browser can't receive notifications."
};

export default {
  name: 'NotificationsCard',
  data() {
    return {
      state: null,
      isBusy: false,
      error: ''
    };
  },
  computed: {
    canToggle() {
      return this.state === PUSH_STATE.ON || this.state === PUSH_STATE.OFF;
    },
    description() {
      return this.state ? DESCRIPTIONS[this.state] : 'Checking this device…';
    }
  },
  async created() {
    try {
      this.state = await getPushState();
    } catch (error) {
      console.error('Could not read the notification state:', error);
      this.state = PUSH_STATE.UNSUPPORTED;
    }
    if (this.state === PUSH_STATE.ON) {
      resyncPushSubscription();
    }
  },
  methods: {
    async toggle() {
      this.isBusy = true;
      this.error = '';
      try {
        this.state = this.state === PUSH_STATE.ON ? await disablePush() : await enablePush();
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Something went wrong. Please try again.';
      } finally {
        this.isBusy = false;
      }
    }
  }
};
</script>

<style scoped>
/* Same shape as InstallAppCard.vue: icon, text, a control on the right. */
.notifications-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.25rem;
}

.notifications-icon {
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

.notifications-icon svg {
  width: 22px;
  height: 22px;
}

.notifications-text {
  flex: 1;
  min-width: 0;
}

.notifications-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 0.15rem;
}

.notifications-description {
  color: var(--text-secondary);
  font-size: 0.85rem;
  margin: 0;
}

.notifications-card .error-message {
  font-size: 0.85rem;
  margin: 0.35rem 0 0;
}

.notifications-switch {
  flex-shrink: 0;
  position: relative;
  width: 44px;
  height: 26px;
  padding: 0;
  border: none;
  border-radius: 13px;
  background-color: var(--border-color);
  cursor: pointer;
  transition: background-color var(--transition-fast);
}

.notifications-switch.is-on {
  background-color: var(--primary-color);
}

.notifications-switch:disabled {
  opacity: 0.6;
  cursor: wait;
}

.notifications-switch-knob {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background-color: #ffffff;
  box-shadow: var(--shadow-sm);
  transition: transform var(--transition-fast);
}

.notifications-switch.is-on .notifications-switch-knob {
  transform: translateX(18px);
}
</style>
