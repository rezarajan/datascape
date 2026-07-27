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
# GITSERVER_IMAGE default carries $$ (this script's own PID), not a fixed
# string (found live, week-three slice 6: two concurrent host runs -
# `nix run .#acceptance` and `nix run .#acceptance-managed` - both build,
# `kind load`, then `docker rmi` the SAME local tag `d7s-gitserver:harness`
# in git-source.sh; racing that build/load/rmi cycle on one shared,
# mutable tag let one run's `docker rmi` interfere with the other's still-
# in-flight `kind load docker-image`, so the loaded image reached the node
# but the gitserver process inside it never actually served git - Flux's
# GitRepository sat un-Ready until the bounded wait timed out with no
# other symptom). $$ isn't derived from $CLUSTER_NAME because the managed
# orchestrator (acceptance-managed.sh) only reassigns CLUSTER_NAME AFTER
# common.sh (and thus this default) has already run and exported - a
# CLUSTER_NAME-keyed tag would silently keep resolving to the pre-
# reassignment default for every managed run, never actually
# differentiating concurrent ones. git-source.sh itself needs no change -
# it already reads $GITSERVER_IMAGE rather than a literal.
export GITSERVER_IMAGE="${GITSERVER_IMAGE:-d7s-gitserver:harness-$$}"

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
# The exact-host pin ceremony's FIRST phase (the same ceremony both
# example files document in their own header comments): the unpinned
# declaration is what compile-managed compiles and deliver-managed
# delivers, because the real endpoint host does not exist until
# tofu-controller provisions it. pin-managed then materializes the
# pinned phase from MANAGED_STACK with the live host and redelivers.
export MANAGED_STACK_UNPINNED="${MANAGED_STACK_UNPINNED:-examples/week-two/managed-stack-unpinned.yaml}"
export MANAGED_OUT="${MANAGED_OUT:-./out-managed}"
export MANAGED_CLUSTER_NAME="${MANAGED_CLUSTER_NAME:-d7s-acceptance-managed}"
# The namespace/Kustomization name examples/week-two/managed-stack.yaml's
# own declared stack name compiles to (internal/adapters/flux/flux.go:
# namespace and Kustomization are both named after Stack.Name).
export MANAGED_NAMESPACE="${MANAGED_NAMESPACE:-week-one}"
# MANAGED_COMPONENT / MANAGED_CREDENTIALS_SECRET (dogfood note 2, finding 1:
# "the managed harness actions hardcode the example component name"). Same
# mechanism as MANAGED_NAMESPACE right above — an overridable env var
# defaulting from examples/week-two/managed-stack.yaml's own declared values
# — rather than parsing the stack yaml at run time: the declaration is
# already the harness's one source of truth for MANAGED_STACK/MANAGED_OUT/
# MANAGED_NAMESPACE, so a fifth value following the same shape keeps every
# unit a plain env-var read (simple, shellcheck-flat) instead of teaching
# each one a yaml-parsing dependency for a value that's a one-line override
# either way. MANAGED_COMPONENT is both the Terraform CR's name
# (internal/adapters/flux/terraform.go: named after pg.Name) and the Neon
# branch/role/database name (same field, templated as "{{.Name}}").
# MANAGED_CREDENTIALS_SECRET is the component's declared
# credentials.secretRef.name — the Secret tofu-controller's
# writeOutputsToSecret populates and the probe reads from, independent of
# the component name (the schema allows them to differ).
export MANAGED_COMPONENT="${MANAGED_COMPONENT:-orders-db}"
export MANAGED_CREDENTIALS_SECRET="${MANAGED_CREDENTIALS_SECRET:-orders-db-app}"
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

