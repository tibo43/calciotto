import {
  buildMatchShareText,
  buildGroupInviteShareText,
  buildTeamsShareText
} from '@/services/shareMessages';

describe('buildMatchShareText', () => {
  it('includes the kick-off and the match link', () => {
    const text = buildMatchShareText({
      kickoffLabel: 'Sun, Sep 6, 2026, 8:30 PM',
      matchUrl: 'https://calciotto.app/matches/abc-123/edit'
    });

    expect(text).toBe(
      'Match: Sun, Sep 6, 2026, 8:30 PM\n'
      + 'Sign-ups are open — Join Agoran:\n'
      + 'https://calciotto.app/matches/abc-123/edit'
    );
  });

  it('puts the URL on its own line rather than hiding it behind anchor text', () => {
    const text = buildMatchShareText({
      kickoffLabel: 'Sun, Sep 6, 2026, 8:30 PM',
      matchUrl: 'https://calciotto.app/matches/abc-123/edit'
    });

    const lines = text.split('\n');
    expect(lines[lines.length - 1]).toBe('https://calciotto.app/matches/abc-123/edit');
  });

  it('uses a general match invitation when sign-ups are closed', () => {
    const text = buildMatchShareText({
      kickoffLabel: 'Sun, Sep 6, 2026, 8:30 PM',
      matchUrl: 'https://calciotto.app/m/abc',
      signupsOpen: false
    });

    expect(text).toContain('View the match in Agoran:');
    expect(text).not.toContain('Sign-ups are open');
  });

  it('appends the group invite code on its own trailing line when given one', () => {
    const text = buildMatchShareText({
      kickoffLabel: 'Sun, Sep 6, 2026, 8:30 PM',
      matchUrl: 'https://calciotto.app/matches/abc-123/edit',
      groupInviteCode: 'ABC23XYZ'
    });

    const lines = text.split('\n');
    expect(lines[lines.length - 1]).toBe('Group code: ABC23XYZ');
  });

  // A failed fetch (see MatchDetails.vue's created()) leaves groupInviteCode
  // as '' — this must read as "nothing to add", not as a line reading
  // "Group code: " with nothing after the colon.
  it('omits the group-code line entirely when no code is given', () => {
    const withEmptyCode = buildMatchShareText({
      kickoffLabel: 'Sun, Sep 6, 2026, 8:30 PM',
      matchUrl: 'https://calciotto.app/matches/abc-123/edit',
      groupInviteCode: ''
    });
    const withNoCodeAtAll = buildMatchShareText({
      kickoffLabel: 'Sun, Sep 6, 2026, 8:30 PM',
      matchUrl: 'https://calciotto.app/matches/abc-123/edit'
    });

    expect(withEmptyCode).not.toContain('Group code');
    expect(withEmptyCode).toBe(withNoCodeAtAll);
  });
});

describe('buildGroupInviteShareText', () => {
  it('includes the invite URL on its own line', () => {
    const text = buildGroupInviteShareText({
      inviteUrl: 'https://calciotto.app/signup?invite=AB2N7TQR',
      groupInviteCode: 'AB2N7TQR'
    });

    const lines = text.split('\n');
    expect(lines).toContain('https://calciotto.app/signup?invite=AB2N7TQR');
  });

  it('includes a "Group code: X" line, even though the code also rides along in the URL', () => {
    const text = buildGroupInviteShareText({
      inviteUrl: 'https://calciotto.app/signup?invite=AB2N7TQR',
      groupInviteCode: 'AB2N7TQR'
    });

    const lines = text.split('\n');
    expect(lines[lines.length - 1]).toBe('Group code: AB2N7TQR');
  });
});

describe('buildTeamsShareText', () => {
  it('lists the kick-off, then each team name followed by its players', () => {
    const text = buildTeamsShareText({
      dateTimeLabel: 'Sun, Sep 6, 2026, 8:30 PM',
      teams: [
        { name: 'Black', players: ['Marco', 'Luca'] },
        { name: 'White', players: ['Anna'] }
      ]
    });

    expect(text).toBe(
      'Match: Sun, Sep 6, 2026, 8:30 PM\n'
      + '\n'
      + 'Black\n'
      + '- Marco\n'
      + '- Luca\n'
      + '\n'
      + 'White\n'
      + '- Anna'
    );
  });

  it('prints "No players yet" for a team with no players instead of a bare header', () => {
    const text = buildTeamsShareText({
      dateTimeLabel: 'Sun, Sep 6, 2026, 8:30 PM',
      teams: [
        { name: 'Black', players: [] },
        { name: 'White', players: ['Anna'] }
      ]
    });

    const lines = text.split('\n');
    expect(lines).toEqual([
      'Match: Sun, Sep 6, 2026, 8:30 PM',
      '',
      'Black',
      'No players yet',
      '',
      'White',
      '- Anna'
    ]);
  });
});
