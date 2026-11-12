#!/usr/bin/env sh
# Regenerates every icon (public/icons/, public/favicon.ico) from one source image, inside the same pinned
# Playwright image the visual tests use (see tests/visual/run-in-docker.sh for
# why that image, and the node_modules volume, rather than the host).
#
# Usage (from frontend/):
#   sh scripts/generate-icons.sh branding/icon-source.png \
#       --light branding/icon-source-light.png \
#       --zoom 1.5 --maskable-zoom 1.22 --background '#1f2937'   # the current logo
#   sh scripts/generate-icons.sh path/to/round-logo.svg --inset 0.8
#
# The current source is a 1024 px square whose ball occupies about 60% of it,
# centred, on navy #1f2937. --zoom 1.5 crops that margin so the ball fills
# about 90% of the any-purpose icons (and of logo-light.png, from the light
# source). --maskable-zoom 1.22 stops at about 73%, inside Android's safe-zone
# circle (80% of the width); the navy still covers the whole maskable square.
# Re-measure both against a new source before reusing these numbers.
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
