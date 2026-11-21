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

for name in LEGACY_FRONTEND_ORIGIN LEGACY_API_ORIGIN NEW_FRONTEND_ORIGIN NEW_API_ORIGIN; do
  if ! [[ ${!name} =~ ^https://[a-z0-9.-]+$ ]]; then
    echo "domain-env: $name must be a bare https origin, got '${!name}'" >&2
    exit 1
  fi
done

# Backend: both frontends may call the API and run the session and passkey
# ceremonies; a reset link with no allowed Origin points at the legacy one.
: "${CORS_ALLOWED_ORIGINS:=$LEGACY_FRONTEND_ORIGIN,$NEW_FRONTEND_ORIGIN}"
: "${FRONTEND_BASE_URL:=$LEGACY_FRONTEND_ORIGIN}"
# Frontend: each domain calls its own API host; anything else the legacy one.
: "${VUE_APP_API_BASE_URL:=$LEGACY_API_ORIGIN}"
: "${VUE_APP_API_BASE_URL_BY_HOST:=$(host_of "$NEW_FRONTEND_ORIGIN")=$NEW_API_ORIGIN,$(host_of "$LEGACY_FRONTEND_ORIGIN")=$LEGACY_API_ORIGIN}"
# Vercel's same-origin /api/auth proxy, server to server.
: "${AUTH_PROXY_TARGET:=$LEGACY_API_ORIGIN}"

for name in LEGACY_FRONTEND_ORIGIN LEGACY_API_ORIGIN NEW_FRONTEND_ORIGIN NEW_API_ORIGIN \
  CORS_ALLOWED_ORIGINS FRONTEND_BASE_URL VUE_APP_API_BASE_URL VUE_APP_API_BASE_URL_BY_HOST AUTH_PROXY_TARGET; do
  printf '%s=%s\n' "$name" "${!name}"
done
