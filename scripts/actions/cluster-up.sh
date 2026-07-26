# cluster-up - a fresh, ephemeral kind cluster. Deletes any pre-existing
# cluster of the same name first so a re-run always starts clean.
require_repo_root

log "kind cluster: create (fresh, ephemeral)"
kind delete cluster --name "$CLUSTER_NAME" >/dev/null 2>&1 || true
kind create cluster --name "$CLUSTER_NAME"
kubectl wait --for=condition=Ready node --all --timeout="$TIMEOUT"
