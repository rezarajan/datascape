#!/usr/bin/env bash
# Fast tier: what a developer waits for on every change (golden rule 43).
# Every gate here is demonstrated able to fail, not just able to pass —
# `gofmt -l` in particular exits 0 whether or not it found anything, so its
# output is compared explicitly (agentic-development.md "known defects").
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> gofmt"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
	echo "not gofmt-formatted:" >&2
	echo "$unformatted" >&2
	exit 1
fi

echo "==> go vet"
go vet ./...

echo "==> go build"
go build ./...

echo "==> go test"
go test ./...

echo "fast tier green"
