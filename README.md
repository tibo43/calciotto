# Calciotto App

## Running development env

``` bash
cd devops
docker-compose --env-file env/develop.env up
```

## Browser sessions and group invitations

The frontend sends authentication requests to `/api/auth` on its own origin.
Vercel forwards these requests to the primary Koyeb API. Vue's development
server forwards them to the local API using `DEV_API_PROXY_TARGET`. Include
the exact frontend origin in the API's `CORS_ALLOWED_ORIGINS` and run the
deployed API with `GIN_MODE=release` for secure cookies.

New browser sessions last seven days from login or signup. Their access
tokens last up to 15 minutes and the client renews them on demand with a
rotating HttpOnly cookie. Successful renewal and temporary errors preserve
push subscriptions. The client uses its existing device cleanup when
renewal confirms the session has ended or the player logs out. Access
tokens issued before the change remain valid until their original expiry.

Group administrators can generate and revoke invitations in group settings.
Each new code lasts seven days. Existing codes get seven days from migration.

For API clients, `GET /matches/details` returns up to 100 matches per
response. Follow `X-Next-Cursor` with the `cursor` query parameter and
keep the same group and season. The frontend follows these pages automatically.
