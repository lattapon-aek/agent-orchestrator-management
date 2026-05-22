#!/usr/bin/env bash
# update.sh — pull the latest source changes and rebuild/install aom
#
# Usage:
#   ./scripts/update.sh
#   ./scripts/update.sh --test

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

if command -v git &>/dev/null && git -C "$PROJECT_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    if git -C "$PROJECT_ROOT" rev-parse --abbrev-ref --symbolic-full-name @{u} >/dev/null 2>&1; then
        git -C "$PROJECT_ROOT" pull --ff-only
    else
        printf '[aom-update] no upstream configured; rebuilding current checkout\n'
    fi
fi

exec "$PROJECT_ROOT/scripts/install.sh" "$@"
