# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

Calciotto is a small full-stack app for organizing amateur football ("calciotto") matches: creating matches, assigning players to color-named teams, and tracking goals per player. Backend is Go (Gin + GORM + PostgreSQL), frontend is Vue 3 (Vue CLI).

## Commands

### Dev environment (all services)
```bash
cd devops
docker compose --env-file env/develop.env up
```
This starts Postgres (with healthcheck), the Go backend (via `air` for live reload, port 8080), and the Vue frontend (`npm install && npm run serve`, port 4000). The `--env-file` flag is required — without it, `${POSTGRES_PORT}`/`${POSTGRES_PASSWORD}` and the bare `DB_NAME`/`DB_USER`/`DB_PASSWORD` refs in `docker-compose.yaml` resolve to empty strings (Compose only auto-loads a file literally named `.env` next to the compose file, which this repo intentionally doesn't ship since `.env` is gitignored). The backend and frontend containers bind-mount `../app` and `../frontend`, so local edits are picked up without rebuilding the image.

### Backend (`app/`)
```bash
go build ./...                 # build
go run .                       # run directly (needs Postgres reachable — set DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME, see Gotchas)
go run ./cmd/seed              # populate the DB with demo players/teams/matches (skips if players already exist)
go run ./cmd/seed -reset       # wipe players/teams/matches/match_players first, then reseed
go run ./cmd/seed -matches 10  # control how many past matches get generated (default 6)
```
There is no separate lint command configured for the Go module, and no test suite currently exists.

### Frontend (`frontend/`)
```bash
npm run serve   # dev server on port 4000
npm run build   # production build
npm run lint    # eslint (plugin:vue/vue3-essential + eslint:recommended)
```

## Architecture

### Backend layering
Strict `handlers/` → `services/` → `models/` layering:
- `internal/handlers/*.go` — Gin controllers only: bind JSON, call the service, map errors to HTTP status/JSON. No business logic here.
- `internal/services/*.go` — one service struct per entity (`PlayerService`, `TeamService`, `MatchService`, `MatchPlayerService`), each wrapping a `*gorm.DB`.
- `internal/models/database.go` — GORM models (`Player`, `Team`, `Match`, `MatchPlayer`), all embedding `BaseModel` which generates a `uuid.UUID` ID (`github.com/google/uuid`) in a `BeforeCreate` hook. All FK-style fields (`MatchPlayer.MatchID/TeamID/PlayerID`) are `uuid.UUID`, not `string` — comparisons against "no value" use `uuid.Nil`, not `""`.
- `internal/models/date.go` — `Date` is a `time.Time`-backed type used everywhere a calendar day is stored (`Match.Date`, DTO date fields). It marshals/unmarshals as `"YYYY-MM-DD"` over JSON (matching the original string-based API contract exactly) and scans/values as a SQL `date` column — this is why `Match.Date` could switch from `string` to a typed date without changing the wire format or requiring frontend changes.
- `internal/models/customModels.go` — DTOs used only for the "match with nested teams/players" API shape (`MatchWithDetails`, `TeamWithPlayers`, `PlayerCustom`) plus a flat row struct (`RowsMatchDetails`) used to scan the raw SQL join in `matches.go`. IDs are `uuid.UUID`, dates are `Date`.

Only `Player` and `Match` routes are actually wired in `main.go`. `TeamHandler` and `MatchPlayerHandler` exist and are fully implemented in `internal/handlers/` and `internal/services/` but are **not registered as routes** — there is currently no HTTP path to create a `Team` or a standalone `MatchPlayer` row directly.

Every handler that reads an ID from the URL (`c.Param("id")`, etc.) parses it with `uuid.Parse` and returns 400 on failure before calling the service — service methods take `uuid.UUID`, never a raw path string.

### Data model
`MatchPlayer` is the join table that carries both roster membership *and* the score: `(match_id, team_id, player_id, goals_scored)`. There's no separate "goals" or "events" table — a team's score is always derived by summing `goals_scored` over its players.

### The flatten/reconstruct pattern (matches.go)
`MatchService.GetMatchesDetails` / `GetMatchDetailsByID` do not use GORM associations or `Preload`. They run a raw SQL query with `LEFT JOIN`s across `matches → match_players → teams/players`, scan into flat `RowsMatchDetails` rows, then manually rebuild the nested `MatchWithDetails{ Teams: []TeamWithPlayers{ Players: []PlayerCustom } }` structure in Go (matching by ID in loops, appending as new IDs are encountered). When a match has no rows in `match_players` for a given team (or at all), there's a fallback path that queries all `teams` directly to still return empty team shells. Any change to the JSON shape returned to the frontend needs to touch both the SQL query and this reconstruction loop, not just the model struct.

`MatchService.UpdateMatch` performs a diff: for each team in the incoming payload it loads existing `MatchPlayer` rows for that `(match_id, team_id)`, creates rows for players not yet present, updates `goals_scored` for players still present, and deletes rows for players removed from the payload. The whole diff runs inside `s.DB.Transaction(...)`.

### Input validation
`PlayerService.CreatePlayer` trims the name, rejects empty input (`ErrEmptyPlayerName`) and rejects case-insensitive duplicates (`ErrPlayerAlreadyExists`) — both are sentinel errors in `internal/services/players.go`, mapped to HTTP 400 in `PlayerHandler.CreatePlayer` via `errors.Is`. `Player.Name` also has a DB-level `uniqueIndex` as a second line of defense. Match date validation happens for free via the `Date` type's `UnmarshalJSON`: a malformed date fails JSON binding in the handler and surfaces as a 400 through the existing `ShouldBindJSON` error path, with no extra service-side check needed.

### Frontend
- `src/router/index.js` defines two routes: `/` (`MatchesAll`) and `/matches/:id/edit` (`MatchDetails`). `App.vue` also has non-router tabs ("Standings"/teams/players) that render static "Coming soon" placeholders and are unrelated to the router views.
- `src/services/api.js` is the single Axios client (`API_BASE_URL` from `VUE_API_BASE_URL` env var, default `http://127.0.0.1:8080`) — every backend call goes through here.
- `MatchesAll.vue` lists matches and lets you create one (custom date-picker, no external date library); `MatchDetails.vue` is where teams are populated (player search/create-on-the-fly) and goals are edited, then persisted via a single `updateMatch` call that sends the whole match payload.
- API payload field casing is PascalCase (`ID`, `Name`, `Colour`, `Players`, `GoalNumber`/`GoalsScored` depending on endpoint) — this comes directly from the Go JSON struct tags in `customModels.go`, not a frontend convention.

### Seeding data
`app/cmd/seed/main.go` is the real seeding tool (replaces an older set of ad hoc `*_test.go` files that abused `go test` to insert rows with no isolation or cleanup — those are gone). It reuses the normal `internal/services` layer, so seeded data goes through the same code path as the API. It creates 12 players, 2 teams (`black`/`white`), and N matches on the last N Sundays, each match splitting the players randomly between the two teams with a weighted-random goal count per player. Run via `go run ./cmd/seed` (locally, with `DB_HOST=localhost` etc. exported) or `docker compose exec backend go run ./cmd/seed` (inside the network, where the `db` hostname resolves).

## Gotchas

- `AutoMigrate` runs on every backend startup (`InitDB` in `connect.go`), so schema changes to the GORM models in `internal/models/database.go` take effect automatically on next run/restart — no separate migration files exist.
- Outside Docker (e.g. running `go run .` or `go run ./cmd/seed` directly on the host), `DB_HOST` defaults to `db` which won't resolve — override it to `localhost` and match the port Postgres is published on (`POSTGRES_PORT`/`DB_PORT` in `devops/env/develop.env`, 5432 by default).
- Forgetting `--env-file env/develop.env` on `docker compose up` silently breaks the `db` service: `${POSTGRES_PORT}`/`${POSTGRES_PASSWORD}` and the bare `DB_NAME`/`DB_USER`/`DB_PASSWORD` refs resolve to empty, and the `pg_isready` healthcheck then fails with `role "root" does not exist` (it falls back to the container's default OS user). Adding `environment:`/`env_file:` on the `db` service itself does not fix this — those only populate the container's runtime env *after* Compose has already resolved the `${...}` placeholders in the YAML from the flag/shell env.
