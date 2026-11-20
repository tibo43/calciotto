<template>
  <div id="app" :class="{ 'dark-mode': isDarkMode, 'design-v2': designV2 }">
    <PlayerGuide v-if="guideMode" :required="guideMode === 'required'" :saving="savingGuide"
      :dark="isDarkMode" :error="guideError" @finish="finishGuide" @dismiss="closeGuide" @skip="skipGuide" />
    <div v-else class="app-container" :class="{ wrap: designV2, 'is-confirming': designV2 && v2Context.confirming, 'is-picking-csc': designV2 && v2Context.pickingCsc }">
        <header v-if="designV2" class="topbar">
          <div class="topbar-left">
            <template v-if="v2Context.pageTitle">
              <router-link class="back" :to="v2Context.backTo || '/profile'" aria-label="Back">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M15 5l-7 7 7 7"/></svg>
              </router-link>
              <h1 class="page-title">{{ v2Context.pageTitle }}</h1>
            </template>
            <template v-else>
            <router-link class="brand" to="/" aria-label="Home">
              <img class="brand-mark brand-mark-dark" src="/icons/v2/header-symbole-sombre.png" width="40" height="40" alt="">
              <img class="brand-mark brand-mark-light" src="/icons/v2/header-symbole-clair.png" width="40" height="40" alt="">
            </router-link>
            <div v-if="isAuthenticatedRoute && !v2Context.hide" class="ctx-pair" aria-label="Choose the group and the season">
              <label class="ctx">
                <span class="sr-only">Group</span>
                <span class="ctx-name" aria-hidden="true">{{ v2GroupLabel }}</span>
                <select :value="v2Context.groupId" @change="onV2Group">
                  <option v-for="group in v2Context.groups" :key="group.id" :value="group.id">{{ group.name }}</option>
                </select>
                <svg class="chev" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M2 4l4 4 4-4"/></svg>
              </label>
              <label class="ctx" :class="{ 'is-disabled': !v2HasSeasons }">
                <span class="sr-only">Season</span>
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
          <nav v-if="isAuthenticatedRoute && !v2Context.pageTitle" class="dnav" aria-label="Main">
            <router-link v-for="item in v2NavItems" :key="item.to" :to="item.to" class="nav-text" :class="{ 'is-on': item.on }"
              :aria-current="item.on ? 'page' : null">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="item.icon" /></svg>
              {{ item.label }}
            </router-link>
          </nav>
          <!-- No theme toggle here: in v2 the theme is a setting, on the
               Account page (ProfileV2), not a control on every page. -->
        </header>
        <!-- Signup, forgot password and reset password have no v2 version
             yet: they keep the original top bar (brand and theme toggle). -->
        <nav v-if="!designV2" class="top-navbar" :class="{ 'scrolled': isScrolled }">
          <div class="nav-container">
            <div class="nav-brand" @click="goHome">
              <img class="brand-logo" :src="isDarkMode ? '/icons/icon-192.png' : '/icons/logo-light.png'"
                alt="Calciotto">
            </div>
            <div class="nav-actions">
              <button
                class="theme-toggle"
                @click="toggleTheme"
                :aria-label="isDarkMode ? 'Switch to light mode' : 'Switch to dark mode'"
              >
                <transition name="theme-icon" mode="out-in">
                  <svg v-if="isDarkMode" key="sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="5"/>
                    <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
                  </svg>
                  <svg v-else key="moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
                  </svg>
                </transition>
              </button>
            </div>
          </div>
        </nav>

        <!-- Main Content -->
        <main class="main-content" :class="{ 'no-top-bar-on-mobile': isAuthenticatedRoute }">
          <router-view />
        </main>

        <nav v-if="designV2 && isAuthenticatedRoute" class="tabbar" aria-label="Main">
          <router-link v-for="item in v2NavItems" :key="item.to" :to="item.to" class="nav-ico" :class="{ 'is-on': item.on }"
            :aria-current="item.on ? 'page' : null">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="item.icon" /></svg>
            <span class="nav-ico-label">{{ item.label }}</span>
          </router-link>
        </nav>
    </div>
  </div>
</template>

<script>
import PlayerGuide from '@/components/PlayerGuide.vue';
import GearLink from '@/components/v2/GearLink.vue';
import { clearToken, completeGuide, getSessionKey, logoutSession } from '@/services/api';
import { clearActiveGroupId, clearMyGroupsCache } from '@/services/activeGroup';
import { syncThemeColorMeta, prefersDarkTheme } from '@/services/theme';
import { disablePushOnLogout, stopPushIfAppUninstalled } from '@/services/pushNotifications';
import {
  knownFeatureStatus, loadFeatureStatus, rememberGuideCompletion, resetFeatureStatus
} from '@/services/featureStatus';

// Routes reachable without a token — nothing group-related is shown there.
const PUBLIC_ROUTE_NAMES = ['Login', 'Signup', 'ForgotPassword', 'ResetPassword'];

// Public pages that have no v2 version yet: they keep the original styling
// (no design-v2 class) and the original top bar.
const V1_ROUTE_NAMES = ['Signup', 'ForgotPassword', 'ResetPassword'];

