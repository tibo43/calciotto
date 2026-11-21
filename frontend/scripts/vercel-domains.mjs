// Writes the API hosts into vercel.json before `vercel build` (deploy.yml):
// the CSP's connect-src lists every API origin the bundle may call, and the
// same-origin /api/auth proxy points at AUTH_PROXY_TARGET. vercel.json is
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

export function applyDomains(config, { apiOrigins, authProxyTarget }) {
  for (const origin of [...apiOrigins, authProxyTarget]) {
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
  return next;
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const file = resolve(import.meta.dirname, '../vercel.json');
  const current = readFileSync(file, 'utf8');
  const updated = `${JSON.stringify(applyDomains(JSON.parse(current), {
    apiOrigins: apiOriginsFromEnv(process.env),
    authProxyTarget: process.env.AUTH_PROXY_TARGET,
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
}
