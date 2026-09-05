import { createRouter, createWebHistory } from 'vue-router';
import { getToken, getMatchDetailsByID } from '@/services/api';
import { decodeMatchId } from '@/services/shortLink';
import { loadMyGroups } from '@/services/activeGroup';

// The match's own group isn't known ahead of time from a bare match id (it
// only ever appears alone, in a URL or a `/m/:code` link), and
// GetMatchDetailsByID resolves an unspecified group_id to the caller's
// *first* group — which silently 404s the moment the match belongs to any
// other group the caller is in. So this tries each of the caller's groups,
// from GET /groups/me, passing it explicitly as group_id until the match is
// found in one of them. A 404 for one group just means the match isn't in
// it, so the search continues; any other failure aborts the search and
// propagates, since it isn't "not this group" but "cannot verify". Returns
// null if the match belongs to no group in the given list at all — either it
// belongs to a group the caller isn't a member of, or the id names no match.
//
// Shared by canEditMatch below and MatchesAndStandings.vue's own resolution
// of a `?match=<code>` deep link (see the `/m/:code` route further down) —
// both need exactly this "which of my groups owns this match" search, one to
// decide an authorization question, the other to decide which group to
// switch into and which season to preselect.
export async function findGroupForMatch(matchId, groups) {
  for (const group of groups) {
    try {
      const details = await getMatchDetailsByID(matchId, group.id);
      return { group, details };
    } catch (error) {
      if (error?.response?.status !== 404) {
        throw error;
      }
      // Not this group — the match may still belong to another one the
      // caller is in, so keep looking rather than giving up here.
    }
  }
  return null;
}

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
// Uses findGroupForMatch above (the caller's groups from GET /groups/me, the
// exact source MatchesAndStandings.vue/MatchDetails.vue already read their
// own role from — see resolveActiveGroup()) and checks that group's role. Any
// failure — no owning group found, or a non-404 error propagated from the
// search — is treated as "cannot verify" one level up, in beforeEnter below —
// failing closed (redirect) rather than open.
export async function canEditMatch(matchId) {
  const groups = await loadMyGroups();
  const found = await findGroupForMatch(matchId, groups);
  return found ? found.group.role === 'admin' : false;
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
  //
  // This used to redirect straight to `/matches/:id/edit`, but that page is
  // admin-only now (see canEditMatch above) — a plain member following a
  // shared link would just bounce straight back to `/`, which defeats the
  // point of sharing it. Landing on the home page's Matches tab instead is
  // strictly more correct today than the old destination ever was: the
  // sign-up list, the confirmed/waiting roster and Man of the Match voting
  // all live in MatchesPanel.vue now, reachable by admin and member alike, so
  // there is no reason left to route a recipient anywhere else. The code is
  // forwarded as-is in `?match=`, not decoded and re-encoded here —
  // MatchesAndStandings.vue does the actual decoding (via
  // findGroupForMatch above) once it knows the caller's groups. decodeMatchId
  // is still called here, once, purely to validate the code the same way the
  // old redirect did — a malformed one falls back home rather than handing a
  // garbage `?match=` value down to the page.
  {
    path: '/m/:code',
    redirect: (to) => {
      try {
        decodeMatchId(to.params.code);
        return { path: '/', query: { match: to.params.code } };
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
