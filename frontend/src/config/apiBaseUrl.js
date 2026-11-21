// Which API the bundle calls. One build is served on every frontend domain
// (the legacy one and the new one, side by side), so the choice is made at
// run time from the page's own host:
//
//   VUE_APP_API_BASE_URL_BY_HOST  "club.agoran.app=https://api.agoran.app,…"
//   VUE_APP_API_BASE_URL          any other host (and local dev)
//
// Both are build-time variables set by deploy.yml from devops/env/domains.env.
export const LOCAL_API_BASE_URL = 'http://127.0.0.1:8080';

// "host=origin" pairs, comma-separated. A malformed pair is ignored rather
// than breaking the whole bundle; hosts compare case-insensitively.
export function parseApiBaseUrlByHost(raw) {
  const byHost = new Map();
  for (const pair of String(raw || '').split(',')) {
    const separator = pair.indexOf('=');
    if (separator < 0) continue;
    const host = pair.slice(0, separator).trim().toLowerCase();
    const origin = pair.slice(separator + 1).trim().replace(/\/+$/, '');
    if (host && /^https?:\/\/[^/\s]+$/.test(origin)) byHost.set(host, origin);
  }
  return byHost;
}

export function resolveApiBaseUrl(hostname, byHostRaw, fallback) {
  const byHost = parseApiBaseUrlByHost(byHostRaw);
  return byHost.get(String(hostname || '').toLowerCase()) || fallback || LOCAL_API_BASE_URL;
}
