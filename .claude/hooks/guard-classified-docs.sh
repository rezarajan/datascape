#!/usr/bin/env bash
# PreToolUse guard (Edit|Write): contract and record docs change only as a deliberate,
# human-authorized act — never as a task side effect (golden rule 68; agentic-development §7).
#
# The protected set is DERIVED from the docs map (docs/README.md) at run time: every row
# classed Contract or Record. No duplicated roster to drift.
#
# Human unlock for an authorized maintenance/append pass:
#   touch .claude/docs-unlock      (gitignored; delete when the pass is done)
# Unlocked edits are logged to .claude/guard.log (gitignored) for audit.
set -euo pipefail

input=$(cat)
file=$(jq -r '.tool_input.file_path // empty' <<<"$input")
[ -z "$file" ] && exit 0

root="${CLAUDE_PROJECT_DIR:-$PWD}"
rel="${file#"$root"/}"

# Derive protected repo-relative paths from the docs-map table.
protected=$(awk -F'|' '/^\|/ {
  path=$2; cls=$3
  gsub(/[ `]/, "", path); gsub(/ /, "", cls)
  gsub(/\*\*/, "", cls)
  if (cls == "Contract" || cls == "Record") print "docs/" path
}' "$root/docs/README.md")

if ! grep -Fxq "$rel" <<<"$protected"; then
  exit 0
fi

if [ -f "$root/.claude/docs-unlock" ]; then
  printf '%s UNLOCKED edit: %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$rel" >>"$root/.claude/guard.log"
  exit 0
fi

cat >&2 <<EOF
BLOCKED: '$rel' is classified Contract or Record on the docs map (docs/README.md).
It changes only as a deliberate, human-authorized act — a dated amendment (contracts)
or an append (records), never as a task side effect.
If the owner has authorized this pass: ask them to run 'touch .claude/docs-unlock',
make the change, then delete the marker. Unlocked edits are logged to .claude/guard.log.
EOF
exit 2
