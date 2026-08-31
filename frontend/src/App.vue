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
        <main class="main-content">
          <router-view />
        </main>
    </div>
  </div>
</template>

<script>
import { clearToken } from '@/services/api';
import { clearActiveGroupId, clearMyGroupsCache } from '@/services/activeGroup';

// Routes reachable without a token — nothing group-related is shown there.
const PUBLIC_ROUTE_NAMES = ['Login', 'Signup', 'ForgotPassword', 'ResetPassword'];

export default {
  name: 'App',
  data() {
    return {
      isScrolled: false,
      isDarkMode: false
    };
  },
  computed: {
    // Gates the nav-menu links (Matches/Groups) and the Profile/Logout icons:
    // all three require a token, so showing them on a public route (where
    // there's no session to act on) makes a standalone login/signup page
    // read as the logged-in app shell instead of its own view.
    isAuthenticatedRoute() {
      return !PUBLIC_ROUTE_NAMES.includes(this.$route.name);
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

  .nav-button-label {
    display: none;
  }

  .nav-button {
    padding: 0.5rem;
  }

  .nav-menu {
    gap: 0.25rem;
  }

  .nav-actions {
    gap: 0.25rem;
  }
}

@media (max-width: 400px) {
  .brand-text {
    font-size: 1.15rem;
  }
}
</style>