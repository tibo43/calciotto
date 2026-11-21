#!/usr/bin/env bash
# Prints the deploy-time domain settings as NAME=value lines (deploy.yml
# appends them to $GITHUB_ENV).
#
# Defaults come from devops/env/domains.env, the one place the domains are
# written down. Any variable already set and non-empty in the environment
# wins (deploy.yml passes the GitHub repository variables of the same name),
# so each setting can be overridden on its own without a code change.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
defaults="$here/../env/domains.env"

value_of() { # NAME: the environment's value, else the defaults file's
  local name=$1 current=${!1:-}
  if [ -n "$current" ]; then
    printf '%s' "$current"
  else
    sed -n "s/^$name=//p" "$defaults" | tail -n 1
  fi
}
host_of() { local origin=$1; origin=${origin#*://}; printf '%s' "${origin%%[:/]*}"; }

LEGACY_FRONTEND_ORIGIN=$(value_of LEGACY_FRONTEND_ORIGIN)
LEGACY_API_ORIGIN=$(value_of LEGACY_API_ORIGIN)
NEW_FRONTEND_ORIGIN=$(value_of NEW_FRONTEND_ORIGIN)
NEW_API_ORIGIN=$(value_of NEW_API_ORIGIN)
LEGACY_FRONTEND_REDIRECT=$(value_of LEGACY_FRONTEND_REDIRECT)

for name in LEGACY_FRONTEND_ORIGIN LEGACY_API_ORIGIN NEW_FRONTEND_ORIGIN NEW_API_ORIGIN; do
  if ! [[ ${!name} =~ ^https://[a-z0-9.-]+$ ]]; then
    echo "domain-env: $name must be a bare https origin, got '${!name}'" >&2
    exit 1
  fi
done
if [ "$LEGACY_FRONTEND_REDIRECT" != true ] && [ "$LEGACY_FRONTEND_REDIRECT" != false ]; then
  echo "domain-env: LEGACY_FRONTEND_REDIRECT must be true or false, got '$LEGACY_FRONTEND_REDIRECT'" >&2
  exit 1
fi

new_host_api="$(host_of "$NEW_FRONTEND_ORIGIN")=$NEW_API_ORIGIN"
if [ "$LEGACY_FRONTEND_REDIRECT" = true ]; then
  # agoran.app everywhere: the legacy frontend only redirects, so it never
  # runs the app, calls the API or holds a session.
  : "${CORS_ALLOWED_ORIGINS:=$NEW_FRONTEND_ORIGIN}"
  : "${VUE_APP_API_BASE_URL_BY_HOST:=$new_host_api}"
else
  # Both frontends serve the app side by side (the way back): each may call
  # the API and run the session and passkey ceremonies, each with its own
  # API host.
  : "${CORS_ALLOWED_ORIGINS:=$LEGACY_FRONTEND_ORIGIN,$NEW_FRONTEND_ORIGIN}"
  : "${VUE_APP_API_BASE_URL_BY_HOST:=$new_host_api,$(host_of "$LEGACY_FRONTEND_ORIGIN")=$LEGACY_API_ORIGIN}"
fi
# A reset link with no allowed Origin, and any host the map doesn't list,
# use the new pair.
: "${FRONTEND_BASE_URL:=$NEW_FRONTEND_ORIGIN}"
: "${VUE_APP_API_BASE_URL:=$NEW_API_ORIGIN}"
# Vercel's same-origin /api/auth proxy, server to server.
: "${AUTH_PROXY_TARGET:=$NEW_API_ORIGIN}"

for name in LEGACY_FRONTEND_ORIGIN LEGACY_API_ORIGIN NEW_FRONTEND_ORIGIN NEW_API_ORIGIN LEGACY_FRONTEND_REDIRECT \
  CORS_ALLOWED_ORIGINS FRONTEND_BASE_URL VUE_APP_API_BASE_URL VUE_APP_API_BASE_URL_BY_HOST AUTH_PROXY_TARGET; do
  printf '%s=%s\n' "$name" "${!name}"
done
