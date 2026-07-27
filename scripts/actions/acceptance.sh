# acceptance - thin orchestrator. Composes the units above in the same
# order, and with the same trap-based ephemeral teardown, as the
# monolithic scripts/acceptance-kind.sh it replaces
# (docs/plans/02-week-two.md Revision 3, slice 6). Every unit named here
# is independently runnable too (`nix run .#<unit>`) - this script's only
# job is sequencing.
#
# Environment prerequisites and their ordering constraints (week-three
# plan, slice 3 adds minio-install): flux-install before istio-install
# (no real dependency, kept in this order historically); minio-install
# before git-source/deliver - the declared external's backing store must
# exist and hold its bucket before the delivered Cluster ever attempts a
# backup, mirroring how the CNPG operator (installed via the infra
# Kustomization deliver reconciles) must exist before the app layer's
# Cluster CR is usable. minio-install does NOT itself create the
# app-namespace backups-credentials secret (that namespace doesn't exist
# yet) - deliver.sh invokes minio-secret for that, positioned right
# alongside the CNPG credentials secret (orders-db-app), once the
# namespace exists and before the Cluster is expected to be healthy.
require_repo_root

# Serialize on the cluster name and pin this run to its own kubeconfig.
# Both halves close the same live-caught failure class (2026-07-27,
# TASK_PROGRESS): concurrent same-name runs deleting each other's
# cluster, and kind's context switching handing a NEIGHBORING cluster's
# context to this run's kubectl calls mid-step.
acquire_cluster_lock "$CLUSTER_NAME"
KUBECONFIG="$(mktemp -t d7s-kubeconfig.XXXXXX)"
export KUBECONFIG

trap 'teardown; rm -f "$KUBECONFIG"' EXIT

compile-and-verify
cluster-up
flux-install
istio-install
minio-install
git-source
deliver
guard
probes
durability-probe

log "acceptance scenario PASSED"
