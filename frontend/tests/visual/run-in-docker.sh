#!/usr/bin/env sh
# Runs the visual tests inside the pinned Playwright image — the only supported
# way to run them from a developer machine.
#
# Not a convenience wrapper: pixel baselines are only comparable when they are
# produced by the same font stack and the same browser build every time. Running
# `npx playwright test` on macOS would compare Mac-rendered text against
# Linux-rendered baselines and fail on every screenshot, and "fixing" that with a
# loose pixel threshold would mean the tests no longer catch anything.
#
# The image tag must match the @playwright/test version in package.json: the
# image ships the browser build its own version expects.
#
# Usage:
#   sh tests/visual/run-in-docker.sh                     # verify against baselines
#   sh tests/visual/run-in-docker.sh --update-snapshots  # accept the current rendering
#   sh tests/visual/run-in-docker.sh home.spec.js        # one file
set -e

IMAGE=mcr.microsoft.com/playwright:v1.62.1-noble
FRONTEND=$(cd "$(dirname "$0")/../.." && pwd)

if ! docker info >/dev/null 2>&1; then
  echo "docker is not available — start Docker Desktop, or run the 'visual' job in CI." >&2
  exit 1
fi

# node_modules lives in a named volume rather than in the bind mount: the host's
# own node_modules is built for macOS, and letting the container's `npm ci`
# overwrite it would break local development until the next reinstall. The volume
# also survives between runs, so the install happens once rather than every time.
#
# --ipc=host is Playwright's own recommendation for Chromium in Docker: the
# default 64MB /dev/shm makes the browser crash on large pages, which here would
# look like a flaky screenshot rather than what it is.
exec docker run --rm -t \
  --ipc=host \
  -v "$FRONTEND":/work \
  -v calciotto-visual-node-modules:/work/node_modules \
  -w /work \
  -e CI=1 \
  "$IMAGE" sh -c '
    set -e
    if [ ! -f node_modules/.visual-install-stamp ] || [ package-lock.json -nt node_modules/.visual-install-stamp ]; then
      npm ci --no-audit --no-fund
      touch node_modules/.visual-install-stamp
    fi
    npm run build
    npx playwright test "$@"
  ' sh "$@"
