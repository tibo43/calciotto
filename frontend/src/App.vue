<template>
  <div id="app" :class="{ 'dark-mode': isDarkMode, 'design-v2': designV2 }">
    <!-- Main App Container -->
    <div class="app-container" :class="{ wrap: designV2, 'is-confirming': designV2 && v2Context.confirming, 'is-picking-csc': designV2 && v2Context.pickingCsc }">
        <header v-if="designV2" class="topbar">
          <div class="topbar-left">
            <router-link class="brand" to="/" aria-label="Home">
              <img class="brand-mark brand-mark-dark" src="/icons/v2/header-symbole-sombre.png" width="40" height="40" alt="">
              <img class="brand-mark brand-mark-light" src="/icons/v2/header-symbole-clair.png" width="40" height="40" alt="">
            </router-link>
            <div v-if="isAuthenticatedRoute && !v2Context.hide" class="ctx-pair" aria-label="Choose the group and the season">
              <label class="ctx">
                <span class="sr-only">Group</span>
                <select :value="v2Context.groupId" @change="onV2Group">
                  <option v-for="group in v2Context.groups" :key="group.id" :value="group.id">{{ group.name }}</option>
                </select>
                <svg class="chev" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M2 4l4 4 4-4"/></svg>
              </label>
              <label class="ctx">
                <span class="sr-only">Season</span>
                <select :value="v2Context.season" @change="onV2Season">
                  <option v-for="season in v2Context.seasons" :key="season" :value="season">{{ season }}</option>
                </select>
                <svg class="chev" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M2 4l4 4 4-4"/></svg>
              </label>
            </div>
          </div>
          <nav v-if="isAuthenticatedRoute" class="dnav" aria-label="Main">
            <router-link to="/" :class="{ 'is-on': isV2Home }"><span class="outline" :class="{ 'is-on': isV2Home }">Home</span></router-link>
            <router-link to="/classements" :class="{ 'is-on': $route.name === 'Classements' }"><span class="outline" :class="{ 'is-on': $route.name === 'Classements' }">Rankings</span></router-link>
            <router-link to="/predictions" :class="{ 'is-on': $route.name === 'Predictions' }"><span class="outline" :class="{ 'is-on': $route.name === 'Predictions' }">Predictions</span></router-link>
            <router-link to="/profile" :class="{ 'is-on': $route.name === 'Profile' }"><span class="outline" :class="{ 'is-on': $route.name === 'Profile' }">Profile</span></router-link>
          </nav>
          <div class="topbar-right">
            <button class="icon-btn" type="button" @click="toggleTheme" :aria-label="isDarkMode ? 'Switch to light mode' : 'Switch to dark mode'">
              <svg class="ico-sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="12" cy="12" r="4"/><path d="M12 3v2M12 19v2M5 12H3M21 12h-2M6.2 6.2l1.4 1.4M16.4 16.4l1.4 1.4M6.2 17.8l1.4-1.4M16.4 7.6l1.4-1.4"/></svg>
              <svg class="ico-moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M16 3a8 8 0 1 0 5 13 7 7 0 0 1-5-13z"/></svg>
            </button>
          </div>
        </header>
        <!-- Top Navigation -->
        <!-- On a phone, an authenticated page has no top bar at all: the
             bottom tab bar navigates, and logout/theme live at the top of
             the bottom bar, next to the Profile tab. Public pages keep it,
             since they have no bottom bar and no profile to reach. -->
        <nav v-if="!designV2" class="top-navbar" :class="{ 'scrolled': isScrolled, 'hidden-on-mobile': isAuthenticatedRoute }">
          <div class="nav-container">
            <!-- Logo/Brand -->
            <div class="nav-brand" @click="goHome">
              <!-- The app icon (public/icons/, generated from
                   branding/icon-source.png), so the web page and the
                   installed app share one mark. Light mode uses the light
                   source (branding/icon-source-light.png, written to
                   logo-light.png). Desktop only (see the mobile rule below). -->
              <img class="brand-logo" :src="isDarkMode ? '/icons/icon-192.png' : '/icons/logo-light.png'"
                alt="Calciotto">
            </div>

            <!-- Desktop Menu — hidden on public routes (Login/Signup/...):
                 these links require a token, so showing them on an
                 unauthenticated page misrepresents it as the logged-in app
                 shell rather than a standalone login view. -->
            <div v-if="isAuthenticatedRoute" class="nav-menu">
              <router-link
                to="/"
                :class="{ 'active': $route.name === 'MatchesAndStandings' }"
                class="nav-button"
                aria-label="Matches"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
                  <line x1="16" y1="2" x2="16" y2="6"/>
                  <line x1="8" y1="2" x2="8" y2="6"/>
                  <line x1="3" y1="10" x2="21" y2="10"/>
                </svg>
                <span class="nav-button-label">Matches</span>
              </router-link>
              <router-link
                to="/predictions"
                :class="{ 'active': $route.name === 'Predictions' }"
                class="nav-button"
                aria-label="Pronostic"
              >
                <!-- Target/bullseye — "getting the call right", not a dice or
                     chip: this is a free prediction game with no stake or
                     balance (see matchPredictions.js), so an icon evoking
                     real-money betting would misrepresent it. -->
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="9"/>
                  <circle cx="12" cy="12" r="5"/>
                  <circle cx="12" cy="12" r="1" fill="currentColor" stroke="none"/>
                </svg>
                <span class="nav-button-label">Pronostic</span>
              </router-link>
              <router-link
                to="/profile"
                :class="{ 'active': $route.name === 'Profile' }"
                class="nav-button"
                aria-label="Profile"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                  <circle cx="12" cy="7" r="4"/>
                </svg>
                <span class="nav-button-label">Profile</span>
              </router-link>
            </div>

            <!-- Actions -->
            <div class="nav-actions">
              <button
                v-if="isAuthenticatedRoute"
                class="icon-nav-button logout-btn"
                @click="logout"
                aria-label="Log out"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
                  <polyline points="16 17 21 12 16 7"/>
                  <line x1="21" y1="12" x2="9" y2="12"/>
                </svg>
              </button>

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

        <!-- Bottom tab bar — mobile only (see the @media block below;
             desktop keeps the top nav-menu as its only navigation). Real
             feedback: a top nav with icon-only, label-less buttons at phone
             width (see .nav-button-label's own mobile rule above) read as
             cramped and unfamiliar next to how every native-feeling mobile
             app places its primary navigation at the thumb, along the
             bottom edge — so this isn't a duplicate of the top nav, it
             replaces it as the actual navigation surface on a phone, and
             the top bar is hidden altogether. Its two actions (theme,
             logout) appear at the end of this bar, next to the Profile tab,
             while the Profile page is open: account-level actions where the
             account lives, out of the way everywhere else. Same three destinations, labels
             and icons as the top nav-menu, so both bars read as one app. -->
        <nav v-if="designV2 && isAuthenticatedRoute" class="tabbar" aria-label="Main">
          <router-link to="/" :class="{ 'is-on': isV2Home }"><span class="outline" :class="{ 'is-on': isV2Home }">Home</span></router-link>
          <router-link to="/classements" :class="{ 'is-on': $route.name === 'Classements' }"><span class="outline" :class="{ 'is-on': $route.name === 'Classements' }">Rankings</span></router-link>
          <router-link to="/predictions" :class="{ 'is-on': $route.name === 'Predictions' }"><span class="outline" :class="{ 'is-on': $route.name === 'Predictions' }">Predictions</span></router-link>
          <router-link to="/profile" :class="{ 'is-on': $route.name === 'Profile' }"><span class="outline" :class="{ 'is-on': $route.name === 'Profile' }">Profile</span></router-link>
        </nav>
        <nav v-if="isAuthenticatedRoute && !designV2" class="bottom-tab-bar">
          <router-link to="/" :class="['bottom-tab-button', { active: $route.name === 'MatchesAndStandings' }]"
            aria-label="Matches">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
              <line x1="16" y1="2" x2="16" y2="6" />
              <line x1="8" y1="2" x2="8" y2="6" />
              <line x1="3" y1="10" x2="21" y2="10" />
            </svg>
            <span class="bottom-tab-label">Matches</span>
          </router-link>
          <router-link to="/predictions" :class="['bottom-tab-button', { active: $route.name === 'Predictions' }]"
            aria-label="Pronostic">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="9" />
              <circle cx="12" cy="12" r="5" />
              <circle cx="12" cy="12" r="1" fill="currentColor" stroke="none" />
            </svg>
            <span class="bottom-tab-label">Pronostic</span>
          </router-link>
          <router-link to="/profile" :class="['bottom-tab-button', { active: $route.name === 'Profile' }]"
            aria-label="Profile">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
              <circle cx="12" cy="7" r="4" />
            </svg>
            <span class="bottom-tab-label">Profile</span>
          </router-link>
          <div v-if="$route.name === 'Profile'" class="bottom-tab-actions">
            <button class="bottom-tab-action" @click="toggleTheme"
              :aria-label="isDarkMode ? 'Switch to light mode' : 'Switch to dark mode'">
              <svg v-if="isDarkMode" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="5" />
                <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
              </svg>
              <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
              </svg>
            </button>
            <button class="bottom-tab-action" @click="logout" aria-label="Log out">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
                <polyline points="16 17 21 12 16 7" />
                <line x1="21" y1="12" x2="9" y2="12" />
              </svg>
            </button>
          </div>
        </nav>
    </div>
  </div>
</template>

<script>
import { computed } from 'vue';
import { clearToken, getMyFeatures } from '@/services/api';
import { clearActiveGroupId, clearMyGroupsCache } from '@/services/activeGroup';
import { syncThemeColorMeta } from '@/services/theme';
import { disablePushOnLogout } from '@/services/pushNotifications';
import {
  readDesignV2Cached,
  writeDesignV2Cached,
  clearDesignV2Cached,
  hasDesignV2,
  ensureDesignV2Fonts,
  removeDesignV2Fonts
} from '@/services/designV2';

// Routes reachable without a token — nothing group-related is shown there.
const PUBLIC_ROUTE_NAMES = ['Login', 'Signup', 'ForgotPassword', 'ResetPassword'];

export default {
  name: 'App',
  provide() {
    return {
      designV2On: computed(() => this.designV2),
      v2Context: this.v2Context,
    };
  },
  data() {
    return {
      isScrolled: false,
      isDarkMode: false,
      // Set in created(), which runs before the first render — data() itself
      // is too early for $route (it's a computed) and for methods.
      designV2: false,
      // Set once a features response has landed this session. Later route
      // changes reuse it; a failure leaves it false so the next navigation
      // retries. Logout bumps the request id and resets this.
      v2Context: { label: '', confirming: false, pickingCsc: false, hide: false, groups: [], seasons: [], groupId: '', season: '', onGroup: null, onSeason: null },
      designV2Loaded: false,
      designV2InFlight: false,
      designV2Request: 0
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
      return this.$route.name === 'MatchesAndStandings' || this.$route.name === 'MatchView';
    }
  },
  watch: {
    isDarkMode(isDark) {
      syncThemeColorMeta(isDark);
    },
    '$route.name'() {
      this.syncDesignV2();
    }
  },
  created() {
    // Before the first render, so a page that already knows it is on the
    // redesign doesn't paint the current design for a frame. Auth pages
    // take the class too when the cache is set: they have no session to
    // ask, and the mockups style them in v2 for someone who already has it.
    this.applyCachedDesignV2();
  },
  mounted() {
    // Check for saved theme preference
    const savedTheme = localStorage.getItem('calciotto-theme');
    if (savedTheme) {
      this.isDarkMode = savedTheme === 'dark';
    } else {
      this.isDarkMode = window.matchMedia('(prefers-color-scheme: dark)').matches;
    }

    // Add scroll listener
    window.addEventListener('scroll', this.handleScroll);
    this.syncDesignV2();
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
    // The cache is enough on its own: an authenticated page confirms it
    // with the API, and an auth page (no token to ask with) keeps whatever
    // this browser last learned. No cache means the current design.
    applyCachedDesignV2() {
      if (readDesignV2Cached()) {
        this.designV2 = true;
        ensureDesignV2Fonts();
      }
    },
    // Cached class first (synchronous), then one features fetch per session
    // to confirm it. Auth pages never fetch — there is no session.
    syncDesignV2() {
      if (!this.isAuthenticatedRoute) {
        this.designV2 = readDesignV2Cached();
        if (this.designV2) {
          ensureDesignV2Fonts();
        }
        return;
      }
      this.applyCachedDesignV2();
      if (this.designV2Loaded || this.designV2InFlight) {
        return;
      }
      this.fetchDesignV2();
    },
    async fetchDesignV2() {
      const requestId = ++this.designV2Request;
      this.designV2InFlight = true;
      try {
        const features = await getMyFeatures();
        // Logout (or a newer fetch) happened while this was in flight.
        // Drop the response: it must not repaint the class or rewrite the
        // cache that logout just cleared.
        if (requestId !== this.designV2Request) {
          return;
        }
        const enabled = hasDesignV2(features);
        writeDesignV2Cached(enabled);
        this.designV2Loaded = true;
        if (!this.isAuthenticatedRoute) {
          return;
        }
        this.designV2 = enabled;
        if (enabled) {
          ensureDesignV2Fonts();
        } else {
          removeDesignV2Fonts();
        }
      } catch (error) {
        if (requestId !== this.designV2Request) {
          return;
        }
        // Keep whatever the cache already painted. A blip must not turn the
        // redesign off for someone who has it, and must not turn it on for
        // someone who doesn't. The next authenticated navigation retries.
        console.error('Error fetching features:', error);
      } finally {
        if (requestId === this.designV2Request) {
          this.designV2InFlight = false;
        }
      }
    },
    async logout() {
      // Invalidate an in-flight features response before anything awaits, so
      // it can't write the cache back after this clears it.
      this.designV2Request += 1;
      this.designV2 = false;
      this.designV2Loaded = false;
      this.designV2InFlight = false;
      // Before the token goes: the next account on this device must not get
      // this one's notifications, and the backend needs the token to forget
      // the device.
      await disablePushOnLogout();
      clearToken();
      // The next account to log in on this browser must not inherit this
      // one's group — the stale-id fallback would catch it, but only after a
      // request scoped to a group they may not belong to.
      clearActiveGroupId();
      // Clear the cached groups promise to prevent the next logged-in user from
      // seeing the previous user's group memberships/roles. Without this, the
      // in-memory promise cache would survive the logout and return stale data.
      clearMyGroupsCache();
      clearDesignV2Cached();
      removeDesignV2Fonts();
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

.nav-menu {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.nav-button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.25rem;
  background: none;
  border: none;
  border-radius: var(--border-radius);
  color: var(--text-secondary);
  font-weight: 500;
  font-size: 0.95rem;
  cursor: pointer;
  transition: all var(--transition-fast);
  position: relative;
  text-decoration: none;
}

.nav-button svg {
  width: 18px;
  height: 18px;
}

.nav-button:hover {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
  transform: translateY(-1px);
}

.nav-button.active {
  background-color: var(--primary-color);
  color: white;
  box-shadow: var(--shadow-md);
}

.nav-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.icon-nav-button {
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
  text-decoration: none;
}

.icon-nav-button:hover {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
}

.icon-nav-button.active {
  background-color: var(--primary-color);
  color: white;
  box-shadow: var(--shadow-md);
}

.icon-nav-button svg {
  width: 20px;
  height: 20px;
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

/* Bottom tab bar — hidden by default (desktop keeps the top nav-menu as its
   only navigation); shown at the same mobile breakpoint the rest of this
   file already uses, in the @media block below. */
.bottom-tab-bar {
  display: none;
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
  background-color: rgba(248, 250, 252, 0.98);
  backdrop-filter: blur(12px);
  border-top: 1px solid var(--border-color);
  /* The home indicator on a notched phone sits in this inset — padding
     rather than margin so the bar's own background still fills behind it. */
  padding-bottom: env(safe-area-inset-bottom, 0px);
}

.dark-mode .bottom-tab-bar {
  background-color: rgba(15, 23, 42, 0.98);
}

.bottom-tab-button {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.2rem;
  padding: 0.5rem 0.25rem;
  color: var(--text-secondary);
  text-decoration: none;
  transition: color var(--transition-fast);
}

.bottom-tab-button svg {
  width: 22px;
  height: 22px;
}

.bottom-tab-label {
  font-size: 0.7rem;
  font-weight: 500;
}

.bottom-tab-button.active {
  color: var(--primary-color);
}

/* Theme and logout, shown only on the Profile page: a compact pair set off
   from the tabs by a divider, so they don't read as two more destinations. */
.bottom-tab-actions {
  display: flex;
  align-items: center;
  gap: 0.15rem;
  padding: 0 0.5rem;
  border-left: 1px solid var(--border-color);
  margin: 0.6rem 0;
}

.bottom-tab-action {
  background: none;
  border: none;
  padding: 0.5rem;
  border-radius: var(--border-radius);
  cursor: pointer;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.bottom-tab-action:hover {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
}

.bottom-tab-action svg {
  width: 20px;
  height: 20px;
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

  .nav-button-label {
    display: none;
  }

  .nav-button {
    padding: 0.5rem;
  }

  /* The bottom tab bar (below) replaces the top nav-menu as the actual
     navigation surface on a phone — see its own comment in the template
     above for why. On an authenticated page the whole top bar goes (see
     .hidden-on-mobile); on a public one only its links do. */
  .nav-menu {
    display: none;
  }

  .nav-actions {
    gap: 0.25rem;
  }

  .top-navbar.hidden-on-mobile {
    display: none;
  }

  .main-content.no-top-bar-on-mobile {
    margin-top: 0;
    min-height: 100vh;
    padding-top: env(safe-area-inset-top, 0px);
  }

  .bottom-tab-bar {
    display: flex;
  }

  /* Leaves room for the fixed bottom tab bar so the last bit of page
     content (e.g. a "Danger zone" card) isn't hidden underneath it. */
  .main-content {
    padding-bottom: calc(4.5rem + env(safe-area-inset-bottom, 0px));
  }
}
</style>