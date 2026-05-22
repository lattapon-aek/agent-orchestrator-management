#!/usr/bin/env bash
# uninstall.sh — wrapper for `aom uninstall`
#
# Usage:
#   ./scripts/uninstall.sh
#   ./scripts/uninstall.sh --help

set -euo pipefail

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    echo "Usage: $0"
    echo "Delegates to: aom uninstall"
    exit 0
fi

exec aom uninstall

