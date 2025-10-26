<template>
  <div id="app" :class="{ 'dark-mode': isDarkMode }">
    <!-- Initial Logo View -->
    <transition name="initial-view">
      <div v-if="showImage" class="initial-view" @click="toggleView">
        <div class="logo-container">
          <img alt="Calciotto Logo" src="./assets/campo.jpg" class="hero-image">
          <div class="logo-overlay">
            <h1 class="title-logo">Calciotto</h1>
            <p class="subtitle">Click to enter</p>
            <div class="pulse-indicator"></div>
          </div>
        </div>
      </div>
    </transition>

    <!-- Main App Container -->
    <transition name="app-container">
      <div v-if="!showImage" class="app-container">
        <!-- Top Navigation -->
        <nav class="top-navbar" :class="{ 'scrolled': isScrolled }">
          <div class="nav-container">
            <!-- Logo/Brand -->
            <div class="nav-brand" @click="goHome">
              <img src="@/assets/logo.png" alt="Logo" class="nav-logo">
              <span class="brand-text">Calciotto</span>
            </div>

            <!-- Desktop Menu -->
            <div class="nav-menu" :class="{ 'active': isMenuOpen }">
              <router-link 
                to="/" 
                @click="closeMenu"
                :class="{ 'active': $route.name === 'MatchesAll' }"
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
              <button 
                @click="setActiveTab('Standings')" 
                :class="{ 'active': activeTab === 'Standings' }"
                class="nav-button"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
                  <circle cx="9" cy="7" r="4"/>
                  <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
                  <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
                </svg>
                Standings
              </button>
            </div>

            <!-- Actions -->
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

              <button 
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
    </transition>
  </div>
</template>

<script>
export default {
  name: 'App',
  data() {
    return {
      showImage: true,
      activeTab: 'matches',
      isMenuOpen: false,
      isScrolled: false,
      isDarkMode: false
    };
  },
  computed: {
    isRouterRoute() {
      // Check if current route should be handled by router
      return this.$route.name === 'MatchesAll' || this.$route.name === 'MatchDetails';
    }
  },
  watch: {
    '$route'(to) {
      // Update activeTab when route changes
      if (to.name === 'MatchesAll') {
        this.activeTab = 'matches';
      }
      this.closeMenu();
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
    toggleView() {
      this.showImage = !this.showImage;
      if (!this.showImage) {
        // Navigate to matches when entering app
        this.$router.push('/');
        this.activeTab = 'matches';
      }
    },
    goHome() {
      this.$router.push('/');
      this.activeTab = 'matches';
    },
    setActiveTab(tab) {
      this.activeTab = tab;
      this.closeMenu();
      
      // If selecting matches tab, navigate to home route
      if (tab === 'matches') {
        this.$router.push('/');
      }
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

/* Initial View Styles - Component-specific */
.initial-view {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--secondary-color) 100%);
  cursor: pointer;
  z-index: 9999;
}

.logo-container {
  position: relative;
  text-align: center;
  transform: scale(1);
  transition: transform var(--transition-smooth);
}

.initial-view:hover .logo-container {
  transform: scale(1.05);
}

.hero-image {
  width: 300px;
  height: 300px;
  border-radius: 50%;
  object-fit: cover;
  box-shadow: var(--shadow-xl);
  border: 4px solid rgba(255, 255, 255, 0.2);
}

.logo-overlay {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: white;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.title-logo {
  font-size: 3rem;
  font-weight: 800;
  margin-bottom: 0.5rem;
  background: linear-gradient(45deg, #ffffff, #f0f9ff);
  background-clip: text;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  text-shadow: none;
}

.subtitle {
  font-size: 1.1rem;
  opacity: 0.9;
  margin-bottom: 1rem;
}

.pulse-indicator {
  width: 12px;
  height: 12px;
  background-color: #ffffff;
  border-radius: 50%;
  margin: 0 auto;
  animation: pulse 2s infinite;
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

.nav-logo {
  height: 40px;
  width: auto;
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
.initial-view-enter-active,
.initial-view-leave-active {
  transition: all var(--transition-smooth);
}

.initial-view-enter-from,
.initial-view-leave-to {
  opacity: 0;
  transform: scale(0.9);
}

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

  .hero-image {
    width: 250px;
    height: 250px;
  }

  .title-logo {
    font-size: 2.5rem;
  }

  .nav-container {
    padding: 0 1rem;
  }
}

@media (max-width: 480px) {
  .hero-image {
    width: 200px;
    height: 200px;
  }

  .title-logo {
    font-size: 2rem;
  }
}
</style>