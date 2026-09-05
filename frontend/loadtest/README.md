# Perf load test — 50 parallel sign-ups

One-off tooling to run a real perf test against production: 50 accounts
created and driven entirely through the UI (not the API), signing up for one
match, then a manual SQL cleanup. Nothing here is part of `npm run test:unit`
or `test:visual` — it's meant to be run by hand, deliberately, against a real
deployment.

## 0. Temporarily raise the signup rate limit

Production throttles `POST /auth/signup` to ~5/hour per IP (see
`app/main.go`), which 50 parallel sign-ups from one machine will hit almost
immediately. Set these two **secrets** on the `production` GitHub
environment, then push anything to `master` (or re-run the `deploy-backend`
job) to roll them out:

- `SIGNUP_RATE_LIMIT_PER_HOUR` — e.g. `200`
- `SIGNUP_RATE_LIMIT_BURST` — e.g. `60`

Both are optional and only synced to Koyeb when set (see the "Sync secrets to
Koyeb" step in `.github/workflows/ci.yml`) — leaving them unset keeps today's
default (5/hour, burst 5) exactly as it is. **Remove both secrets and
redeploy once the test is done** to revert.

## 1. Create the group, admin and match

From `app/`, pointed at the production database:

```bash
DATABASE_URL="<production DATABASE_URL>" go run ./cmd/perfsetup -frontend-url "<your Vercel URL>"
```

This prints a group id, invite code, admin login, and match id — keep this
output, you need the group id for step 4 and the invite code for step 2.

## 2. Run the load test

From `frontend/`, against the deployed frontend:

```bash
BASE_URL="<your Vercel URL>" INVITE_CODE="<from step 1>" USERS=50 node loadtest/run.js
```

Each of the 50 virtual users, in parallel: signs up via `/signup?invite=...`
(auto-logs in), participates in the match, opens the group's roster on
`/profile` to see the other sign-ups, then logs out. A pass/fail summary with
p50/p95/max durations prints at the end.

## 3. Clean up

Fill in the group id from step 1 into `devops/perf-cleanup.sql`'s
verification queries, review the counts, then uncomment and run the `DELETE`
transaction:

```bash
psql "<production DATABASE_URL>" -v group_id="'<group id from step 1>'" -f ../devops/perf-cleanup.sql
```

This removes the throwaway group, its teams, the match and its
registrations, the group memberships, and every `perfload+uN@perfload.test`
/ `perfload-admin@perfload.test` player. Nothing outside that scope is
touched.

## 4. Revert the rate limit

Delete (or empty) the two GitHub secrets from step 0 and push to `master`
again so the next deploy drops the `--env` overrides, returning to the
hardcoded 5/hour, burst 5 default.
