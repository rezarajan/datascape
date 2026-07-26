# acceptance - thin orchestrator. Composes the units above in the same
# order, and with the same trap-based ephemeral teardown, as the
# monolithic scripts/acceptance-kind.sh it replaces
# (docs/plans/02-week-two.md Revision 3, slice 6). Every unit named here
# is independently runnable too (`nix run .#<unit>`) - this script's only
# job is sequencing.
require_repo_root

trap 'teardown' EXIT

compile-and-verify
cluster-up
flux-install
istio-install
git-source
deliver
guard
probes

log "acceptance scenario PASSED"
