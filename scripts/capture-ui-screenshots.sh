#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Capture ginsights UI screenshots with Playwright.

Usage:
  scripts/capture-ui-screenshots.sh [repo] [--out DIR] [--port PORT] [--update-readme]

Defaults:
  repo: .
  --out: qa/screenshots
  --port: 43118

Environment:
  NODE_BIN                 Node.js executable. Defaults to `node`.
  PLAYWRIGHT_NODE_MODULES  Optional node_modules directory containing Playwright.
  GINSIGHTS_QA_ARGS        Extra args passed to `ginsights serve`.
  GOCACHE                  Go build cache path.

Requirements:
  go
  curl
  Node.js
  Playwright available locally or via PLAYWRIGHT_NODE_MODULES

The script starts `ginsights serve`, captures every report section at desktop
1920x1080 and mobile 390x844, then shuts the server down.
USAGE
}

repo="."
out_dir=""
port="${GINSIGHTS_QA_PORT:-43118}"
update_readme=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --help|-h)
      usage
      exit 0
      ;;
    --out)
      if [ "$#" -lt 2 ]; then
        echo "capture-ui-screenshots.sh: --out requires a value" >&2
        exit 2
      fi
      out_dir="$2"
      shift 2
      ;;
    --port)
      if [ "$#" -lt 2 ]; then
        echo "capture-ui-screenshots.sh: --port requires a value" >&2
        exit 2
      fi
      port="$2"
      shift 2
      ;;
    --update-readme)
      update_readme=1
      shift
      ;;
    -*)
      echo "capture-ui-screenshots.sh: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
    *)
      repo="$1"
      shift
      ;;
  esac
done

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "capture-ui-screenshots.sh: required command not found: $1" >&2
    exit 1
  fi
}

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [ "$out_dir" = "" ]; then
  out_dir="$root/qa/screenshots"
elif [ "${out_dir#/}" = "$out_dir" ]; then
  out_dir="$root/$out_dir"
fi

node_bin="${NODE_BIN:-node}"
require_cmd go
require_cmd curl
if ! command -v "$node_bin" >/dev/null 2>&1 && [ ! -x "$node_bin" ]; then
  echo "capture-ui-screenshots.sh: Node.js executable not found: $node_bin" >&2
  exit 1
fi

mkdir -p "$out_dir"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/ginsights-ui-qa.XXXXXX")"
server_log="$tmp_dir/server.log"
server_pid=""

cleanup() {
  if [ "$server_pid" != "" ] && kill -0 "$server_pid" >/dev/null 2>&1; then
    kill "$server_pid" >/dev/null 2>&1 || true
    wait "$server_pid" >/dev/null 2>&1 || true
  fi
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

cd "$root"

export GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/ginsights-go-cache}"

# shellcheck disable=SC2086
go run ./cmd/ginsights serve "$repo" --port "$port" ${GINSIGHTS_QA_ARGS:-} >"$server_log" 2>&1 &
server_pid="$!"

base_url="http://127.0.0.1:$port"
for _ in $(seq 1 80); do
  if curl -fsS "$base_url/healthz" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$server_pid" >/dev/null 2>&1; then
    cat "$server_log" >&2
    exit 1
  fi
  sleep 0.1
done

if ! curl -fsS "$base_url/healthz" >/dev/null 2>&1; then
  echo "capture-ui-screenshots.sh: server did not become ready at $base_url" >&2
  cat "$server_log" >&2
  exit 1
fi

args=(
  "$root/scripts/capture-ui-screenshots.mjs"
  --base-url "$base_url"
  --out "$out_dir"
)

if [ "$update_readme" -eq 1 ]; then
  args+=(--readme-asset "$root/docs/assets/ginsights-dashboard.jpg")
fi

"$node_bin" "${args[@]}"
