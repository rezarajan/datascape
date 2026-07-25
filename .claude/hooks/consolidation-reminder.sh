#!/usr/bin/env bash
# SessionStart: warn when the doc-consolidation cadence has lapsed.
# Fix from platformctl (agentic-development §7): additive-only doc growth with no
# consolidation pass produced bloated, self-duplicating docs. Cadence: 28 days.
# Passes are recorded as dated entries in docs/consolidation.md (a Record).
set -euo pipefail

root="${CLAUDE_PROJECT_DIR:-$PWD}"
f="$root/docs/consolidation.md"
[ -f "$f" ] || { echo "docs/consolidation.md is missing — the consolidation cadence has no record. Recreate it."; exit 0; }

last=$(grep -oE '^## [0-9]{4}-[0-9]{2}-[0-9]{2}' "$f" | tail -1 | awk '{print $2}' || true)
if [ -z "$last" ]; then
  echo "docs/consolidation.md has no dated pass entry — the consolidation cadence is untracked."
  exit 0
fi

days=$(( ( $(date +%s) - $(date -d "$last" +%s) ) / 86400 ))
if [ "$days" -ge 28 ]; then
  echo "Doc consolidation OVERDUE: last pass $last (${days}d ago; cadence 28d). Additive-only docs bloat without it (agentic-development.md, known defects). Ask the owner to authorize a consolidation pass (touch .claude/docs-unlock), then record it in docs/consolidation.md."
fi
exit 0
