import { t } from '@/i18n';

// Put the link on its own line so it stays clickable in common share targets.
export function buildMatchShareText({ kickoffLabel, matchUrl, groupInviteCode, signupsOpen = true }) {
  const lines = [
    t('share.match', { when: kickoffLabel }),
    signupsOpen ? t('share.signupsOpen') : t('share.viewMatch'),
    matchUrl
  ];
  if (groupInviteCode) {
    lines.push(t('share.groupCode', { code: groupInviteCode }));
  }
  return lines.join('\n');
}

export function buildGroupInviteShareText({ inviteUrl, groupInviteCode }) {
  return [
    t('share.joinGroup'),
    inviteUrl,
    t('share.groupCode', { code: groupInviteCode })
  ].join('\n');
}

export function buildTeamsShareText({ dateTimeLabel, teams }) {
  const lines = [t('share.match', { when: dateTimeLabel })];
  for (const team of teams) {
    lines.push('', team.name);
    if (team.players.length === 0) {
      lines.push(t('share.noPlayers'));
    } else {
      for (const player of team.players) {
        lines.push(t('share.player', { name: player }));
      }
    }
  }
  return lines.join('\n');
}
