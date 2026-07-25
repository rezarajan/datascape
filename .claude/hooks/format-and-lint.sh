#!/usr/bin/env bash
# PostToolUse (Edit|Write): format-and-lint the edited file (agentic-development §7).
# Dispatch by extension. Extend this case statement when a language lands in the repo;
# a language with no entry here is a known gap, not silent coverage (golden rule 7).
set -euo pipefail

input=$(cat)
file=$(jq -r '.tool_input.file_path // empty' <<<"$input")
[ -n "$file" ] && [ -f "$file" ] || exit 0

case "$file" in
  *.sh)
    if ! err=$(bash -n "$file" 2>&1); then
      printf 'Shell syntax error in %s:\n%s\n' "$file" "$err" >&2
      exit 2
    fi
    if command -v shellcheck >/dev/null 2>&1; then
      if ! out=$(shellcheck -S warning "$file" 2>&1); then
        printf '%s\n' "$out" >&2
        exit 2
      fi
    fi
    ;;
  *.json)
    if ! err=$(jq empty "$file" 2>&1); then
      printf 'Invalid JSON in %s:\n%s\n' "$file" "$err" >&2
      exit 2
    fi
    ;;
esac
exit 0
