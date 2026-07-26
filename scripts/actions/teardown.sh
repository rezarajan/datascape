# teardown - delete the ephemeral kind cluster. Invoked directly for a
# manual cleanup, and via the acceptance orchestrator's EXIT trap so a
# failed run still tears down.
log "tearing down cluster $CLUSTER_NAME (ephemeral per run)"
kind delete cluster --name "$CLUSTER_NAME" >/dev/null 2>&1 || true
