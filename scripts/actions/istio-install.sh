# istio-install - Istio ambient profile on the current kubeconfig
# context's cluster. Environment prerequisite (docs/plans/01-week-one.md),
# not compiled by d7s.
require_repo_root

# Gateway API CRDs are an environment prerequisite the harness must
# install itself: `istioctl install --set profile=ambient -y` does NOT
# install them, yet week-four's egress compiler emits a waypoint Gateway
# (gateway.networking.k8s.io/v1) into every mesh-joined app namespace
# (internal/adapters/flux/flux.go) - without the CRD, that apply fails
# ("no matches for kind Gateway in version gateway.networking.k8s.io/v1"),
# the app Kustomization never reaches Ready, and deliver's bounded wait
# times out with no clearer symptom (confirmed live on a throwaway kind
# cluster, both directions: CI runs 30254082510/30270332686, and applying
# these CRDs alone made the identical waypoint apply succeed). kind
# doesn't ship the Gateway API CRDs the way some managed distros do.
#
# Version v1.5.1, standard channel (the release channel with only GA
# resources - a waypoint Gateway needs nothing from the experimental
# channel): the version Istio's own docs pin for istioctl 1.30.x, the
# version this repo's flake.nix resolves istioctl to via its pinned
# nixpkgs revision (https://istio.io/latest/docs/ambient/getting-started/,
# checked 2026-07-27). Vendored into the repo (scripts/vendor/gateway-api)
# rather than fetched live at install time - the same determinism idiom
# flake.nix uses to pin every binary this harness runs, applied here to a
# manifest instead: a re-run next year still applies exactly this CRD
# set, offline-capable, never a moving upstream target.
log "gateway-api: install standard channel CRDs (prerequisite for compiled waypoints, v1.5.1)"
kubectl apply -f scripts/vendor/gateway-api/standard-install-v1.5.1.yaml
kubectl wait --for=condition=Established crd/gateways.gateway.networking.k8s.io --timeout="$TIMEOUT"

log "istio: install ambient profile"
istioctl install --set profile=ambient -y
