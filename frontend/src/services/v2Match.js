import { t } from '@/i18n';
import { resolveActiveGroup } from '@/services/activeGroup';
import { findGroupForMatch } from '@/router/index';

// Loads a match from a bare id (the v2 match and edit pages, a deep link) in
// the group it actually belongs to. GET /matches/:id/details without a
// group_id is resolved against the caller's *first* group by the backend, so
// a player in several groups would get a 404 for every match of the others
// (see findGroupForMatch). The active group is tried first: it is almost
// always the right one, so the common case still costs a single request.
export async function findMatchInMyGroups(matchId) {
  const { groups, activeGroupId } = await resolveActiveGroup();
  const ordered = [
    ...groups.filter((group) => group.id === activeGroupId),
    ...groups.filter((group) => group.id !== activeGroupId),
  ];
  const found = await findGroupForMatch(matchId, ordered);
  if (!found) {
    throw new Error(t('match.notInGroups'));
  }
  return { ...found, groups };
}

// The backend's own reason when it gives one (a 409 says exactly why: list
// closed, voting over…), the caller's fallback otherwise — the same rule as
// today's Matches tab.
export function actionErrorMessage(error, fallback) {
  return error?.response?.data?.error || fallback;
}
