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

trap 'teardown-managed' EXIT

compile-managed
cluster-up
# flux-install before tofu-install: tofu-controller's own release
# manifests (rbac, deployment) target the flux-system namespace, which
# only `flux install` creates - found live, 2026-07-26 (repeated
# "namespaces flux-system not found" errors when the order was
# reversed). istio-install is deliberately NOT part of this scenario:
# guarantees.mtls + placement: managed refuses to compile (week-two
# plan), so a managed-only stack never emits a mesh object for istio to
# enforce - installing it here would cost time proving nothing.
flux-install
tofu-install
git-source
deliver-managed
probe-managed

log "managed acceptance scenario PASSED"