_poll_timeout_diagnostics() {
	# The waypoint/Gateway-API CRD blocker (see require_gateway_api_prereq)
	# took two identical CI reds and an out-of-band live repro on a
	# throwaway kind cluster to diagnose - poll_n's own TIMEOUT line names
	# only the wait's description, nothing about cluster-side state, and
	# by the time a human goes looking the orchestrator's EXIT trap has
	# already torn the cluster down. Dump bounded, clearly-labeled state
	# right here, before that happens - this is a timeout diagnostic, not
	# the check itself: every command is best-effort (`|| true`) so a
	# cluster that's unreachable for the SAME reason the wait timed out
	# reports as exactly that, never a second, louder failure that
	# obscures the first.
	{
		echo "--- timeout diagnostics (bounded, best-effort) ---"
		if command -v flux >/dev/null 2>&1; then
			echo "-- flux get kustomizations -A --"
			flux get kustomizations -A 2>&1 || true
			echo "-- flux get helmreleases -A --"
			flux get helmreleases -A 2>&1 || true
		fi
		echo "-- kubectl get events -A --sort-by=.lastTimestamp (last 30) --"
		kubectl get events -A --sort-by=.lastTimestamp 2>&1 | tail -n 30 || true
		echo "--- end timeout diagnostics ---"
	} >&2
}

