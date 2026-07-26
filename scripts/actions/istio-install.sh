# istio-install - Istio ambient profile on the current kubeconfig
# context's cluster. Environment prerequisite (docs/plans/01-week-one.md),
# not compiled by d7s.
log "istio: install ambient profile"
istioctl install --set profile=ambient -y
