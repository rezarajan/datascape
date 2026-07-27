#!/usr/bin/env bash
# prereq-refusals - cluster-free, network-free conformance test for the
# harness's fail-closed environment-prerequisite checks and poll_n's wait
# announcement (scripts/lib/common.sh).
#
# Contract-level repro for a live-caught bug (rule 39): dogfood note 3
# (docs/dogfood.md, 2026-07-26) found `deliver` failing with a raw
# Kubernetes NotFound instead of a remedy when MinIO wasn't installed; the
# same-day follow-up found the identical trap for the in-cluster git
# source, plus a bounded poll that only speaks up at timeout (reading as a
# hang). Both fixes landed without anything pinning them against
# regression - this test is that pin.
#
# Every require_*_prereq check in common.sh only branches on `kubectl
# get ...`'s exit status, never its stdout (see each function's body) -
# so standing in for "found" vs "not found" needs nothing more than a
# stub `kubectl` on PATH whose exit code is driven by one env var. No
# real cluster, no network, no golden Kubernetes objects to fake.
#
# Fail-then-pass discipline throughout (rule 49): every check is asserted
# to refuse (with its remedy) when the stub reports "absent", AND to
# succeed when the stub reports "present" - a check that always refuses
# would pass this test's first half silently forever.
set -euo pipefail
cd "$(dirname "$0")/../.."

STUB_DIR="$(mktemp -d)"
trap 'rm -rf "$STUB_DIR"' EXIT

cat >"$STUB_DIR/kubectl" <<'STUB'
#!/usr/bin/env bash
# Fake kubectl - every require_*_prereq check calls `kubectl get ...`
# only for its exit status (redirected to /dev/null), so this is the
# entire fake: report "found" or "not found" via one env var, ignore
# every argument.
exit "${STUB_KUBECTL_EXIT:-0}"
STUB
chmod +x "$STUB_DIR/kubectl"
export PATH="$STUB_DIR:$PATH"

# shellcheck source=../lib/common.sh
# shellcheck disable=SC1091
source scripts/lib/common.sh

pass=0
fail=0

# assert_refuses <desc> <prereq-fn> <remedy-substring> - calls <prereq-fn>
# with the stub reporting "not found" (STUB_KUBECTL_EXIT=1). Run inside a
# command substitution (its own subshell) deliberately: every
# require_*_prereq calls `exit 1` on refusal, not `return 1`, so this is
# the only safe way to invoke them without killing this test script.
assert_refuses() {
	local desc="$1" fn="$2" remedy_substr="$3"
	local out status
	if out="$(STUB_KUBECTL_EXIT=1 "$fn" 2>&1)"; then
		status=0
	else
		status=$?
	fi
	if [ "$status" -eq 0 ]; then
		echo "FAIL: $desc: expected a nonzero (refusing) exit, got 0" >&2
		fail=$((fail + 1))
		return
	fi
	case "$out" in
	*"$remedy_substr"*) ;;
	*)
		echo "FAIL: $desc: remedy '$remedy_substr' not found in: $out" >&2
		fail=$((fail + 1))
		return
		;;
	esac
	echo "PASS: $desc (refuses, remedy present)"
	pass=$((pass + 1))
}

# assert_passes <desc> <prereq-fn> - same check, stub reporting "found"
# (STUB_KUBECTL_EXIT=0): the other half of fail-then-pass.
assert_passes() {
	local desc="$1" fn="$2"
	local status
	if STUB_KUBECTL_EXIT=0 "$fn" >/dev/null 2>&1; then
		status=0
	else
		status=$?
	fi
	if [ "$status" -ne 0 ]; then
		echo "FAIL: $desc: expected exit 0 against a present stub, got $status" >&2
		fail=$((fail + 1))
		return
	fi
	echo "PASS: $desc (succeeds when present)"
	pass=$((pass + 1))
}

assert_refuses "require_flux_prereq / absent" require_flux_prereq "nix run .#flux-install"
assert_passes "require_flux_prereq / present" require_flux_prereq

assert_refuses "require_istio_prereq / absent" require_istio_prereq "nix run .#istio-install"
assert_passes "require_istio_prereq / present" require_istio_prereq

assert_refuses "require_minio_prereq / absent" require_minio_prereq "nix run .#minio-install"
assert_passes "require_minio_prereq / present" require_minio_prereq

assert_refuses "require_gitserver_prereq / absent" require_gitserver_prereq "nix run .#git-source"
assert_passes "require_gitserver_prereq / present" require_gitserver_prereq

assert_refuses "require_gateway_api_prereq / absent" require_gateway_api_prereq "nix run .#istio-install"
assert_passes "require_gateway_api_prereq / present" require_gateway_api_prereq

# poll_n: announces the wait as it starts, not only on timeout (found
# live: an operator running deliver without git-source watched Flux
# DNS-fail silently through the whole bounded budget - a bounded wait
# that only speaks up at the end is indistinguishable from a hang).
# succeed_on_second fails its first call and succeeds its second - run
# inside poll_n's own retry loop (interval 0, no real wait) to prove both
# halves at once: the announcement prints before the command ever
# succeeds, and reaching success on attempt 2 never prints TIMEOUT.
ATTEMPTS=0
succeed_on_second() {
	ATTEMPTS=$((ATTEMPTS + 1))
	[ "$ATTEMPTS" -ge 2 ]
}

if poll_out="$(poll_n 5 0 "test wait" succeed_on_second 2>&1)"; then
	poll_status=0
else
	poll_status=$?
fi

if [ "$poll_status" -ne 0 ]; then
	echo "FAIL: poll_n: expected success by attempt 2, got exit $poll_status: $poll_out" >&2
	fail=$((fail + 1))
elif [ "${poll_out#*"waiting (bounded): test wait"}" = "$poll_out" ]; then
	echo "FAIL: poll_n did not announce the wait at start: $poll_out" >&2
	fail=$((fail + 1))
elif [ "${poll_out#*TIMEOUT}" != "$poll_out" ]; then
	echo "FAIL: poll_n reported TIMEOUT despite succeeding on attempt 2: $poll_out" >&2
	fail=$((fail + 1))
else
	echo "PASS: poll_n announces the wait at start, succeeds without a false timeout"
	pass=$((pass + 1))
fi

echo
echo "$pass passed, $fail failed"
[ "$fail" -eq 0 ]
