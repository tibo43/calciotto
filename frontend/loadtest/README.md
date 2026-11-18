# Perf load test — 50 parallel sign-ups

Run the load test against a local frontend, API, and disposable PostgreSQL
database. Playwright drives the signup, participation, and logout flows
through the browser. The runner accepts loopback addresses and blocks remote
requests.

## 1. Start a disposable database and create fixtures

From the repository root, with Docker, Go and Node installed:

```bash
unset DATABASE_URL DATABASE_URL_UNPOOLED
export DB_HOST=127.0.0.1 DB_PORT=5433 DB_USER=calciotto_perf DB_NAME=calciotto_perf
export DB_PASSWORD="$(openssl rand -hex 24)"
export PERF_ADMIN_PASSWORD="$(openssl rand -hex 24)"
POSTGRES_PASSWORD="$DB_PASSWORD" docker run --rm --detach --name calciotto-perf-db \
  --publish "127.0.0.1:$DB_PORT:5432" \
  --env POSTGRES_PASSWORD --env POSTGRES_USER="$DB_USER" \
  --env POSTGRES_DB="$DB_NAME" postgres:17
docker exec calciotto-perf-db pg_isready -U "$DB_USER" -d "$DB_NAME"
(cd app && go run ./cmd/perfsetup -frontend-url http://127.0.0.1:4000)
```

Run the setup once PostgreSQL is ready. It prints the invite code, admin email,
group ID, and match ID. The admin can sign in with the password chosen for
this run in `PERF_ADMIN_PASSWORD`. To seed an existing local player as a second
admin, pass `-existing-player-id` with their UUID from this test database.

## 2. Start the local app and run the test

Start the API with the same `DB_*` environment. For 50 sign-ups from the same
machine, raise the signup rate and burst for this local process:

```bash
(cd app && JWT_SECRET="$(openssl rand -hex 32)" \
  SIGNUP_RATE_LIMIT_PER_HOUR=200 SIGNUP_RATE_LIMIT_BURST=60 go run .)
```

In another terminal, start the frontend using the loopback API address:

```bash
cd frontend
npm ci
VUE_APP_API_BASE_URL=http://127.0.0.1:8080 npm run serve -- --host 127.0.0.1
```

From `frontend/` in a third terminal:

```bash
npx playwright install chromium
BASE_URL=http://127.0.0.1:4000 INVITE_CODE="<from step 1>" USERS=50 node loadtest/run.js
```

Each of the 50 virtual users, in parallel: signs up via `/signup?invite=...`
(auto-logs in), participates in the match, opens the group's roster on
`/profile` to see the other sign-ups, then logs out. A pass/fail summary with
p50/p95/max durations (overall, and broken down per step: signup, participate,
view_roster, logout) prints at the end **and** is written to
`loadtest/reports/report-<timestamp>.{json,txt}` — the JSON carries the full
per-user breakdown (useful for a spreadsheet or a follow-up script), the
`.txt` is the same summary in a form you can paste straight into a message.
These report files are gitignored — they're per-run artifacts against a
moving target, not something to commit.

If a run reports every user failing at the `signup` step with "no
navigation and no error message shown", **check the failure screenshot in
`loadtest/reports/failures/` before assuming the app is broken** — a common
cause is the signup request simply taking longer than the wait allows under
concurrent load while the backend hashes the passwords. If the screenshot shows
the app already past signup (matches loaded, group selected), that's exactly
this: raise `SIGNUP_TIMEOUT_MS` (default 60000) rather than treating it as a
bug, and treat the actual signup latency this reveals as a real perf-test
finding in its own right.

If instead a failure's error text is the UI's generic fallback ("Signup
failed. Please try again.") rather than a timeout, that error already
resolved quickly — raising a timeout won't change it. Every such failure logs
the HTTP status or network error in `[brackets]` right
after the UI text (e.g. `[http 502]` or `[network error: ...]`) — that's
the backend or connection actually breaking under load, a genuine perf
finding rather than a script issue. `STEP_TIMEOUT_MS` (default 120000)
raises every other wait in the script (page rendering, the roster panel,
logout) together, for a "just give it plenty of room and see what actually
happens" run.

## 3. Clean up

Stop the local frontend and API, then run `docker stop calciotto-perf-db` to
remove the disposable database container. If reusing an isolated database,
`devops/perf-cleanup.sql` provides a dry run and a separate cleanup
transaction for a loopback connection to `calciotto_perf`.
