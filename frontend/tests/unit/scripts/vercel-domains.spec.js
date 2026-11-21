import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { apiOriginsFromEnv, applyDomains, legacyRedirectFromEnv } from '../../../scripts/vercel-domains.mjs';

const root = (path) => resolve(import.meta.dirname, '../../../..', path);
const vercelJson = () => JSON.parse(readFileSync(root('frontend/vercel.json'), 'utf8'));

// devops/scripts/domain-env.sh with nothing overridden: the defaults from
// devops/env/domains.env, as deploy.yml derives them.
function defaultDomainEnv(overrides = {}) {
  const output = execFileSync('bash', [root('devops/scripts/domain-env.sh')], {
    env: { PATH: process.env.PATH, ...overrides },
    encoding: 'utf8',
  });
  return Object.fromEntries(output.trim().split('\n').map((line) => {
    const separator = line.indexOf('=');
    return [line.slice(0, separator), line.slice(separator + 1)];
  }));
}

const csp = (config) => config.headers[0].headers.find((h) => h.key === 'Content-Security-Policy').value;
const redirects = (config) => config.redirects;
const proxy = (config) => config.rewrites.find((r) => r.source === '/api/auth/:path*').destination;

describe('domain settings', () => {
  it('derives both frontends and both API hosts from domains.env', () => {
    const env = defaultDomainEnv();
    expect(env.CORS_ALLOWED_ORIGINS).toBe('https://club.agoran.app');
    expect(env.FRONTEND_BASE_URL).toBe('https://club.agoran.app');
    expect(env.VUE_APP_API_BASE_URL).toBe('https://api.agoran.app');
    expect(env.VUE_APP_API_BASE_URL_BY_HOST)
      .toBe('club.agoran.app=https://api.agoran.app');
    expect(env.AUTH_PROXY_TARGET).toBe('https://api.agoran.app');
    expect(env.LEGACY_FRONTEND_REDIRECT).toBe('true');
  });

  it('lets a set variable override its default', () => {
    const env = defaultDomainEnv({ NEW_FRONTEND_ORIGIN: 'https://club.example.app', CORS_ALLOWED_ORIGINS: 'https://only.test' });
    expect(env.CORS_ALLOWED_ORIGINS).toBe('https://only.test');
    expect(env.VUE_APP_API_BASE_URL_BY_HOST).toBe('club.example.app=https://api.agoran.app');
  });

  it('keeps the committed vercel.json in step with the defaults', () => {
    const env = defaultDomainEnv();
    const expected = applyDomains(vercelJson(), {
      apiOrigins: apiOriginsFromEnv(env),
      authProxyTarget: env.AUTH_PROXY_TARGET,
      legacyRedirect: legacyRedirectFromEnv(env),
    });
    expect(vercelJson()).toEqual(expected);
    expect(csp(expected)).toContain("connect-src 'self' https://api.agoran.app;");
    expect(proxy(expected)).toBe('https://api.agoran.app/auth/:path*');
    expect(redirects(expected)).toEqual([{
      source: '/((?!sw\\.js$).*)',
      has: [{ type: 'host', value: 'calcio-otto.vercel.app' }],
      destination: 'https://club.agoran.app/$1',
      permanent: false,
    }]);
  });

  it('goes back to both frontends side by side when LEGACY_FRONTEND_REDIRECT is false, and refuses anything else', () => {
    const env = defaultDomainEnv({ LEGACY_FRONTEND_REDIRECT: 'false' });
    expect(legacyRedirectFromEnv(env)).toBeNull();
    expect(env.CORS_ALLOWED_ORIGINS).toBe('https://calcio-otto.vercel.app,https://club.agoran.app');
    expect(env.VUE_APP_API_BASE_URL_BY_HOST)
      .toBe('club.agoran.app=https://api.agoran.app,calcio-otto.vercel.app=https://calciotto.koyeb.app');
    const next = applyDomains(vercelJson(), {
      apiOrigins: apiOriginsFromEnv(env),
      authProxyTarget: env.AUTH_PROXY_TARGET,
      legacyRedirect: legacyRedirectFromEnv(env),
    });
    expect(redirects(next)).toBeUndefined();
    expect(csp(next)).toContain("connect-src 'self' https://api.agoran.app https://calciotto.koyeb.app;");
    expect(() => defaultDomainEnv({ LEGACY_FRONTEND_REDIRECT: 'yes' })).toThrow();
  });

  it('rewrites the CSP and the auth proxy from other values', () => {
    const next = applyDomains(vercelJson(), {
      apiOrigins: apiOriginsFromEnv({ VUE_APP_API_BASE_URL: 'https://a.test', VUE_APP_API_BASE_URL_BY_HOST: 'x=https://b.test,y=https://a.test' }),
      authProxyTarget: 'https://b.test',
    });
    expect(csp(next)).toContain("connect-src 'self' https://a.test https://b.test;");
    expect(csp(next)).toContain("script-src 'self';");
    expect(proxy(next)).toBe('https://b.test/auth/:path*');
  });

  it('refuses anything but bare https origins', () => {
    expect(() => applyDomains(vercelJson(), { apiOrigins: ['https://a.test/'], authProxyTarget: 'https://a.test' })).toThrow();
    expect(() => applyDomains(vercelJson(), { apiOrigins: ['https://a.test'], authProxyTarget: 'http://a.test' })).toThrow();
    expect(() => applyDomains(vercelJson(), { apiOrigins: [], authProxyTarget: 'https://a.test' })).toThrow();
  });
});
