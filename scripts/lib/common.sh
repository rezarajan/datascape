# Shared config and helpers for the acceptance harness's flake-exposed
# actions (docs/plans/02-week-two.md Revision 3, slice 6 - decomposing the
# former monolith, scripts/acceptance-kind.sh). This file is embedded into
# every unit at nix-build time (see flake.nix's `mkAction`) rather than
# sourced at runtime, so each unit stays a single self-contained script
# that shellcheck verifies at build time - but the values and helpers
# below still have exactly one source of truth, so behavior never forks
# between units
# (golden rule 44: the bounded `poll` wait is defined once, here, and
# shared - never duplicated).
#
# Every value is overridable by env, same as the monolith it replaces.

export CLUSTER_NAME="${CLUSTER_NAME:-d7s-acceptance}"
export TIMEOUT="${TIMEOUT:-300s}"
export POLL_ATTEMPTS="${POLL_ATTEMPTS:-40}"
export POLL_INTERVAL="${POLL_INTERVAL:-5}"
export STACK="${STACK:-examples/week-one/stack.yaml}"
export OUT="${OUT:-./out}"
export GITSERVER_NS="${GITSERVER_NS:-d7s-harness-git}"
export GITSERVER_IMAGE="${GITSERVER_IMAGE:-d7s-gitserver:harness}"

log() { printf '\n==> %s\n' "$1"; }

poll() {
	# poll <description> <command...> - retries under an honest bounded
	# deadline (golden rule 44: no fixed-duration sleeps; one knob scales
	# every wait every action shares).
	local desc="$1"
	shift
	local i
	for ((i = 1; i <= POLL_ATTEMPTS; i++)); do
		if "$@"; then
			return 0
		fi
		sleep "$POLL_INTERVAL"
	done
	echo "TIMEOUT waiting for: $desc" >&2
	return 1
}

require_repo_root() {
	# Fail closed (rules 34/35): every action assumes it runs from the
	# datascape repo root, the same assumption the monolith made via
	# `cd "$(dirname "$0")/.."` - `nix run .#<action>` doesn't get a
	# meaningful $0 to relativize from, so this checks the landmark files
	# instead and refuses loudly, remedy included, if they're missing.
	if [ ! -f go.mod ] || [ ! -f flake.nix ]; then
		echo "refusing to run: not at the datascape repo root (go.mod / flake.nix not found in \$PWD) - remedy: run 'nix run .#<action>' from the repository root" >&2
		exit 1
	fi
}
