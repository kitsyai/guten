#!/usr/bin/env bash
# Build the guten CLI binaries with goreleaser (OSS) and publish them to the
# GitHub Release for the prefixed cli/<version> tag via gh.
#
# Usage: scripts/release-cli.sh v0.2.0        (run from the repo root)
# Needs: goreleaser (OSS) + gh (authenticated, or GH_TOKEN set).
set -euo pipefail

VERSION="${1:?usage: scripts/release-cli.sh <vX.Y.Z>}"
TAG="cli/${VERSION}"

# OSS goreleaser can't parse the cli/ tag prefix; feed it the plain version and
# skip validate (the working tag is prefixed) and publish (gh does that).
GORELEASER_CURRENT_TAG="${VERSION}" goreleaser release --clean --skip=validate,publish

gh release create "${TAG}" \
    dist/guten_*.tar.gz dist/guten_*.zip dist/checksums.txt \
    --title "guten CLI ${VERSION}" \
    --notes "Prebuilt guten CLI ${VERSION}. Download your platform archive, or \`go install github.com/kitsyai/guten/cli/cmd/guten@${TAG}\`." \
  || gh release upload "${TAG}" dist/guten_*.tar.gz dist/guten_*.zip dist/checksums.txt --clobber
