# acceptance-managed - the managed/Neon acceptance scenario (docs/plans/
# 02-week-two.md Revision 3, slice 5). Composed as its own entry point
# rather than chained after the self-hosted `acceptance` orchestrator,
# for two reasons found while building this slice:
#
# 1. examples/week-two/managed-stack.yaml mirrors examples/week-one/
#    stack.yaml's stack name exactly (the seam proof's whole point -
#    only placement flips), so delivering both scenarios' compiled
#    output into the SAME cluster would collide (same namespace, same
#    Flux Kustomization name). Running each on its own ephemeral kind
#    cluster (CLUSTER_NAME vs MANAGED_CLUSTER_NAME) sidesteps that
#    without renaming either example.
# 2. The owner's own CI-tiering directive (Revision 3) already treats
#    the two scenarios as separate tiers - self-hosted always runs, the
#    managed one only where NEON_API_KEY exists - which maps directly
#    onto two independently runnable entry points
#    (`nix run .#acceptance` / `nix run .#acceptance-managed`) rather
#    than one script with an internal skip branch.
require_repo_root

log "API-key process: NEON_API_KEY from the environment, or a gitignored ./.env"
require_neon_api_key

# Resolved once, up front, and exported so every child action (including
# teardown-managed, invoked via the EXIT trap even if an earlier step
# fails) inherits it - week-two plan Revision 4: the Neon project is an
# environment prerequisite, discovered from the project-scoped key
# rather than requiring a second env var (see discover_neon_project_id).
if [ -z "${NEON_PROJECT_ID:-}" ]; then
	log "discover the Neon project id NEON_API_KEY is scoped to"
	NEON_PROJECT_ID=$(discover_neon_project_id)
fi
export NEON_PROJECT_ID

export STACK="$MANAGED_STACK"
export OUT="$MANAGED_OUT"
export CLUSTER_NAME="$MANAGED_CLUSTER_NAME"

# Same serialization + kubeconfig isolation as acceptance.sh — the
# live-caught concurrent-run collision class (2026-07-27, TASK_PROGRESS).
acquire_cluster_lock "$CLUSTER_NAME"
KUBECONFIG="$(mktemp -t d7s-kubeconfig.XXXXXX)"
export KUBECONFIG

trap 'teardown-managed; rm -f "$KUBECONFIG"' EXIT

compile-managed
cluster-up
# flux-install before tofu-install: tofu-controller's own release
# manifests (rbac, deployment) target the flux-system namespace, which
# only `flux install` creates - found live, 2026-07-26 (repeated
# "namespaces flux-system not found" errors when the order was
# reversed). istio-install (which also installs the Gateway API CRDs)
# became mandatory here with week-four's egress compiler: a managed
# stack now emits a waypoint Gateway, exact-host ServiceEntries, and
# identity-scoped AuthorizationPolicies (THE MESH IS MANDATORY — the
# week-four plan's verbatim owner directive), so without Istio the
# delivery refuses at require_gateway_api_prereq (observed as the
# named refusal in CI runs 30276159658/30297205201; before that guard,
# the same gap was a silent Kustomization timeout). The pre-week-four
# comment here claimed the opposite from an assumption that mtls was
# the only mesh trigger — superseded by the egress compiler.
flux-install
istio-install
tofu-install
git-source
deliver-managed
# The exact-host pin ceremony's second phase (pin-managed's own header
# documents it): only after the Terraform CR is Ready does the live
# endpoint host exist to pin, recompile, and redeliver - and only the
# pinned revision compiles the exact-host ServiceEntry +
# AuthorizationPolicy the probes below exercise.
pin-managed
probe-managed

log "managed acceptance scenario PASSED"
