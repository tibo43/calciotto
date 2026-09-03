import { buildWhatsAppShareText, buildWhatsAppShareUrl } from '@/services/whatsappShare';

describe('buildWhatsAppShareText', () => {
  it('includes the kick-off and the match link, with no emoji WhatsApp might not render', () => {
    const text = buildWhatsAppShareText({
      kickoffLabel: 'Sun, Sep 6, 2026, 8:30 PM',
      matchUrl: 'https://calciotto.app/matches/abc-123/edit'
    });

    expect(text).toBe(
      'Match: Sun, Sep 6, 2026, 8:30 PM\n'
      + 'Sign-ups are open — Join Calciotto:\n'
      + 'https://calciotto.app/matches/abc-123/edit'
    );
  });

  // WhatsApp text messages have no link-with-custom-text syntax — only a raw
  // URL gets auto-linkified — so "Join Calciotto" can only ever be a label,
  // never the tappable text itself.
  it('puts the URL on its own line rather than hiding it behind anchor text', () => {
    const text = buildWhatsAppShareText({
      kickoffLabel: 'Sun, Sep 6, 2026, 8:30 PM',
      matchUrl: 'https://calciotto.app/matches/abc-123/edit'
    });

    const lines = text.split('\n');
    expect(lines[lines.length - 1]).toBe('https://calciotto.app/matches/abc-123/edit');
  });
});

describe('buildWhatsAppShareUrl', () => {
  it('builds a wa.me link with no phone number, so WhatsApp asks who to send it to', () => {
    const url = buildWhatsAppShareUrl('hello world');

    expect(url).toBe('https://wa.me/?text=hello%20world');
  });

  it('percent-encodes newlines and special characters', () => {
    const url = buildWhatsAppShareUrl('line one\nline two & more');

    expect(url).toBe('https://wa.me/?text=' + encodeURIComponent('line one\nline two & more'));
    expect(url).not.toContain('\n');
  });
});
