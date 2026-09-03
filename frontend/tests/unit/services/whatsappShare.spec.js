import { buildWhatsAppShareText, buildWhatsAppShareUrl } from '@/services/whatsappShare';

describe('buildWhatsAppShareText', () => {
  it('includes the kick-off, the sign-up count and the match link', () => {
    const text = buildWhatsAppShareText({
      kickoffLabel: 'Sun, Sep 6, 2026, 8:30 PM',
      signupCountLabel: '12 / 16 signed up',
      matchUrl: 'https://calciotto.app/matches/abc-123/edit'
    });

    expect(text).toContain('Sun, Sep 6, 2026, 8:30 PM');
    expect(text).toContain('12 / 16 signed up');
    expect(text).toContain('https://calciotto.app/matches/abc-123/edit');
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
