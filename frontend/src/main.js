import { createApp } from 'vue'
import App from './App.vue'
import router from './router';
import { getInstallTracker } from './services/pwaInstall';
import 'bootstrap/dist/css/bootstrap.min.css';
import './assets/global-styles.css'
import './assets/design-v2.css'

// Before mounting: Chrome's install offer can arrive right after load, long
// before the Profile page that shows it is opened (see pwaInstall.js).
getInstallTracker();

// Production only: a service worker on the dev server would outlive restarts
// and get in the way of hot reload, and there is nothing to test there yet.
if (process.env.NODE_ENV === 'production' && 'serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register(`${process.env.BASE_URL}sw.js`).catch((error) => {
      console.error('Service worker registration failed:', error);
    });
  });
}

const app = createApp(App);
app.use(router);
// Mount only once the initial route has matched. App.vue applies the cached
// design-v2 class in created(), and the router's start location has no name,
// so mounting earlier would skip that cache and flash today's design.
router.isReady().then(() => {
  app.mount('#app');
});
