#!/usr/bin/env bash
# PreToolUse guard (Task/Agent): deny subagent spawns that would silently inherit the
# parent's (expensive) model (agentic-development §7, §10 — model economy as policy).
#
# Allowed: an explicit 'model' in the call, or a checked-in agent whose frontmatter pins
# one. The pinned roster is DERIVED from .claude/agents/*.md frontmatter at run time —
# the platformctl defect was a duplicated roster that drifted; do not add a list here.
set -euo pipefail

input=$(cat)
tool=$(jq -r '.tool_name // empty' <<<"$input")
case "$tool" in Task|Agent) ;; *) exit 0 ;; esac

model=$(jq -r '.tool_input.model // empty' <<<"$input")
[ -n "$model" ] && exit 0

stype=$(jq -r '.tool_input.subagent_type // empty' <<<"$input")
root="${CLAUDE_PROJECT_DIR:-$PWD}"

if [ -n "$stype" ] && [ -f "$root/.claude/agents/$stype.md" ]; then
  # Pinned iff the agent's YAML frontmatter (first --- block) declares a model.
  if awk '/^---$/{n++; next} n==1 && /^model:[[:space:]]*[^[:space:]]/ {found=1} END{exit !found}' \
      "$root/.claude/agents/$stype.md"; then
    exit 0
  fi
fi

cat >&2 <<EOF
BLOCKED: this subagent spawn would silently inherit the parent model.
Model economy is policy (agentic-development §10): pass an explicit 'model'
(cheap unless the task is genuine design judgment or root-cause-unknown work),
or use a checked-in agent from .claude/agents/ whose frontmatter pins one.
EOF
exit 2
