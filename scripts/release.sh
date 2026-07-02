#!/usr/bin/env bash
# release.sh — bump guten (go + cli + js) to X.Y.Z in lockstep, commit, and push
# the module tags that trigger CI: js/vX -> npm publish, cli/vX -> binaries;
# go/vX makes the Go module resolvable. Mirrors kitsy/cnos's release.sh.
#
# Usage:
#   scripts/release.sh patch              # 3.4.6 -> 3.4.7
#   scripts/release.sh minor              # 3.4.6 -> 3.5.0
#   scripts/release.sh major              # 3.4.6 -> 4.0.0
#   scripts/release.sh 0.3.0              # explicit version
#   scripts/release.sh --skip-tests patch
#   scripts/release.sh --no-tag 0.3.0     # bump + push main + go tag only
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; cd "$ROOT"
die()  { echo "ERROR: $*" >&2; exit 1; }
info() { echo "  → $*"; }
step() { echo; echo "▸ $*"; }

SKIP_TESTS=false; NO_TAG=false; NEW=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-tests) SKIP_TESTS=true; shift ;;
    --no-tag)     NO_TAG=true;     shift ;;
    -h|--help)    echo "usage: release.sh [--skip-tests] [--no-tag] <major|minor|patch|X.Y.Z>"; exit 0 ;;
    -*)           die "unknown option: $1" ;;
    *)            [[ -z "$NEW" ]] || die "unexpected argument: $1"; NEW="$1"; shift ;;
  esac
done
[[ -n "${NEW:-}" ]] || die "expected major|minor|patch or X.Y.Z"

OLD=$(sed -n 's/.*"version": "\([^"]*\)".*/\1/p' js/package.json | head -1)
[[ -n "$OLD" ]] || die "cannot read version from js/package.json"

# Semantic bump: major|minor|patch auto-increments from the current version
# (e.g. 3.4.6 --> major=4.0.0, minor=3.5.0, patch=3.4.7). Or pass an explicit X.Y.Z.
case "$NEW" in
  major|minor|patch)
    IFS=. read -r MA MI PA <<< "$OLD"
    [[ "$MA" =~ ^[0-9]+$ && "$MI" =~ ^[0-9]+$ && "$PA" =~ ^[0-9]+$ ]] || die "cannot parse current version '$OLD'"
    case "$NEW" in
      major) MA=$((MA + 1)); MI=0; PA=0 ;;
      minor) MI=$((MI + 1)); PA=0 ;;
      patch) PA=$((PA + 1)) ;;
    esac
    NEW="${MA}.${MI}.${PA}"
    ;;
  *)
    [[ "$NEW" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "expected major|minor|patch or X.Y.Z (got '$NEW')"
    ;;
esac
[[ "$OLD" != "$NEW" ]] || die "already at $NEW"
echo; echo "Release: $OLD -> $NEW"

step "Pre-flight"
[[ "$(git branch --show-current)" == "main" ]] || die "must be on main"
[[ -z "$(git status --porcelain)" ]] || die "working tree dirty — commit/stash first"
git fetch --quiet origin main
[[ "$(git rev-parse HEAD)" == "$(git rev-parse origin/main)" ]] || die "behind origin/main — git pull first"
info "on main, clean, up to date ✓"

for t in "go/v${NEW}" "cli/v${NEW}" "js/v${NEW}"; do
  if git rev-parse -q --verify "refs/tags/${t}" >/dev/null 2>&1 || git ls-remote --tags origin "${t}" | grep -q "refs/tags/${t}"; then
    die "tag ${t} already exists — bump to a new version"
  fi
done
info "tags go/cli/js v${NEW} are free ✓"

step "Tests"
if [[ "$SKIP_TESTS" == false ]]; then
  ( cd go && go test ./... >/dev/null ) && info "go ✓"
  ( cd cli && go test ./... >/dev/null ) && info "cli ✓"
  ( cd js && ./node_modules/.bin/vitest run >/dev/null ) && info "js ✓"
else
  info "skipped (--skip-tests)"
fi

step "Bump versions (-> $NEW)"
# Value-agnostic: replace the current version whatever it is, so go/cli/js need
# not already be in sync. (js: first "version" field only; cli: the unique var.)
sed -i "0,/\"version\": \"[^\"]*\"/s//\"version\": \"${NEW}\"/" js/package.json
sed -i "s/var version = \"[^\"]*\"/var version = \"${NEW}\"/" cli/cmd/guten/main.go
info "js/package.json + cli/cmd/guten/main.go ✓"

step "Commit + push version bump"
git add js/package.json cli/cmd/guten/main.go
git commit -m "chore(release): guten v${NEW}

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
git push origin main

step "Tag + push go/v${NEW} (Go module)"
git tag "go/v${NEW}"; git push origin "go/v${NEW}"

step "Point cli at guten/go@v${NEW}"
( cd cli && GOFLAGS=-mod=mod GOPROXY=direct GOPRIVATE='github.com/kitsyai/*' \
    go get "github.com/kitsyai/guten/go@v${NEW}" && go mod tidy )
if [[ -n "$(git status --porcelain cli/go.mod cli/go.sum)" ]]; then
  git add cli/go.mod cli/go.sum
  git commit -m "chore(cli): guten/go@v${NEW}

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
  git push origin main
  info "cli deps updated ✓"
fi

if [[ "$NO_TAG" == true ]]; then
  echo; echo "Bumped + pushed main + go/v${NEW}. Finish with:"
  echo "  git tag cli/v${NEW} js/v${NEW} && git push origin cli/v${NEW} js/v${NEW}"
  exit 0
fi

step "Tag + push cli/v${NEW} and js/v${NEW} (triggers CI)"
git tag "cli/v${NEW}"; git tag "js/v${NEW}"
git push origin "cli/v${NEW}" "js/v${NEW}"
echo; echo "Done. CI publishes @kitsy/guten to npm (js/v${NEW}) and cli binaries (cli/v${NEW})."
