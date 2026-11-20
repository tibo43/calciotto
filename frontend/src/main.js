import { createApp } from 'vue'
import App from './App.vue'
import router from './router';
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
// Mount only once the initial route has matched. App.vue decides in created()
// whether this is a public page or a signed-in one, and the router's start
// location has no name, so mounting earlier would decide on the wrong route.
//
// Until then nothing is rendered, and a route guard can make that wait last
// (the v2-only routes ask GET /players/me/features first). The body carries
// the theme meanwhile, so a dark-mode viewer gets the dark background rather
// than a white page. App.vue owns the theme from mount on, on its own root.
const darkBeforeMount = prefersDarkTheme();
if (darkBeforeMount) {
  document.body.classList.add('dark-mode');
}
router.isReady().then(() => {
  app.mount('#app');
  document.body.classList.remove('dark-mode');
});
