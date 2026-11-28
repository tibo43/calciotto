<template>
  <div id="app" class="design-v2" :class="{ 'dark-mode': isDarkMode }">
    <PlayerGuide v-if="guideMode" :required="guideMode === 'required'" :saving="savingGuide"
      :dark="isDarkMode" :error="guideError" @finish="finishGuide" @dismiss="closeGuide" @skip="skipGuide" />
    <div v-else class="app-container wrap" :class="{ 'is-confirming': v2Context.confirming, 'is-picking-csc': v2Context.pickingCsc }">
        <header class="topbar">
          <div class="topbar-left">
            <template v-if="v2Context.pageTitle">
              <router-link class="back" :to="v2Context.backTo || '/profile'" :aria-label="$t('common.back')">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M15 5l-7 7 7 7"/></svg>
              </router-link>
              <h1 class="page-title">{{ v2Context.pageTitle }}</h1>
            </template>
            <template v-else>
            <router-link class="brand" to="/" :aria-label="$t('nav.home')">
              <img class="brand-mark brand-mark-dark" src="/icons/v2/header-symbole-sombre.png" width="40" height="40" alt="">
              <img class="brand-mark brand-mark-light" src="/icons/v2/header-symbole-clair.png" width="40" height="40" alt="">
            </router-link>
            <div v-if="isAuthenticatedRoute && !v2Context.hide" class="ctx-pair" :aria-label="$t('context.label')">
              <label class="ctx">
                <span class="sr-only">{{ $t('context.group') }}</span>
                <span class="ctx-name" aria-hidden="true">{{ v2GroupLabel }}</span>
                <select :value="v2Context.groupId" @change="onV2Group">
                  <option v-for="group in v2Context.groups" :key="group.id" :value="group.id">{{ group.name }}</option>
                </select>
                <svg class="chev" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M2 4l4 4 4-4"/></svg>
              </label>
              <label class="ctx" :class="{ 'is-disabled': !v2HasSeasons }">
                <span class="sr-only">{{ $t('context.season') }}</span>
                <span class="ctx-name" aria-hidden="true">{{ v2SeasonLabel }}</span>
                <select :value="v2Context.season" :disabled="!v2HasSeasons" @change="onV2Season">
                  <option v-for="season in v2Context.seasons" :key="season" :value="season">{{ season }}</option>
                </select>
                <svg v-if="v2HasSeasons" class="chev" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><path d="M2 4l4 4 4-4"/></svg>
              </label>
            </div>
            </template>
          </div>
          <GearLink v-if="v2Context.pageTitle && v2Context.pageAction" class="btn btn-ghost hero-manage page-action"
            :to="v2Context.pageAction.to" :label="v2Context.pageAction.label" />
          <nav v-if="isAuthenticatedRoute && !v2Context.pageTitle" class="dnav" :aria-label="$t('nav.main')">
            <router-link v-for="item in v2NavItems" :key="item.to" :to="item.to" class="nav-text" :class="{ 'is-on': item.on }"
              :aria-current="item.on ? 'page' : null">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="item.icon" /></svg>
              {{ item.label }}
            </router-link>
          </nav>
          <!-- No theme toggle here: in v2 the theme is a setting, on the
               Account page (ProfileV2), not a control on every page. -->
        </header>
        <!-- Main Content -->
        <main class="main-content" :class="{ 'no-top-bar-on-mobile': isAuthenticatedRoute }">
          <router-view />
        </main>

        <nav v-if="isAuthenticatedRoute" class="tabbar" :aria-label="$t('nav.main')">
          <router-link v-for="item in v2NavItems" :key="item.to" :to="item.to" class="nav-ico" :class="{ 'is-on': item.on }"
            :aria-current="item.on ? 'page' : null">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="item.icon" /></svg>
            <span class="nav-ico-label">{{ item.label }}</span>
          </router-link>
        </nav>
    </div>
    <!-- Outside the guide/app switch, so a failed language save is reported
         on any page, the guide included. -->
    <LocaleSaveErrorToast />
  </div>
</template>

<script>
import PlayerGuide from '@/components/PlayerGuide.vue';
import GearLink from '@/components/v2/GearLink.vue';
import LocaleSaveErrorToast from '@/components/LocaleSaveErrorToast.vue';
import { clearToken, completeGuide, getSessionKey, logoutSession } from '@/services/api';
import { clearActiveGroupId, clearMyGroupsCache } from '@/services/activeGroup';
import { syncThemeColorMeta, prefersDarkTheme } from '@/services/theme';
import { resetSavedLocale, syncSavedLocale } from '@/services/playerLocale';
import { disablePushOnLogout, stopPushIfAppUninstalled } from '@/services/pushNotifications';
import {
  knownFeatureStatus, loadFeatureStatus, rememberGuideCompletion, resetFeatureStatus
} from '@/services/featureStatus';

// Routes reachable without a token — nothing group-related is shown there.
const PUBLIC_ROUTE_NAMES = ['Login', 'Signup', 'ForgotPassword', 'ResetPassword'];