export default {
  name: 'App',
  components: { GearLink, PlayerGuide },
  provide() {
    return {
      v2Context: this.v2Context,
      openGuide: this.openGuide,
    };
  },
  data() {
    return {
      isScrolled: false,
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
    designV2() {
      return !V1_ROUTE_NAMES.includes(this.$route.name);
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
      return this.v2HasSeasons ? (this.v2Context.season || '') : 'No season yet';
    },
    isV2Profile() {
      return String(this.$route.path || '').startsWith('/profile');
    },
    // The v2 menu: icon and name in the desktop top bar, icon and name in
    // the phone tab bar. The visible label is the accessible name.
    v2NavItems() {
      return [
        { to: '/', label: 'Matches', on: this.isV2Home, icon: 'M3 9a9 3 0 1 0 18 0 9 3 0 1 0-18 0zM3 9v6a9 3 0 0 0 18 0V9M7 9a5 1.4 0 1 0 10 0 5 1.4 0 1 0-10 0zM7.5 17.6v-3M12 18v-3M16.5 17.6v-3' },
        { to: '/rankings', label: 'Rankings', on: this.$route.name === 'Rankings', icon: 'M8 21h8M12 17v4M7 4h10v5a5 5 0 0 1-10 0V4zM7 6H4v2a3 3 0 0 0 3 3M17 6h3v2a3 3 0 0 1-3 3' },
        { to: '/predictions', label: 'Predictions', on: this.$route.name === 'Predictions', icon: 'M11 12a1 1 0 1 0 2 0 1 1 0 1 0-2 0M12 7a5 5 0 1 0 5 5M13 3.055a9 9 0 1 0 7.941 7.945M15 6v3h3l3-3h-3V3zM15 9l-3 3' },
        { to: '/profile', label: 'Profile', on: this.isV2Profile, icon: 'M12 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM4 21a8 8 0 0 1 16 0' },
      ];
    }
  },
  watch: {
    isDarkMode(isDark) {
      syncThemeColorMeta(isDark);
    },
    '$route.name'() {
      this.syncGuide();
    }
  },
  created() {
    this.syncGuide();
  },
  mounted() {
    // Android keeps an uninstalled app's notifications going through Chrome;
    // this turns them off when the site is opened again in a tab (see
    // stopPushIfAppUninstalled). Fire and forget: it never blocks the page.
    stopPushIfAppUninstalled();

    syncThemeColorMeta(this.isDarkMode);

    // Add scroll listener
    window.addEventListener('scroll', this.handleScroll);
  },
  beforeUnmount() {
    window.removeEventListener('scroll', this.handleScroll);
  },
  methods: {
    goHome() {
      this.$router.push('/');
    },
    handleScroll() {
      this.isScrolled = window.scrollY > 20;
    },
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
          this.guideError = 'Could not save your progress. Try again.';
        }
      } finally {
        if (sessionKey === getSessionKey()) this.savingGuide = false;
      }
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

/* Navigation Styles - Component-specific */
.top-navbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: var(--navbar-height);
  background-color: rgba(248, 250, 252, 0.95);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--border-color);
  transition: all var(--transition-smooth);
  z-index: 1000;
}

.dark-mode .top-navbar {
  background-color: rgba(15, 23, 42, 0.95);
}

.top-navbar.scrolled {
  box-shadow: var(--shadow-md);
  background-color: rgba(248, 250, 252, 0.98);
}

.dark-mode .top-navbar.scrolled {
  background-color: rgba(15, 23, 42, 0.98);
}

.nav-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 1.5rem;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.nav-brand {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  cursor: pointer;
  transition: transform var(--transition-fast);
}

.nav-brand:hover {
  transform: scale(1.05);
}

.brand-logo {
  display: block;
  width: 44px;
  height: 44px;
  border-radius: 10px;
}

.nav-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.theme-toggle {
  background: none;
  border: none;
  padding: 0.5rem;
  border-radius: var(--border-radius);
  cursor: pointer;
  color: var(--text-secondary);
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
}

.theme-toggle:hover {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
}

.theme-toggle svg {
  width: 20px;
  height: 20px;
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

.theme-icon-enter-active,
.theme-icon-leave-active {
  transition: all var(--transition-fast);
}

.theme-icon-enter-from {
  opacity: 0;
  transform: rotate(-90deg) scale(0.8);
}

.theme-icon-leave-to {
  opacity: 0;
  transform: rotate(90deg) scale(0.8);
}

/* Mobile Responsive: everything (brand, Matches/Profile, logout, theme
   toggle) stays on one row instead of opening a separate dropdown menu —
   the nav links just drop their text label and become icon-only, matching
   the existing icon-only actions, rather than being hidden off-screen
   behind a hamburger toggle. */
@media (max-width: 768px) {
  .nav-container {
    padding: 0 1rem;
  }

  /* No logo on a phone: the installed app already has it on the home
     screen, and the bar there only exists on the public pages. */
  .nav-brand {
    display: none;
  }

  /* With the logo gone, the actions are the bar's only item on a public
     page; space-between would otherwise put them on the left. */
  .nav-actions {
    margin-left: auto;
  }

  .nav-actions {
    gap: 0.25rem;
  }

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
