#!/usr/bin/env bash
set -euo pipefail

if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Already inside a Git work tree."
  exit 0
fi

git init -b main
git add .
git -c user.name="ginsights starter" -c user.email="starter@example.com" commit -m "Initial ginsights starter"
echo "Git repository initialized."