export default {
  name: 'App',
  components: { GearLink, LocaleSaveErrorToast, PlayerGuide },
  provide() {
    return {
      v2Context: this.v2Context,
      openGuide: this.openGuide,
    };
  },
  data() {
    return {
      isDarkMode: prefersDarkTheme(),
      guideMode: null,
      savingGuide: false,
      guideError: '',
      v2Context: { label: '', confirming: false, pickingCsc: false, hide: false, pageTitle: '', backTo: '', pageAction: null, groups: [], seasons: [], groupId: '', season: '', onGroup: null, onSeason: null },
      // Bumped on logout, so a guide answer for the previous session is
      // ignored when it lands.
      guideRequest: 0
    };
  },
  computed: {
    // Gates the nav-menu links (Matches/Groups) and the Profile/Logout icons:
    // all three require a token, so showing them on a public route (where
    // there's no session to act on) makes a standalone login/signup page
    // read as the logged-in app shell instead of its own view.
    isAuthenticatedRoute() {
      return this.isAuthenticatedRouteName(this.$route.name);
    },
    isV2Home() {
      // Only the home itself: a match page is not a tab of the menu, so it
      // highlights none (the same for the edit and new match pages).
      return this.$route.name === 'MatchesAndStandings';
    },
    // What the group select shows (see .ctx-name in design-v2-ui.css).
    v2GroupLabel() {
      const group = (this.v2Context.groups || []).find((item) => item.id === this.v2Context.groupId);
      return group ? group.name : '';
    },
    v2HasSeasons() {
      return (this.v2Context.seasons || []).length > 0;
    },
    // Keep the season control in the same place when a group has no
    // matches yet, so the header does not jump. The visible label is
    // "No season yet" and the select is disabled.
    v2SeasonLabel() {
      return this.v2HasSeasons ? (this.v2Context.season || '') : this.$t('context.noSeason');
    },
    isV2Profile() {
      return String(this.$route.path || '').startsWith('/profile');
    },
    // The v2 menu: icon and name in the desktop top bar, icon and name in
    // the phone tab bar. The visible label is the accessible name.
    v2NavItems() {
      return [
        { to: '/', label: this.$t('nav.matches'), on: this.isV2Home, icon: 'M3 9a9 3 0 1 0 18 0 9 3 0 1 0-18 0zM3 9v6a9 3 0 0 0 18 0V9M7 9a5 1.4 0 1 0 10 0 5 1.4 0 1 0-10 0zM7.5 17.6v-3M12 18v-3M16.5 17.6v-3' },
        { to: '/rankings', label: this.$t('nav.rankings'), on: this.$route.name === 'Rankings', icon: 'M8 21h8M12 17v4M7 4h10v5a5 5 0 0 1-10 0V4zM7 6H4v2a3 3 0 0 0 3 3M17 6h3v2a3 3 0 0 1-3 3' },
        { to: '/predictions', label: this.$t('nav.predictions'), on: this.$route.name === 'Predictions', icon: 'M11 12a1 1 0 1 0 2 0 1 1 0 1 0-2 0M12 7a5 5 0 1 0 5 5M13 3.055a9 9 0 1 0 7.941 7.945M15 6v3h3l3-3h-3V3zM15 9l-3 3' },
        { to: '/profile', label: this.$t('nav.profile'), on: this.isV2Profile, icon: 'M12 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM4 21a8 8 0 0 1 16 0' },
      ];
    }
  },
  watch: {
    isDarkMode(isDark) {
      syncThemeColorMeta(isDark);
    },
    '$route.name'() {
      this.syncGuide();
      this.syncLocale();
    }
  },
  created() {
    this.syncGuide();
    this.syncLocale();
  },
  mounted() {
    // Android keeps an uninstalled app's notifications going through Chrome;
    // this turns them off when the site is opened again in a tab (see
    // stopPushIfAppUninstalled). Fire and forget: it never blocks the page.
    stopPushIfAppUninstalled();

    syncThemeColorMeta(this.isDarkMode);
  },
  methods: {
    toggleTheme() {
      this.isDarkMode = !this.isDarkMode;
      localStorage.setItem('calciotto-theme', this.isDarkMode ? 'dark' : 'light');
    },
    onV2Group(event) {
      if (this.v2Context.onGroup) this.v2Context.onGroup(event.target.value);
    },
    onV2Season(event) {
      if (this.v2Context.onSeason) this.v2Context.onSeason(event.target.value);
    },
    isAuthenticatedRouteName(name) {
      // A missing name is the router's start location, before the real
      // route has matched. That is not an authenticated page — treating it
      // as one would flash the redesign onto login.
      return Boolean(name) && !PUBLIC_ROUTE_NAMES.includes(name);
    },
    // The first-run guide is the one thing the shell still asks the server
    // (GET /players/me/features, once per session). It never holds the page:
    // the app paints straight away and the guide opens when the answer says
    // the player has not done it yet.
    async syncGuide() {
      if (!this.isAuthenticatedRoute) {
        if (this.guideMode === 'required') this.guideMode = null;
        return;
      }
      const requestId = this.guideRequest;
      const sessionKey = getSessionKey();
      const status = knownFeatureStatus() ?? await loadFeatureStatus();
      if (requestId !== this.guideRequest || sessionKey !== getSessionKey()) return;
      if (!this.isAuthenticatedRoute) return;
      if (status.guideCompleted === false) {
        if (!this.guideMode) {
          this.guideMode = 'required';
          this.guideError = '';
          this.savingGuide = false;
        }
      } else if (this.guideMode === 'required') {
        this.guideMode = null;
      }
    },
    // After a failed save: let the player in for this session only. The
    // server still says the guide is incomplete, so the next session asks
    // again.
    skipGuide() {
      if (this.guideMode !== 'required' || this.savingGuide) return;
      rememberGuideCompletion();
      this.guideMode = null;
      this.guideError = '';
    },
    openGuide() {
      if (!this.guideMode) this.guideMode = 'replay';
    },
    closeGuide() {
      if (this.guideMode === 'replay') this.guideMode = null;
    },
    async finishGuide() {
      if (this.guideMode === 'replay') {
        this.closeGuide();
        return;
      }
      if (this.guideMode !== 'required' || this.savingGuide) return;
      const sessionKey = getSessionKey();
      this.savingGuide = true;
      this.guideError = '';
      try {
        await completeGuide();
        if (sessionKey === getSessionKey()) {
          rememberGuideCompletion();
          this.guideMode = null;
        }
      } catch {
        if (sessionKey === getSessionKey() && this.guideMode === 'required') {
          this.guideError = this.$t('guide.saveError');
        }
      } finally {
        if (sessionKey === getSessionKey()) this.savingGuide = false;
      }
    },
    // The player's saved language on a signed-in page (a no-op after the
    // first read of the session, or when main.js already did it); the
    // browser's language on a public one, whatever was chosen before.
    syncLocale() {
      if (this.isAuthenticatedRoute) syncSavedLocale();
      else resetSavedLocale();
    },
    async logout() {
      // Invalidate an in-flight features response before anything awaits, so
      // it can't open this session's guide after logout.
      this.guideRequest += 1;
      this.guideMode = null;
      resetFeatureStatus();
      // Before the token goes: the next account on this device must not get
      // this one's notifications, and the backend needs the token to forget
      // the device.
      await disablePushOnLogout();
      try {
        await logoutSession();
      } catch {
        console.warn('Could not revoke the server session.');
      }
      clearToken();
      resetSavedLocale();
      // The next account to log in on this browser must not inherit this
      // one's group — the stale-id fallback would catch it, but only after a
      // request scoped to a group they may not belong to.
      clearActiveGroupId();
      // Clear the cached groups promise to prevent the next logged-in user from
      // seeing the previous user's group memberships/roles. Without this, the
      // in-memory promise cache would survive the logout and return stale data.
      clearMyGroupsCache();
      this.$router.push('/login');
    }
  }
}
</script>