poll_n() {
	# poll_n <attempts> <interval> <description> <command...> - the
	# general form: retries under an honest bounded deadline sized to
	# what the specific wait actually costs (golden rule 44: a bound that
	# matches reality, not a blanket widening of every wait it shares
	# `poll`'s own default budget with). `poll` below is the common case.
	local attempts="$1" interval="$2" desc="$3"
	shift 3
	# Announce the wait as it starts, not only on timeout (found live: a
	# bounded poll that only speaks up at the end reads as a silent hang to
	# a human operator - one line, no spinner, same discipline as `log`).
	echo "waiting (bounded): $desc" >&2
	local i
	for ((i = 1; i <= attempts; i++)); do
		if "$@"; then
			return 0
		fi
		sleep "$interval"
	done
	echo "TIMEOUT waiting for: $desc" >&2
	_poll_timeout_diagnostics
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

# shellcheck disable=SC2329 # shared helper - only deliver.sh calls this
require_flux_prereq() {
	# deliver's very first `kubectl apply` targets a Kustomization CR in
	# namespace flux-system (internal/adapters/flux/flux.go's emitted
	# objects) - without Flux installed, that apply fails with a raw
	# "namespaces \"flux-system\" not found" instead of naming the actual
	# missing prerequisite. Same class of trap dogfood note 3 found for
	# MinIO (docs/dogfood.md, 2026-07-26) - fail closed with the remedy
	# instead (rules 34, 35).
	if ! kubectl get namespace flux-system >/dev/null 2>&1; then
		echo "refusing to deliver: namespace flux-system not found - Flux is an environment prerequisite for delivery - remedy: run 'nix run .#flux-install' first; the 'nix run .#acceptance' orchestrator does this for you" >&2
		exit 1
	fi
}

# shellcheck disable=SC2329 # shared helper - both acceptance
# orchestrators call this before their first cluster operation
acquire_cluster_lock() {
	# One orchestrator per cluster name at a time. Found live, 2026-07-27
	# (TASK_PROGRESS): two concurrent acceptance runs shared the same
	# cluster name, and cluster-up's clean-slate delete destroyed the
	# first run's cluster mid-flight — a phantom failure that read as a
	# rollout timeout, not as the collision it was. The lock is held on
	# fd 9 for the orchestrator's whole lifetime (children inherit it;
	# flock releases on process exit, so a crashed run never wedges the
	# next one). Refuse loudly with the remedy (rules 34, 35), same
	# discipline as every require_*_prereq above.
	local name="$1"
	exec 9>"/tmp/d7s-kind-$name.lock"
	if ! flock -n 9; then
		echo "refusing to run: another acceptance run holds cluster $name (lock /tmp/d7s-kind-$name.lock) - remedy: wait for that run to finish, or set CLUSTER_NAME/MANAGED_CLUSTER_NAME to a unique value for a parallel cluster" >&2
		exit 1
	fi
}

# shellcheck disable=SC2329 # shared helper - only deliver.sh calls this
require_istio_prereq() {
	# guarantees.mtls compiles a PeerAuthentication/AuthorizationPolicy
	# pair with nothing to enforce against unless Istio ambient mode is
	# installed - without it, Flux's own reconciliation of those objects
	# never reaches Ready and deliver's poll below just times out after
	# its full bounded budget with a generic message, not a clear
	# prerequisite name. Fail closed up front instead (rules 34, 35),
	# same reasoning as require_flux_prereq above.
	if ! kubectl get crd peerauthentications.security.istio.io >/dev/null 2>&1; then
		echo "refusing to deliver: Istio ambient mode is not installed (CRD peerauthentications.security.istio.io not found) - guarantees.mtls has no mesh to enforce against - remedy: run 'nix run .#istio-install' first; the 'nix run .#acceptance' orchestrator does this for you" >&2
		exit 1
	fi
}

# shellcheck disable=SC2329 # shared helper - deliver.sh and
# deliver-managed.sh both call this
require_gateway_api_prereq() {
	# week-four's egress compiler emits a waypoint Gateway
	# (gateway.networking.k8s.io/v1) into every mesh-joined app namespace
	# (internal/adapters/flux/flux.go) - without the Gateway API CRDs
	# installed, that apply fails outright ("no matches for kind Gateway
	# in version gateway.networking.k8s.io/v1"), the app Kustomization
	# never reaches Ready, and deliver's poll below just times out after
	# its full bounded budget with no clearer symptom (confirmed live on
	# a throwaway kind cluster, both directions: CI runs
	# 30254082510/30270332686). istio-install installs these CRDs
	# (vendored, scripts/vendor/gateway-api) alongside Istio itself, but
	# a piecemeal path that skips it hits the identical trap
	# require_istio_prereq guards against above. Fail closed with the
	# remedy instead (rules 34, 35), same reasoning.
	if ! kubectl get crd gateways.gateway.networking.k8s.io >/dev/null 2>&1; then
		echo "refusing to deliver: the Gateway API CRDs are not installed (CRD gateways.gateway.networking.k8s.io not found) - compiled waypoints have nothing to apply against - remedy: run 'nix run .#istio-install' first; the 'nix run .#acceptance' orchestrator does this for you" >&2
		exit 1
	fi
}

# shellcheck disable=SC2329 # shared helper - only deliver.sh calls this
require_minio_prereq() {
	# The declared external's backing store must exist before deliver's
	# minio-secret step reads its root credentials out of $MINIO_NS (dogfood
	# note 3, docs/dogfood.md, 2026-07-26: the first human cold run hit a
	# raw "namespaces \"d7s-harness-minio\" not found" right here, because
	# the piecemeal QUICKSTART sequence it followed omitted minio-install).
	# Fail closed with the remedy instead (rules 34, 35, 49).
	if ! kubectl get namespace "$MINIO_NS" >/dev/null 2>&1 || ! kubectl get deployment minio -n "$MINIO_NS" >/dev/null 2>&1; then
		echo "refusing to deliver: MinIO not found (namespace $MINIO_NS / deployment minio) - MinIO is a harness prerequisite for this stack's durability guarantee - remedy: run 'nix run .#minio-install' first; the 'nix run .#acceptance' orchestrator does this for you" >&2
		exit 1
	fi
}

# shellcheck disable=SC2329 # shared helper - deliver.sh and
# deliver-managed.sh both call this (both call register_git_source)
require_gitserver_prereq() {
	# register_git_source (below) applies a GitRepository naming the
	# in-cluster git server git-source stands up - without it, the
	# GitRepository just sits un-Ready, DNS-failing on
	# d7s-gitserver.$GITSERVER_NS.svc (the namespace doesn't even exist),
	# which register_git_source's own poll then waits out its full
	# bounded budget for - reading as a silent hang to a human, one step
	# over from dogfood note 3's MinIO trap (docs/dogfood.md,
	# 2026-07-26) - same class of bug, found live on the same piecemeal
	# path. Fail closed with the remedy instead (rules 34, 35), before
	# ever registering the GitRepository.
	if ! kubectl get namespace "$GITSERVER_NS" >/dev/null 2>&1 || ! kubectl get deployment d7s-gitserver -n "$GITSERVER_NS" >/dev/null 2>&1; then
		echo "refusing to deliver: the in-cluster git source is not found (namespace $GITSERVER_NS / deployment d7s-gitserver) - the in-cluster git source is a prerequisite - remedy: run 'nix run .#git-source' first; the orchestrators do this for you" >&2
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
