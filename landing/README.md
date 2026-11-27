# Agoran landing page

Static one-page site for `agoran.app` (apex). No build step, no JavaScript, no service worker, no web app manifest — the PWA install stays on `club.agoran.app`.

- Deployed by a **separate Vercel project** whose **Root Directory is `landing/`** (framework preset: Other, no build command, no output directory).
- Independent from `frontend/` and `app/`: fonts, logo and screenshots are copied into `assets/`, nothing is referenced across folders.
- Light/dark follows `prefers-color-scheme` only (no toggle, no storage). Logo and app screenshots switch with the theme via `<picture>`.
- Every « Open the app » button points to https://club.agoran.app.
- Preview locally: `python3 -m http.server -d landing 8000`.
