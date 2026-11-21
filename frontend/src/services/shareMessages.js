// Put the link on its own line so it stays clickable in common share targets.
export function buildMatchShareText({ kickoffLabel, matchUrl, groupInviteCode, signupsOpen = true }) {
  const lines = [
    `Match: ${kickoffLabel}`,
    signupsOpen ? 'Sign-ups are open — Join Agoran:' : 'View the match in Agoran:',
    matchUrl
  ];
  if (groupInviteCode) {
    lines.push(`Group code: ${groupInviteCode}`);
  }
  return lines.join('\n');
}

export function buildGroupInviteShareText({ inviteUrl, groupInviteCode }) {
  return [
    'Join our Agoran group:',
    inviteUrl,
    `Group code: ${groupInviteCode}`
  ].join('\n');
}

export function buildTeamsShareText({ dateTimeLabel, teams }) {
  const lines = [`Match: ${dateTimeLabel}`];
  for (const team of teams) {
    lines.push('', team.name);
    if (team.players.length === 0) {
      lines.push('No players yet');
    } else {
      for (const player of team.players) {
        lines.push(`- ${player}`);
      }
    }
  }
  return lines.join('\n');
}
