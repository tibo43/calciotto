<template>
  <div id="app" :class="{ 'dark-mode': isDarkMode }">
    <!-- Main App Container -->
    <div class="app-container">
        <!-- Top Navigation -->
        <nav class="top-navbar" :class="{ 'scrolled': isScrolled }">
          <div class="nav-container">
            <!-- Logo/Brand -->
            <div class="nav-brand" @click="goHome">
              <span class="brand-text">Calciotto</span>
            </div>

            <!-- Desktop Menu — hidden on public routes (Login/Signup/...):
                 these links require a token, so showing them on an
                 unauthenticated page misrepresents it as the logged-in app
                 shell rather than a standalone login view. -->
            <div v-if="isAuthenticatedRoute" class="nav-menu" :class="{ 'active': isMenuOpen }">
              <router-link 
                to="/" 
                @click="closeMenu"
                :class="{ 'active': $route.name === 'MatchesAndStandings' }"
                class="nav-button"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
                  <line x1="16" y1="2" x2="16" y2="6"/>
                  <line x1="8" y1="2" x2="8" y2="6"/>
                  <line x1="3" y1="10" x2="21" y2="10"/>
                </svg>
                Matches
              </router-link>
              <router-link
                to="/groups"
                @click="closeMenu"
                :class="{ 'active': $route.name === 'Groups' }"
                class="nav-button"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M3 21v-2a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v2"/>
                  <circle cx="9" cy="7" r="4"/>
                  <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
                  <path d="M21 21v-2a4 4 0 0 0-3-3.87"/>
                </svg>
                Groups
              </router-link>
            </div>

            <!-- Actions -->
            <div class="nav-actions">
              <!-- The group Matches/Standings are scoped to, plus the
                   onboarding actions (join/create) that used to live on the
                   Groups page. Rendered for any authenticated, non-public
                   route so a player with zero groups still has a way to get
                   into one. -->
              <GroupSwitcher
                v-if="showGroupSwitcher"
                :groups="groups"
                :active-group-id="activeGroupId"
                :is-dark-mode="isDarkMode"
              />

              <router-link
                v-if="isAuthenticatedRoute"
                to="/profile"
                class="icon-nav-button"
                :class="{ 'active': $route.name === 'Profile' }"
                :aria-label="$route.name === 'Profile' ? 'Profile (current page)' : 'Profile'"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                  <circle cx="12" cy="7" r="4"/>
                </svg>
              </router-link>

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

              <button
                v-if="isAuthenticatedRoute"
                class="mobile-menu-toggle"
                :class="{ 'active': isMenuOpen }"
                @click="toggleMenu"
                aria-label="Toggle navigation menu"
              >
                <span></span>
                <span></span>
                <span></span>
              </button>
            </div>
          </div>
        </nav>

        <!-- Main Content -->
        <main class="main-content">
          <!-- Router View for Match Routes -->
          <router-view v-if="isRouterRoute" />
          
          <!-- Tab Content for non-router routes -->
          <transition v-else name="tab-content" mode="out-in">
            <div v-if="activeTab === 'teams'" key="teams" class="tab-content">
              <div class="empty-state">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="empty-icon">
                  <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
                  <circle cx="9" cy="7" r="4"/>
                  <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
                  <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
                </svg>
                <h2>Teams</h2>
                <p>Coming soon...</p>
              </div>
            </div>
            <div v-else-if="activeTab === 'players'" key="players" class="tab-content">
              <div class="empty-state">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="empty-icon">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                  <circle cx="12" cy="7" r="4"/>
                </svg>
                <h2>Players</h2>
                <p>Coming soon...</p>
              </div>
            </div>
          </transition>
        </main>

        <!-- Mobile Menu Overlay -->
        <transition name="overlay">
          <div v-if="isMenuOpen" class="mobile-overlay" @click="closeMenu"></div>
        </transition>
    </div>
  </div>
</template>

<script>
import { clearToken, getToken } from '@/services/api';
import { clearActiveGroupId, clearMyGroupsCache, resolveActiveGroup } from '@/services/activeGroup';
import GroupSwitcher from '@/components/GroupSwitcher.vue';

// Routes reachable without a token — the group selector has nothing to show
// there (no /groups/me to call), same list refreshGroups() early-returns on.
const PUBLIC_ROUTE_NAMES = ['Login', 'Signup', 'ForgotPassword', 'ResetPassword'];

