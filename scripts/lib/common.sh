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
# DESTROY_POLL_ATTEMPTS gives the Terraform CR's destroy leg its own,
# honestly-sized budget (found live, 2026-07-26: the shared 200s
# POLL_ATTEMPTS*POLL_INTERVAL budget that comfortably covers the apply
# leg was not enough for destroy - a fresh runner pod cold-starts and
# re-runs `tofu init`, re-downloading the Neon provider, before it can
# even plan the destroy; the two legs cost about the same PLUS that
# re-init). Double the apply leg's budget - a bound matching reality,
# not sleep-padding (golden rule 44's spirit).
export DESTROY_POLL_ATTEMPTS="${DESTROY_POLL_ATTEMPTS:-80}"
export STACK="${STACK:-examples/week-one/stack.yaml}"
export OUT="${OUT:-./out}"
export GITSERVER_NS="${GITSERVER_NS:-d7s-harness-git}"
export GITSERVER_IMAGE="${GITSERVER_IMAGE:-d7s-gitserver:harness}"

# MinIO — the acceptance stack's declared `external` object store (week-
# three plan, slice 3): stood up by the harness as environment
# scaffolding, exactly like Flux/Istio above — never d7s-compiled or
# mutated (problem definition Amendment 2, "external by provenance").
# Its own namespace mirrors GITSERVER_NS's naming ("d7s-harness-<thing>").
export MINIO_NS="${MINIO_NS:-d7s-harness-minio}"
export MINIO_SERVICE="${MINIO_SERVICE:-minio}"
export MINIO_BUCKET="${MINIO_BUCKET:-d7s-backups}"
# Pinned exact release tags (verified pullable, 2026-07-26) — harness
# scaffolding, not compiled output, but pinned anyway rather than
# `:latest` so a re-run next month isn't the first thing to notice an
# upstream break.
export MINIO_IMAGE="${MINIO_IMAGE:-minio/minio:RELEASE.2024-11-07T00-52-20Z}"
export MC_IMAGE="${MC_IMAGE:-minio/mc:RELEASE.2024-11-05T11-29-45Z}"
# MINIO_ROOT_CREDS_SECRET_NAME lives in MinIO's own namespace (never the
# app namespace) — minio-install generates the value once per run and
# stores it there; minio-secret reads it back to materialize the
# app-namespace credentials Secret examples/week-one/stack.yaml's
# external names (the same two-step shape neon-secret/deliver-managed
# already use for the identical "app namespace doesn't exist yet"
# constraint).
export MINIO_ROOT_CREDS_SECRET_NAME="${MINIO_ROOT_CREDS_SECRET_NAME:-minio-root-creds}"
# Must match internal/adapters/flux/durability.go's
# objectStoreAccessKeyIDSecretKey / objectStoreSecretAccessKeySecretKey
# constants exactly — the harness materializes what the compiled
# Cluster's barmanObjectStore.s3Credentials references by these key
# names, in whichever Secret the declared external's credentials.secretRef
# names.
export OBJECT_STORE_ACCESS_KEY_ID_KEY="${OBJECT_STORE_ACCESS_KEY_ID_KEY:-ACCESS_KEY_ID}"
export OBJECT_STORE_SECRET_ACCESS_KEY_KEY="${OBJECT_STORE_SECRET_ACCESS_KEY_KEY:-ACCESS_SECRET_KEY}"
# Must match examples/week-one/stack.yaml's declared external's
# credentials.secretRef.name exactly.
export OBJECT_STORE_CREDENTIALS_SECRET_NAME="${OBJECT_STORE_CREDENTIALS_SECRET_NAME:-backups-credentials}"

# Managed/Neon scenario defaults (docs/plans/02-week-two.md Revision 3,
# slice 5). The managed orchestrator overrides STACK/OUT/CLUSTER_NAME to
# these before calling the same shared actions the self-hosted scenario
# uses (cluster-up, git-source, teardown) - a separate ephemeral cluster,
# never the self-hosted one, so the two scenarios' compiled trees (both
# happen to declare the stack name "week-one" - managed-stack.yaml
# mirrors stack.yaml with only placement flipped, the seam proof's whole
# point) never collide.
export MANAGED_STACK="${MANAGED_STACK:-examples/week-two/managed-stack.yaml}"
export MANAGED_OUT="${MANAGED_OUT:-./out-managed}"
export MANAGED_CLUSTER_NAME="${MANAGED_CLUSTER_NAME:-d7s-acceptance-managed}"
# The namespace/Kustomization name examples/week-two/managed-stack.yaml's
# own declared stack name compiles to (internal/adapters/flux/flux.go:
# namespace and Kustomization are both named after Stack.Name).
export MANAGED_NAMESPACE="${MANAGED_NAMESPACE:-week-one}"
# tofu-controller version pin (week-two plan slice 1's health verdict,
# TASK_PROGRESS 2026-07-26: v0.16.4, OpenTofu-first, pinnable).
export TOFU_CONTROLLER_VERSION="${TOFU_CONTROLLER_VERSION:-v0.16.4}"
# Must match internal/adapters/flux/terraform.go's neonAPIKeySecretName /
# neonAPIKeySecretKey constants exactly - the harness materializes what
# the compiled Terraform CR references by name.
export NEON_API_KEY_SECRET_NAME="${NEON_API_KEY_SECRET_NAME:-neon-api-key}"
export NEON_API_KEY_SECRET_KEY="${NEON_API_KEY_SECRET_KEY:-apiKey}"
# Must match internal/adapters/flux/terraform.go's neonProjectIDSecretKey
# constant exactly (week-two plan Revision 4: branch-per-stack - the
# Neon project id lives alongside the API key in the same Secret).
export NEON_PROJECT_ID_SECRET_KEY="${NEON_PROJECT_ID_SECRET_KEY:-projectId}"
# SKIP_EXIT_CODE is the documented exit status an action uses to report
# "skipped, not failed" (rule 49: skipped reports unknown, never
# coverage) - distinct from 0 (passed) and 1 (a real failure), so the
# orchestrator and CI step can tell the three apart by exit code alone,
# never by scraping output text.
export SKIP_EXIT_CODE=75

