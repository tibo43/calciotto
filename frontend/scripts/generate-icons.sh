#!/usr/bin/env sh
# Regenerates public/icons/ from one source image, inside the same pinned
# Playwright image the visual tests use (see tests/visual/run-in-docker.sh for
# why that image, and the node_modules volume, rather than the host).
#
# Usage (from frontend/):
#   sh scripts/generate-icons.sh public/favicon.svg --inset 0.8
#   sh scripts/generate-icons.sh path/to/logo.png              # full-bleed square logo
set -e

IMAGE=mcr.microsoft.com/playwright:v1.62.1-noble
FRONTEND=$(cd "$(dirname "$0")/.." && pwd)

exec docker run --rm -t \
  -v "$FRONTEND":/work \
  -v calciotto-visual-node-modules:/work/node_modules \
  -w /work \
  --user "$(id -u):$(id -g)" -e HOME=/tmp \
  "$IMAGE" sh -c '
    set -e
    if [ ! -f node_modules/.visual-install-stamp ] || [ package-lock.json -nt node_modules/.visual-install-stamp ]; then
      echo "node_modules volume is not installed yet: run npm run test:visual once first." >&2
      exit 1
    fi
    node scripts/generate-icons.js "$@"
  ' sh "$@"