<style>
/* Import global styles - Add this import in your main.js or main.ts instead */
/* @import './assets/global-styles.css'; */

/* App-specific styles that can't be moved to global */
#app {
  position: relative;
  /* `.dark-mode` toggles here, not on <body> — global-styles.css's `body {
     color: var(--text-primary) }` resolves against :root's (light) value
     since body is an ANCESTOR of #app and never sees the .dark-mode class,
     then that already-resolved light colour is what plain CSS inheritance
     hands down to every descendant. Most components never notice, because
     they redeclare `color: var(--text-primary)` themselves — which
     re-resolves the variable against the nearer #app.dark-mode scope and
     picks up the dark value correctly. But anything that relies on plain
     inheritance instead of setting its own `color` (e.g. MatchDetails.vue's
     .signup-list-title) stays stuck showing light-mode text on a dark
     background. Restating the same declaration here breaks the inherited
     chain at the one point that actually sees the theme class, so every
     un-styled descendant inherits the *correct* value from here instead. */
  color: var(--text-primary);
}

/* Main Content */
.app-container {
  background-color: var(--bg-secondary);
}

.main-content {
  margin-top: var(--navbar-height);
  min-height: calc(100vh - var(--navbar-height));
}

/* Transitions */
.app-container-enter-active,
.app-container-leave-active {
  transition: all var(--transition-smooth);
}

.app-container-enter-from {
  opacity: 0;
  transform: translateY(20px);
}

.app-container-leave-to {
  opacity: 0;
  transform: translateY(-20px);
}

@media (max-width: 768px) {
  .main-content.no-top-bar-on-mobile {
    margin-top: 0;
    min-height: 100vh;
    padding-top: env(safe-area-inset-top, 0px);
  }

  /* Leaves room for the fixed bottom tab bar so the last bit of page
     content (e.g. a "Danger zone" card) isn't hidden underneath it. */
  .main-content {
    padding-bottom: calc(4.5rem + env(safe-area-inset-bottom, 0px));
  }
}
</style>
