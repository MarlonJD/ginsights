#!/usr/bin/env bash
set -euo pipefail

repo_url="${GINSIGHTS_REPO_URL:-https://github.com/MarlonJD/ginsights.git}"
ref="${GINSIGHTS_REF:-main}"
install_dir="${GINSIGHTS_INSTALL_DIR:-$HOME/.local/bin}"
dry_run=0

usage() {
  cat <<'USAGE'
Install ginsights from source.

Usage:
  install.sh [--install-dir DIR] [--ref REF] [--repo URL] [--dry-run]

Defaults:
  --repo https://github.com/MarlonJD/ginsights.git
  --ref main
  --install-dir $HOME/.local/bin

Environment overrides:
  GINSIGHTS_REPO_URL
  GINSIGHTS_REF
  GINSIGHTS_INSTALL_DIR

Requirements:
  git
  go
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --help|-h)
      usage
      exit 0
      ;;
    --install-dir)
      if [ "$#" -lt 2 ]; then
        echo "install.sh: --install-dir requires a value" >&2
        exit 2
      fi
      install_dir="$2"
      shift 2
      ;;
    --ref)
      if [ "$#" -lt 2 ]; then
        echo "install.sh: --ref requires a value" >&2
        exit 2
      fi
      ref="$2"
      shift 2
      ;;
    --repo)
      if [ "$#" -lt 2 ]; then
        echo "install.sh: --repo requires a value" >&2
        exit 2
      fi
      repo_url="$2"
      shift 2
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    *)
      echo "install.sh: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "install.sh: required command not found: $1" >&2
    exit 1
  fi
}

run() {
  printf '+ %s\n' "$*"
  if [ "$dry_run" -eq 0 ]; then
    "$@"
  fi
}

require_cmd git
require_cmd go

tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/ginsights-install.XXXXXX")"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

echo "Installing ginsights"
echo "  repo:        $repo_url"
echo "  ref:         $ref"
echo "  install dir: $install_dir"

if [ "$dry_run" -eq 1 ]; then
  run git clone --depth 1 --branch "$ref" "$repo_url" "$tmp_dir/src"
elif git ls-remote --exit-code --heads "$repo_url" "$ref" >/dev/null 2>&1 || git ls-remote --exit-code --tags "$repo_url" "$ref" >/dev/null 2>&1; then
  run git clone --depth 1 --branch "$ref" "$repo_url" "$tmp_dir/src"
else
  run git clone "$repo_url" "$tmp_dir/src"
  run git -C "$tmp_dir/src" checkout "$ref"
fi

run mkdir -p "$install_dir"
run go build -trimpath -ldflags="-s -w" -o "$install_dir/ginsights" "$tmp_dir/src/cmd/ginsights"

if [ "$dry_run" -eq 1 ]; then
  echo "Dry run complete; no files installed."
else
  echo "Installed: $install_dir/ginsights"
  echo "Try: $install_dir/ginsights help"
fi