export default {
  name: 'App',
  components: { GroupSwitcher },
  data() {
    return {
      activeTab: 'matches',
      isMenuOpen: false,
      isScrolled: false,
      isDarkMode: false,
      groups: [],
      activeGroupId: '',
      // Plain reactive data rather than a computed: a computed reading the
      // non-reactive getToken() alongside the reactive $route.name has shown
      // stale/stuck values in practice (the cached computed not
      // re-evaluating on a route change even though a fresh call of the same
      // expression returns the right answer). Setting this explicitly,
      // synchronously, at the top of refreshGroups() sidesteps that
      // entirely.
      showGroupSwitcher: false
    };
  },
  computed: {
    isRouterRoute() {
      // Check if current route should be handled by router
      return ['MatchesAndStandings', 'MatchDetails', 'Groups', 'Profile', 'Login', 'Signup', 'ForgotPassword', 'ResetPassword'].includes(this.$route.name);
    },
    // Gates the nav-menu links (Matches/Groups) and the Profile/Logout icons:
    // all three require a token, so showing them on a public route (where
    // there's no session to act on) makes a standalone login/signup page
    // read as the logged-in app shell instead of its own view.
    isAuthenticatedRoute() {
      return !PUBLIC_ROUTE_NAMES.includes(this.$route.name);
    }
  },
  watch: {
    '$route'(to) {
      // Update activeTab when route changes
      if (to.name === 'MatchesAndStandings') {
        this.activeTab = 'matches';
      }
      this.closeMenu();
      this.refreshGroups();
    }
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

    this.refreshGroups();
  },
  beforeUnmount() {
    window.removeEventListener('scroll', this.handleScroll);
  },
  methods: {
    goHome() {
      this.$router.push('/');
      this.activeTab = 'matches';
    },
    toggleMenu() {
      this.isMenuOpen = !this.isMenuOpen;
      document.body.style.overflow = this.isMenuOpen ? 'hidden' : '';
    },
    closeMenu() {
      this.isMenuOpen = false;
      document.body.style.overflow = '';
    },
    handleScroll() {
      this.isScrolled = window.scrollY > 20;
    },
    toggleTheme() {
      this.isDarkMode = !this.isDarkMode;
      localStorage.setItem('calciotto-theme', this.isDarkMode ? 'dark' : 'light');
    },
    // App.vue outlives every route, so the selector has to be (re)filled on
    // navigation and not only on mount: logging in doesn't remount it, it just
    // pushes a route, and at mount time there was no token to call
    // /groups/me with.
    async refreshGroups() {
      this.showGroupSwitcher = Boolean(getToken()) && !PUBLIC_ROUTE_NAMES.includes(this.$route.name);
      if (!this.showGroupSwitcher) {
        this.groups = [];
        this.activeGroupId = '';
        return;
      }
      try {
        // Switching, joining, and creating a group all go through a full
        // page reload (see GroupSwitcher/CreateGroupModal), so the cached
        // list is never stale on a plain in-app navigation — no force-refresh
        // heuristic needed here.
        const { groups, activeGroupId } = await resolveActiveGroup();
        this.groups = groups;
        this.activeGroupId = activeGroupId;
      } catch (error) {
        // The selector is a convenience; every view still falls back to the
        // backend's own group resolution without it.
        console.error('Error loading the group selector:', error);
      }
    },
    logout() {
      clearToken();
      // The next account to log in on this browser must not inherit this
      // one's group — the stale-id fallback would catch it, but only after a
      // request scoped to a group they may not belong to.
      clearActiveGroupId();
      // Clear the cached groups promise to prevent the next logged-in user from
      // seeing the previous user's group memberships/roles. Without this, the
      // in-memory promise cache would survive the logout and return stale data.
      clearMyGroupsCache();
      this.groups = [];
      this.activeGroupId = '';
      this.showGroupSwitcher = false;
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

.brand-text {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--primary-color);
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

.mobile-menu-toggle {
  display: none;
  flex-direction: column;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0.5rem;
  gap: 0.25rem;
}

.mobile-menu-toggle span {
  width: 1.5rem;
  height: 2px;
  background-color: var(--text-primary);
  transition: all var(--transition-fast);
  border-radius: 1px;
}

.mobile-menu-toggle.active span:nth-child(1) {
  transform: rotate(45deg) translate(0.375rem, 0.375rem);
}

.mobile-menu-toggle.active span:nth-child(2) {
  opacity: 0;
}

.mobile-menu-toggle.active span:nth-child(3) {
  transform: rotate(-45deg) translate(0.375rem, -0.375rem);
}

/* Main Content */
.app-container {
  background-color: var(--bg-secondary);
}

.main-content {
  margin-top: var(--navbar-height);
  min-height: calc(100vh - var(--navbar-height));
}

/* Tab Content */
.tab-content {
  padding: 2rem;
}

.tab-content .empty-state h2 {
  font-size: 2rem;
  margin-bottom: 0.5rem;
  color: var(--text-primary);
}

.tab-content .empty-state svg {
  color: var(--primary-color);
}

/* Mobile Overlay */
.mobile-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  z-index: 999;
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

.tab-content-enter-active,
.tab-content-leave-active {
  transition: all var(--transition-smooth);
}

.tab-content-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.tab-content-leave-to {
  opacity: 0;
  transform: translateY(-10px);
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

.overlay-enter-active,
.overlay-leave-active {
  transition: opacity var(--transition-smooth);
}

.overlay-enter-from,
.overlay-leave-to {
  opacity: 0;
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .mobile-menu-toggle {
    display: flex;
  }

  .nav-menu {
    position: fixed;
    top: var(--navbar-height);
    left: 0;
    right: 0;
    background-color: var(--bg-primary);
    flex-direction: column;
    padding: 2rem 1rem;
    gap: 1rem;
    transform: translateY(-100%);
    transition: transform var(--transition-smooth);
    box-shadow: var(--shadow-lg);
    max-height: calc(100vh - var(--navbar-height));
    overflow-y: auto;
  }

  .nav-menu.active {
    transform: translateY(0);
  }

  .nav-button {
    width: 100%;
    justify-content: center;
    padding: 1rem;
    font-size: 1.1rem;
  }

  .nav-container {
    padding: 0 1rem;
  }
}

@media (max-width: 480px) {
}
</style>