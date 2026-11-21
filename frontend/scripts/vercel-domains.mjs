// Writes the domain settings into vercel.json before `vercel build`
// (deploy.yml): the CSP's connect-src lists every API origin the bundle may
// call, the same-origin /api/auth proxy points at AUTH_PROXY_TARGET, and,
// while LEGACY_FRONTEND_REDIRECT is true, the legacy frontend host redirects
// every request to the new frontend. vercel.json is
// static JSON, so this is how the domains stay environment variables
// (devops/scripts/domain-env.sh derives them from devops/env/domains.env).
//
//   node scripts/vercel-domains.mjs           rewrite vercel.json from the env
//   node scripts/vercel-domains.mjs --check   fail if it differs
import { readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const AUTH_SOURCE = '/api/auth/:path*';

const isOrigin = (value) => /^https:\/\/[a-z0-9.-]+$/.test(value);

// Every API origin the bundle may call: VUE_APP_API_BASE_URL and each value
// of VUE_APP_API_BASE_URL_BY_HOST, in order, without duplicates.
export function apiOriginsFromEnv(env) {
  const origins = [env.VUE_APP_API_BASE_URL];
  for (const pair of String(env.VUE_APP_API_BASE_URL_BY_HOST || '').split(',')) {
    const separator = pair.indexOf('=');
    if (separator >= 0) origins.push(pair.slice(separator + 1).trim());
  }
  return [...new Set(origins.filter(Boolean))];
}

// The legacy frontend's redirect, or null when it is off. Read from the
// settings domain-env.sh prints.
export function legacyRedirectFromEnv(env) {
  if (env.LEGACY_FRONTEND_REDIRECT !== 'true') return null;
  return { from: env.LEGACY_FRONTEND_ORIGIN, to: env.NEW_FRONTEND_ORIGIN };
}

export function applyDomains(config, { apiOrigins, authProxyTarget, legacyRedirect = null }) {
  const origins = [...apiOrigins, authProxyTarget];
  if (legacyRedirect) origins.push(legacyRedirect.from, legacyRedirect.to);
  for (const origin of origins) {
    if (!isOrigin(origin)) throw new Error(`not a bare https origin: ${origin}`);
  }
  if (apiOrigins.length === 0) throw new Error('no API origin');
  const next = structuredClone(config);
  let csp = 0;
  for (const rule of next.headers || []) {
    for (const header of rule.headers || []) {
      if (header.key !== 'Content-Security-Policy') continue;
      header.value = header.value.replace(/connect-src [^;]*/, `connect-src 'self' ${apiOrigins.join(' ')}`);
      csp += 1;
    }
  }
  const proxy = (next.rewrites || []).find((rewrite) => rewrite.source === AUTH_SOURCE);
  if (csp !== 1 || !proxy) throw new Error('vercel.json: expected one Content-Security-Policy and the /api/auth rewrite');
  proxy.destination = `${authProxyTarget}/auth/:path*`;
  // The script owns `redirects`: only the legacy host's, or none. Vercel
  // applies redirects before rewrites and keeps the query string, so a
  // shared /m/:code, /signup?invite= or /reset-password?token= link lands on
  // the same page of the new domain. 307, not 308: a permanent redirect is
  // cached by browsers and could not be taken back.
  delete next.redirects;
  if (legacyRedirect) {
    next.redirects = [{
      source: '/:path*',
      has: [{ type: 'host', value: new URL(legacyRedirect.from).hostname }],
      destination: `${legacyRedirect.to}/:path*`,
      permanent: false,
    }];
  }
  return next;
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const file = resolve(import.meta.dirname, '../vercel.json');
  const current = readFileSync(file, 'utf8');
  const updated = `${JSON.stringify(applyDomains(JSON.parse(current), {
    apiOrigins: apiOriginsFromEnv(process.env),
    authProxyTarget: process.env.AUTH_PROXY_TARGET,
    legacyRedirect: legacyRedirectFromEnv(process.env),
  }), null, 2)}\n`;
  if (process.argv.includes('--check')) {
    if (updated !== current) {
      console.error('vercel.json does not match the domain settings');
      process.exit(1);
    }
  } else {
    writeFileSync(file, updated);
  }
  console.log(updated.match(/connect-src [^;]*/)[0]);
  console.log(`/api/auth -> ${process.env.AUTH_PROXY_TARGET}`);
  const redirect = legacyRedirectFromEnv(process.env);
  console.log(redirect ? `redirect ${redirect.from} -> ${redirect.to}` : 'no legacy redirect');
}
