<template>
  <!-- Push notifications on this device. A hidden feature: Profile.vue only
       renders this for players with PUSH_FEATURE. While they are on, the
       card lists each notification so the player can turn types off. -->
  <div class="notifications-card card-base" :class="{ 'is-expanded': showPreferences }">
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
    <div v-if="showPreferences" class="notifications-preferences">
      <section v-for="group in groups" :key="group.label" class="notifications-group">
        <h4 class="notifications-group-title">{{ group.label }}</h4>
        <div v-for="item in group.items" :key="item.type" class="notifications-pref">
          <span :id="'notification-pref-' + item.type" class="notifications-pref-label">{{ item.label }}</span>
          <button type="button" role="switch" class="notifications-switch"
            :class="{ 'is-on': preferences[item.type] !== false }"
            :aria-checked="(preferences[item.type] !== false).toString()"
            :aria-labelledby="'notification-pref-' + item.type"
            :disabled="preferenceBusy !== ''"
            @click="togglePreference(item.type)">
            <span class="notifications-switch-knob" />
          </button>
        </div>
      </section>
    </div>
  </div>
</template>

<script>
import {
  PUSH_STATE, getPushState, enablePush, disablePush, resyncPushSubscription
} from '@/services/pushNotifications';
import { getNotificationPreferences, updateNotificationPreferences } from '@/services/api';

// The six push types, in the order the card shows them. `type` matches the
// backend's PushType constants; the label is what the player reads.
const NOTIFICATION_GROUPS = [
  {
    label: 'Match',
    items: [
      { type: 'registration_open', label: 'Sign-ups open' },
      { type: 'team_assigned', label: 'Your team' },
      { type: 'result_recorded', label: 'Final score' }
    ]
  },
  {
    label: 'Predictions',
    items: [
      { type: 'predictions_open', label: 'Predictions open' },
      { type: 'predictions_settled', label: 'Predictions settled' }
    ]
  },
  {
    label: 'Reminder',
    items: [
      { type: 'kickoff_reminder', label: 'Kick-off in 30 minutes' }
    ]
  }
];

function defaultPreferences() {
  const preferences = {};
  NOTIFICATION_GROUPS.forEach((group) => {
    group.items.forEach((item) => {
      preferences[item.type] = true;
    });
  });
  return preferences;
}

// A type the server doesn't mention stays on: that is the default, and an
// older response must not look like the player turned the new type off.
function withDefaults(loaded) {
  return { ...defaultPreferences(), ...(loaded || {}) };
}

const DESCRIPTIONS = {
  [PUSH_STATE.ON]: 'On for this device. Choose which ones you get.',
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
      error: '',
      preferences: null,
      preferenceBusy: '',
      groups: NOTIFICATION_GROUPS
    };
  },
  computed: {
    canToggle() {
      return this.state === PUSH_STATE.ON || this.state === PUSH_STATE.OFF;
    },
    showPreferences() {
      return this.state === PUSH_STATE.ON && this.preferences !== null;
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
      this.preferences = defaultPreferences();
      resyncPushSubscription();
      this.loadPreferences();
    }
  },
  methods: {
    async loadPreferences() {
      this.preferenceBusy = 'loading';
      try {
        this.preferences = withDefaults(await getNotificationPreferences());
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Could not load your notification choices.';
      } finally {
        this.preferenceBusy = '';
      }
    },
    async toggle() {
      this.isBusy = true;
      this.error = '';
      const turningOn = this.state !== PUSH_STATE.ON;
      try {
        this.state = turningOn ? await enablePush() : await disablePush();
        if (this.state === PUSH_STATE.ON) {
          this.preferences = defaultPreferences();
          await this.loadPreferences();
        } else {
          this.preferences = null;
        }
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Something went wrong. Please try again.';
      } finally {
        this.isBusy = false;
      }
    },
    async togglePreference(type) {
      if (this.preferenceBusy) {
        return;
      }
      const next = this.preferences[type] === false;
      const previous = this.preferences[type];
      this.preferences = { ...this.preferences, [type]: next };
      this.preferenceBusy = type;
      this.error = '';
      try {
        this.preferences = withDefaults(await updateNotificationPreferences({ [type]: next }));
      } catch (error) {
        this.preferences = { ...this.preferences, [type]: previous };
        this.error = error.response?.data?.error || error.message || 'Something went wrong. Please try again.';
      } finally {
        this.preferenceBusy = '';
      }
    }
  }
};
</script>

<style scoped>
/* Same shape as InstallAppCard.vue: icon, text, a control on the right.
   The per-notification list is a second row, only while they are on. */
.notifications-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.25rem;
}

.notifications-card.is-expanded {
  flex-wrap: wrap;
  row-gap: 0.85rem;
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

.notifications-preferences {
  flex: 1 0 100%;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  border-top: 1px solid var(--border-color);
  padding-top: 0.85rem;
}

.notifications-group {
  margin: 0;
}

.notifications-group-title {
  margin: 0 0 0.15rem;
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--text-secondary);
}

.notifications-pref {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  min-height: 32px;
}

.notifications-pref-label {
  min-width: 0;
  font-size: 0.9rem;
  color: var(--text-primary);
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
