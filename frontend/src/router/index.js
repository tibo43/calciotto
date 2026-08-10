import { createRouter, createWebHistory } from 'vue-router';

const routes = [
  {
    path: '/',
    name: 'MatchesAll',
    component: () => import('@/components/MatchesAll.vue'),
    props: true
  },
  {
    path: '/matches/:id/edit',
    name: 'MatchDetails',
    component: () => import('@/components/MatchDetails.vue'),
    props: true
  },
  {
    path: '/standings',
    name: 'Standings',
    component: () => import('@/components/Standings.vue')
  }
];

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes,
});

export default router;
