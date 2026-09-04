import { createRouter, createWebHistory } from 'vue-router';
import { getToken, getMatchDetailsByID } from '@/services/api';
import { decodeMatchId } from '@/services/shortLink';
import { loadMyGroups } from '@/services/activeGroup';

// MatchDetails.vue (the editing page) is admin-only now: every control on it
// was already hidden from a non-admin (see MatchDetails.vue's own isAdmin
// gating), but a plain member could still land on the page itself and see a
// mostly-blank shell. MatchesPanel.vue — reachable by any member — covers
// reading a match (including, since this same product change, its full
// sign-up list and Man of the Match voting), so a non-admin has no remaining
// reason to be here at all.
//
// This is a FRONTEND-only gate. GET /matches/:id/details itself stays open to
// any group member on the backend — MatchesPanel.vue depends on reading it —
// so this does not (and must not) tighten that route.
//
// The match's own group isn't known ahead of time from the URL alone (it only
// carries a match id), and GetMatchDetailsByID resolves an unspecified
// group_id to the caller's *first* group — which silently answers "not
// found" the moment the match belongs to any other group the caller is in.
// So this tries each of the caller's groups, from GET /groups/me (the exact
// source MatchesAndStandings.vue/MatchDetails.vue already read their own role
// from — see resolveActiveGroup()), passing it explicitly as group_id until
// the match is found in one of them, then checks that group's role. A 404 for
// one group just means the match isn't in it, so the search continues; any
// other failure aborts the search and is treated as "cannot verify" one level
// up, in beforeEnter below — failing closed (redirect) rather than open.
export async function canEditMatch(matchId) {
  const groups = await loadMyGroups();
  for (const group of groups) {
    try {
      await getMatchDetailsByID(matchId, group.id);
      return group.role === 'admin';
    } catch (error) {
      if (error?.response?.status !== 404) {
        throw error;
      }
      // Not this group — the match may still belong to another one the
      // caller is in, so keep looking rather than giving up here.
    }
  }
  // No group of the caller's contains this match at all — either it belongs
  // to a group they aren't a member of, or the id names no match.
  return false;
}

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
    props: true,
    beforeEnter: async (to) => {
      try {
        const allowed = await canEditMatch(to.params.id);
        if (!allowed) {
          return { path: '/' };
        }
      } catch (error) {
        // Same degrade-instead-of-break contract as resolveActiveGroup()'s own
        // callers: a failure here (a network error, a malformed id, ...) is
        // treated the same as "not allowed" rather than letting a half-broken
        // navigation through.
        console.error('Error checking match edit access:', error);
        return { path: '/' };
      }
      return true;
    }
  },
  // The "tinylink" shared on WhatsApp (see whatsappShare.js) — a bare
  // redirect rather than its own component, since decoding a code and
  // resolving where it points to is all there is to do. An invalid code
  // (hand-edited, truncated in transit) falls back home instead of throwing
  // partway through a navigation.
  {
    path: '/m/:code',
    redirect: (to) => {
      try {
        return `/matches/${decodeMatchId(to.params.code)}/edit`;
      } catch {
        return '/';
      }
    }
  },
  // Matches and standings used to be two pages; they are one now. Kept as a
  // redirect so an old bookmark or link still lands on the merged page rather
  // than a blank 404.
  {
    path: '/standings',
    redirect: '/'
  },
  // Groups used to be its own page; group management (roster, roles, invite
  // code, teams) is now part of Profile. Kept as a redirect so an old
  // bookmark or link still lands on the merged page rather than a 404.
  {
    path: '/groups',
    redirect: '/profile'
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