log() { printf '\n==> %s\n' "$1"; }

poll_n() {
	# poll_n <attempts> <interval> <description> <command...> - the
	# general form: retries under an honest bounded deadline sized to
	# what the specific wait actually costs (golden rule 44: a bound that
	# matches reality, not a blanket widening of every wait it shares
	# `poll`'s own default budget with). `poll` below is the common case.
	local attempts="$1" interval="$2" desc="$3"
	shift 3
	local i
	for ((i = 1; i <= attempts; i++)); do
		if "$@"; then
			return 0
		fi
		sleep "$interval"
	done
	echo "TIMEOUT waiting for: $desc" >&2
	return 1
}

poll() {
	# poll <description> <command...> - retries under the shared default
	# bounded deadline (golden rule 44: no fixed-duration sleeps; one
	# knob scales every wait that doesn't need its own).
	local desc="$1"
	shift
	poll_n "$POLL_ATTEMPTS" "$POLL_INTERVAL" "$desc" "$@"
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

# shellcheck disable=SC2329 # shared helper - not every action embedding
# common.sh calls it (e.g. teardown-managed already has NEON_API_KEY in
# its environment from the orchestrator and never re-reads .env itself).
require_neon_api_key() {
	# require_neon_api_key - the API-key process (week-two plan Revision 3,
	# owner directive verbatim): NEON_API_KEY is NEVER in version control.
	# It enters via the environment, falling back to sourcing a gitignored
	# ./.env at the repo root. The value is never printed, echoed, or
	# logged here or anywhere it's passed on to - only exported so the
	# caller can materialize the in-cluster Secret from it. A missing key
	# is a SKIP, not a FAIL (rule 49): the managed scenario cannot be
	# verified this run, but that must never look like a passing check, so
	# it exits with SKIP_EXIT_CODE rather than 0 or 1.
	if [ -z "${NEON_API_KEY:-}" ] && [ -f ./.env ]; then
		set -a
		# shellcheck disable=SC1091
		. ./.env
		set +a
	fi
	if [ -z "${NEON_API_KEY:-}" ]; then
		echo "SKIPPED (unknown): NEON_API_KEY not set — managed seam NOT verified this run" >&2
		echo "remedy: export NEON_API_KEY=<neon-api-key>, or place it in a gitignored ./.env at the repo root (NEON_API_KEY=...) — never commit it" >&2
		exit "$SKIP_EXIT_CODE"
	fi
	export NEON_API_KEY
}

discover_neon_project_id() {
	# discover_neon_project_id - the Neon project is an environment
	# prerequisite (week-two plan Revision 4), exactly like the
	# Kubernetes cluster itself - d7s never creates it, so the harness
	# must be told, or find out, which one NEON_API_KEY is scoped to.
	# Neon exposes no plain "which project is this key scoped to"
	# endpoint (checked, 2026-07-26); the one reliable discovery path is
	# that a project-scoped key calling the project-LISTING endpoint
	# (which only makes sense at org scope) fails with its own project id
	# named in the error body's subject_project_id field - guaranteed
	# present precisely because that's the mismatch being reported. Fails
	# loudly, never guesses, if the response doesn't match that shape.
	# NEON_PROJECT_ID in the environment/.env always wins over discovery
	# (an explicit override escape hatch, same precedence as every other
	# harness value).
	if [ -n "${NEON_PROJECT_ID:-}" ]; then
		printf '%s' "$NEON_PROJECT_ID"
		return 0
	fi
	local message id
	message=$(curl -s -m 30 "https://console.neon.tech/api/v2/projects" \
		-H "Authorization: Bearer $NEON_API_KEY" -H "Accept: application/json" \
		| jq -r '.message // empty')
	id=$(printf '%s' "$message" | grep -oP 'subject_project_id:"\K[a-zA-Z0-9-]+' || true)
	if [ -z "$id" ]; then
		echo "could not discover the Neon project id NEON_API_KEY is scoped to (expected a project-scoped-key error naming subject_project_id, got: $message)" >&2
		echo "remedy: set NEON_PROJECT_ID explicitly in the environment or ./.env" >&2
		exit 1
	fi
	printf '%s' "$id"
}

# shellcheck disable=SC2329 # shared helper - not every action embedding
# common.sh calls it (only deliver and deliver-managed register a git
# source; e.g. teardown-managed never touches git at all).
register_git_source() {
	# register_git_source - apply the GitRepository CR every emitted
	# Kustomization/Terraform CR's sourceRef names, and wait for flux to
	# clone it. Shared by deliver and deliver-managed (slice 6's DRY
	# principle): both scenarios' compiled output reads from a
	# same-shaped git source, just a different cluster/content per run.
	kubectl apply -f - <<EOF
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: d7s
  namespace: flux-system
spec:
  interval: 1m
  url: http://d7s-gitserver.$GITSERVER_NS.svc.cluster.local/d7s.git
  ref:
    branch: main
EOF
	poll "GitRepository d7s ready (flux cloned the git source)" bash -c \
		"flux reconcile source git d7s -n flux-system >/dev/null 2>&1; \
		 [ \"\$(kubectl get gitrepository d7s -n flux-system -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}' 2>/dev/null)\" = 'True' ]"
}
