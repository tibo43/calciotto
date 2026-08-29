import { getMyGroups } from './api';

// Which group the app is currently scoped to. This is a per-device
// preference, not server state — the backend has no notion of an "active"
// group, it just takes whatever group_id a request carries (see
// resolveGroupID in internal/handlers/groupscope.go) — so localStorage is the
// right home for it, same as the auth token in api.js.
const ACTIVE_GROUP_KEY = 'calciotto-active-group';

export const getActiveGroupId = () => localStorage.getItem(ACTIVE_GROUP_KEY) || '';
export const setActiveGroupId = (groupId) => localStorage.setItem(ACTIVE_GROUP_KEY, groupId);
export const clearActiveGroupId = () => localStorage.removeItem(ACTIVE_GROUP_KEY);

// GET /groups/me is the only way to tell whether the stored id still points at
// a group the player belongs to, and every scoped view needs that answer
// before its first request. The active group only ever changes through a full
// page reload, so caching the in-flight promise for the lifetime of the page
// keeps this to a single request no matter how many callers ask. A failure is
// not cached — the next caller retries.
let myGroupsPromise = null;

export function loadMyGroups({ force = false } = {}) {
  if (force || !myGroupsPromise) {
    myGroupsPromise = getMyGroups()
      .then((groups) => (Array.isArray(groups) ? groups : []))
      .catch((error) => {
        myGroupsPromise = null;
        throw error;
      });
  }
  return myGroupsPromise;
}

// Clear the cached groups promise. This is needed on logout/login transitions
// where the active group changes but the page does not reload — the module-scope
// cache would otherwise return the previous user's groups/roles to the new user.
export function clearMyGroupsCache() {
  myGroupsPromise = null;
}

// The stored id can go stale: the player left the group, or another account
// used this browser. Sending it anyway would scope every request to a group
// the caller isn't a member of, which RequireGroupMembership rejects — so
// fall back to the first group of the list instead, and rewrite the stored
// preference so the nav selector and the views agree on the same group.
export async function resolveActiveGroup(options) {
  const groups = await loadMyGroups(options);
  const stored = getActiveGroupId();
  const active = groups.find((group) => group.id === stored) || groups[0] || null;

  if (!active) {
    clearActiveGroupId();
  } else if (active.id !== stored) {
    setActiveGroupId(active.id);
  }

  return { groups, activeGroupId: active ? active.id : '' };
}

// What the views call: they only need the id. An empty string means "no
// group_id to send", which leaves the backend on its own GetFirstGroupForPlayer
// fallback — the exact behaviour that existed before this selector, so a
// failure here degrades instead of taking the view down.
export async function resolveActiveGroupId() {
  try {
    const { activeGroupId } = await resolveActiveGroup();
    return activeGroupId;
  } catch (error) {
    console.error('Error resolving the active group:', error);
    return '';
  }
}
