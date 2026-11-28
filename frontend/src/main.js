import { createApp } from 'vue'
import App from './App.vue'
import router from './router';
import { i18n } from './i18n';
import { initLocale } from './services/locale';
import { startLocaleSync, syncSavedLocale } from './services/playerLocale';
import { getInstallTracker } from './services/pwaInstall';
import { prefersDarkTheme } from './services/theme';
import 'bootstrap/dist/css/bootstrap.min.css';
import './assets/global-styles.css'
import './assets/design-v2.css'
import './assets/design-v2-ui.css'

// Before mounting: Chrome's install offer can arrive right after load, long
// before the Profile page that shows it is opened (see pwaInstall.js).
getInstallTracker();

// Production only: a service worker on the dev server would outlive restarts
// and get in the way of hot reload, and there is nothing to test there yet.
if (import.meta.env.PROD && 'serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register(`${import.meta.env.BASE_URL}sw.js`).catch((error) => {
      console.error('Service worker registration failed:', error);
    });
  });
}

const app = createApp(App);
app.use(router);
app.use(i18n);
// The phone's language first: nothing is stored on the device. A signed-in
// player's saved choice (players.locale) is read from the server right away and
// restored below, and later choices are saved (see services/playerLocale.js).
initLocale();
startLocaleSync();
const savedLocale = syncSavedLocale();
// Mount only once the initial route has matched. App.vue decides in created()
// whether this is a public page or a signed-in one, and the router's start
// location has no name, so mounting earlier would decide on the wrong route.
//
// Until then nothing is rendered, and a route guard can make that wait last
// (the edit route checks the caller's role first). The body carries
// the theme meanwhile, so a dark-mode viewer gets the dark background rather
// than a white page. App.vue owns the theme from mount on, on its own root.
const darkBeforeMount = prefersDarkTheme();
if (darkBeforeMount) {
  document.body.classList.add('dark-mode');
}
// The saved language gets a short head start on the first paint, so that a
// player who chose another language than their phone's does not see the
// phone's flash by. A slow server never holds the app longer than this.
const SAVED_LOCALE_WAIT_MS = 500;
router.isReady().then(async () => {
  await Promise.race([savedLocale, new Promise((resolve) => setTimeout(resolve, SAVED_LOCALE_WAIT_MS))]);
  app.mount('#app');
  document.body.classList.remove('dark-mode');
});
