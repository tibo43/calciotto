import { createRouter, createWebHistory } from 'vue-router';
import { getToken } from '@/services/api';

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
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/components/Login.vue')
  },
  {
    path: '/signup',
    name: 'Signup',
    component: () => import('@/components/Signup.vue')
  }
];

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes,
});

router.beforeEach((to) => {
  const isPublic = to.name === 'Login' || to.name === 'Signup';
  if (!isPublic && !getToken()) {
    return { name: 'Login' };
  }
  return true;
});

export default router;
