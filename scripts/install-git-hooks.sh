#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOOK_DIR="$ROOT_DIR/.git/hooks"

if [[ ! -d "$HOOK_DIR" ]]; then
  echo ".git/hooks not found" >&2
  exit 1
fi

install -m 0755 "$ROOT_DIR/githooks/pre-commit" "$HOOK_DIR/pre-commit"
install -m 0755 "$ROOT_DIR/githooks/pre-push" "$HOOK_DIR/pre-push"

echo "Installed pre-commit and pre-push hooks."
