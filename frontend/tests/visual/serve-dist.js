// Static server for the production build, with the SPA fallback Vercel does in
// production (frontend/vercel.json's `/(.*)` → /index.html rewrite).
//
// The visual tests deep-link into /matches/:id/edit, and the router is
// createWebHistory, so without the fallback that URL 404s and every screenshot
// captures an error page instead of the app. It is a rewrite rather than a
// redirect, and only after the filesystem check, exactly like the production
// rule — otherwise the fallback would swallow a genuinely missing asset and the
// page would render half-styled instead of failing visibly.
//
// Written by hand rather than pulling in a static-server dependency: it needs
// ~40 lines, and the alternative is another package in the tree for the sake of
// a test fixture.

const http = require('http');
const fs = require('fs');
const path = require('path');

const DIST = path.join(__dirname, '..', '..', 'dist');
const PORT = Number(process.env.VISUAL_SERVER_PORT || 4173);

const CONTENT_TYPES = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.ico': 'image/x-icon',
  '.woff': 'font/woff',
  '.woff2': 'font/woff2',
  '.map': 'application/json; charset=utf-8',
};

const send = (res, status, body, type) => {
  res.writeHead(status, { 'Content-Type': type, 'Cache-Control': 'no-store' });
  res.end(body);
};

const server = http.createServer((req, res) => {
  const urlPath = decodeURIComponent(new URL(req.url, 'http://localhost').pathname);
  // Resolve inside DIST and check the result still starts there, so a
  // `../`-laden request can't read outside the build directory.
  const candidate = path.resolve(DIST, `.${urlPath}`);
  if (!candidate.startsWith(DIST)) {
    send(res, 403, 'Forbidden', 'text/plain; charset=utf-8');
    return;
  }

  const file = fs.existsSync(candidate) && fs.statSync(candidate).isFile() ? candidate : null;
  if (file) {
    send(res, 200, fs.readFileSync(file), CONTENT_TYPES[path.extname(file)] || 'application/octet-stream');
    return;
  }

  // An asset that isn't there is a 404, not the SPA shell: serving index.html
  // for a missing chunk would make a broken build look like a styling bug.
  if (path.extname(urlPath)) {
    send(res, 404, 'Not found', 'text/plain; charset=utf-8');
    return;
  }

  const index = path.join(DIST, 'index.html');
  if (!fs.existsSync(index)) {
    send(res, 500, 'dist/index.html is missing — run `npm run build` first', 'text/plain; charset=utf-8');
    return;
  }
  send(res, 200, fs.readFileSync(index), CONTENT_TYPES['.html']);
});

server.listen(PORT, '127.0.0.1', () => {
  console.log(`serving ${DIST} on http://127.0.0.1:${PORT}`);
});
