#!/usr/bin/env bash
# SessionStart: point git at the versioned hooks dir so the commit-msg style gate
# is active in every checkout (idempotent; silent on success).
set -euo pipefail
root="${CLAUDE_PROJECT_DIR:-$PWD}"
git -C "$root" config core.hooksPath .githooks
exit 0
