import Vue from 'vue';
import VueRouter from 'vue-router';
import PlayersAll from '@/components/PlayersAll.vue';
import PlayerDetails from '@/components/PlayerDetails.vue';

Vue.use(VueRouter);

const routes = [
  { path: '/players', component: PlayersAll },
  { path: '/players/:id', component: PlayerDetails },
];

const router = new VueRouter({
  mode: 'history',
  routes,
});

export default router;
