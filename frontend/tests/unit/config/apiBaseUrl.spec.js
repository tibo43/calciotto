import { LOCAL_API_BASE_URL, parseApiBaseUrlByHost, resolveApiBaseUrl } from '@/config/apiBaseUrl';

const BY_HOST = 'club.agoran.app=https://api.agoran.app,calcio-otto.vercel.app=https://calciotto.koyeb.app';

describe('apiBaseUrl', () => {
  it('sends each frontend domain to its own API host', () => {
    expect(resolveApiBaseUrl('club.agoran.app', BY_HOST, 'https://calciotto.koyeb.app')).toBe('https://api.agoran.app');
    expect(resolveApiBaseUrl('calcio-otto.vercel.app', BY_HOST, 'https://fallback.test')).toBe('https://calciotto.koyeb.app');
    expect(resolveApiBaseUrl('CLUB.AGORAN.APP', BY_HOST, '')).toBe('https://api.agoran.app');
  });

  it('falls back to VUE_APP_API_BASE_URL, then to the local API', () => {
    expect(resolveApiBaseUrl('preview-123.vercel.app', BY_HOST, 'https://calciotto.koyeb.app')).toBe('https://calciotto.koyeb.app');
    expect(resolveApiBaseUrl('localhost', undefined, undefined)).toBe(LOCAL_API_BASE_URL);
    expect(resolveApiBaseUrl('localhost', '', 'http://127.0.0.1:8080')).toBe('http://127.0.0.1:8080');
  });

  it('ignores malformed pairs and trailing slashes', () => {
    expect([...parseApiBaseUrlByHost(' a.test = https://api.a.test/ ,broken,=https://x.test,b.test=not-a-url,c.test=https://c.test/path')])
      .toEqual([['a.test', 'https://api.a.test']]);
  });
});
