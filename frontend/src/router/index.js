import { createRouter, createWebHistory } from 'vue-router';
import { getToken } from '@/services/api';

const routes = [
  {
    path: '/',
    name: 'MatchesAndStandings',
    component: () => import('@/components/MatchesAndStandings.vue'),
    props: true
  },
  {
    path: '/matches/:id/edit',
    name: 'MatchDetails',
    component: () => import('@/components/MatchDetails.vue'),
    props: true
  },
  // Matches and standings used to be two pages; they are one now. Kept as a
  // redirect so an old bookmark or link still lands on the merged page rather
  // than a blank 404.
  {
    path: '/standings',
    redirect: '/'
  },
  {
    path: '/groups',
    name: 'Groups',
    component: () => import('@/components/Groups.vue')
  },
  {
    path: '/profile',
    name: 'Profile',
    component: () => import('@/components/Profile.vue')
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
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: () => import('@/components/ForgotPassword.vue')
  },
  {
    path: '/reset-password',
    name: 'ResetPassword',
    component: () => import('@/components/ResetPassword.vue')
  }
];

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes,
});

router.beforeEach((to) => {
  // The password-reset pages are public for the same reason Login/Signup are:
  // someone who lost their password has no token to be let through with.
  const publicRoutes = ['Login', 'Signup', 'ForgotPassword', 'ResetPassword'];
  const isPublic = publicRoutes.includes(to.name);
  if (!isPublic && !getToken()) {
    return { name: 'Login' };
  }
  return true;
});

export default router;
