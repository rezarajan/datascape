# tofu-install - tofu-controller (flux-iac), pinned at
# $TOFU_CONTROLLER_VERSION, on the current kubeconfig context's cluster.
# Environment prerequisite (docs/plans/02-week-two.md, slice 1's health
# verdict, TASK_PROGRESS 2026-07-26: pinnable, OpenTofu-first since
# v0.16.0), not compiled by d7s - it joins flux-install/istio-install as
# a declared prerequisite, same reasoning as both. The three manifests
# applied are the exact release-tagged assets (never the `main`-branch
# release.yaml the project's own getting-started guide points at, which
# would make the install itself a moving, unpinned target).
require_repo_root

TOFU_CONTROLLER_BASE="https://github.com/flux-iac/tofu-controller/releases/download/$TOFU_CONTROLLER_VERSION"

log "tofu-controller: install $TOFU_CONTROLLER_VERSION (crds, rbac, deployment)"
echo "(--server-side for the CRDs: client-side apply's last-applied-configuration"
echo " annotation exceeds Kubernetes' 262144-byte limit on this CRD - found by"
echo " running this action live against a real kind cluster, golden rule 40)"
kubectl apply --server-side -f "$TOFU_CONTROLLER_BASE/tofu-controller.crds.yaml"
kubectl apply -f "$TOFU_CONTROLLER_BASE/tofu-controller.rbac.yaml"

# Root cause, found live 2026-07-26 (Terraform CR condition: "cannot
# access GitRepository/flux-system/d7s, cross-namespace references have
# been disabled"): tofu-controller v0.16 defaults to
# --allow-cross-namespace-refs=false (verified against the pinned
# v0.16.4 tag's cmd/manager/main.go), refusing a Terraform CR's sourceRef
# into a different namespace than the CR itself. kustomize-controller in
# this same cluster defaults the OTHER way (cross-namespace
# GitRepository refs allowed) - which is exactly how every emitted
# Kustomization already reads the flux-system GitRepository "d7s" from
# its own stack namespace. Steward decision: enable it here, matching
# kustomize-controller's own existing posture rather than weakening
# anything beyond it. Rejected alternatives: emitting the Terraform CR
# into flux-system itself would break stack-namespace ownership;
# compiling a per-stack GitRepository would multiply environment setup
# for no isolation gain in a single-tenant v1, and would still need the
# git URL - an environment binding compiled output cannot contain. Full
# multi-tenancy hardening (cross-namespace refs disabled everywhere,
# namespace-local sources emitted per stack) is named future trust-
# boundary/skeleton work, not this week's.
#
# A first attempt applied the release manifest unmodified, then
# `kubectl patch`ed the flag into the running Deployment - found live to
# race the controller's own first rollout: the fresh Terraform CR was
# reconciled (and permanently Stalled) by the ORIGINAL unpatched pod
# before the patched replacement ever became Ready, so the flag never
# governed the pod that actually processed the CR. Fixed deterministically
# instead: inject the flag into the release-pinned manifest BEFORE the
# first apply, so only one pod version - the correctly configured one -
# is ever created. No rollout, no race.
log "tofu-controller: install $TOFU_CONTROLLER_VERSION's deployment with cross-namespace refs enabled from the first apply"
DEPLOYMENT_MANIFEST="$(mktemp)"
curl -sL -m 30 "$TOFU_CONTROLLER_BASE/tofu-controller.deployment.yaml" -o "$DEPLOYMENT_MANIFEST"
if ! grep -q -- '- --log-encoding=json' "$DEPLOYMENT_MANIFEST"; then
	echo "tofu-install: the downloaded deployment manifest's args list doesn't match what this fix expects (looking for '- --log-encoding=json') - refusing to guess" >&2
	echo "remedy: re-verify tofu-controller.deployment.yaml's exact args shape for $TOFU_CONTROLLER_VERSION and update this action" >&2
	rm -f "$DEPLOYMENT_MANIFEST"
	exit 1
fi
sed -i 's/- --log-encoding=json/&\n            - --allow-cross-namespace-refs/' "$DEPLOYMENT_MANIFEST"
kubectl apply -f "$DEPLOYMENT_MANIFEST"
rm -f "$DEPLOYMENT_MANIFEST"

kubectl rollout status deployment/tofu-controller -n flux-system --timeout="$TIMEOUT"

log "verify precondition: the running tofu-controller actually has --allow-cross-namespace-refs"
echo "(never assume the manifest edit took - delivery must not start against an unverified controller config)"
running_args=$(kubectl get deployment tofu-controller -n flux-system -o jsonpath='{.spec.template.spec.containers[0].args}')
if ! printf '%s' "$running_args" | grep -q -- '--allow-cross-namespace-refs'; then
	echo "tofu-install: the running tofu-controller deployment's args do not include --allow-cross-namespace-refs (got: $running_args)" >&2
	echo "remedy: the flag name/semantics may be wrong for $TOFU_CONTROLLER_VERSION - re-verify against cmd/manager/main.go for that tag" >&2
	exit 1
fi

# Warm the runner image the deployment above defaults to (verified
# against tofu-controller.deployment.yaml's own env var, 2026-07-26).
# tofu-controller spawns one runner pod per Terraform CR reconcile on
# demand - on a fresh kind node with nothing cached, that first pull
# (307MB, found live) can outlast the bounded Ready-wait poll a later
# action runs, timing it out for a reason that has nothing to do with
# Terraform/Neon at all.
#
# `kind load docker-image` was tried first and rejected: it re-imports
# via `ctr images import --all-platforms`, which failed live
# ("content digest ... not found") - the well-known kind/containerd
# multi-arch import gap when the local Docker image lacks every
# platform's blobs that `--all-platforms` demands. Warming the image
# through the kubelet's own pull path instead - a real, disposable pod -
# sidesteps that gap entirely and is what actually determines whether a
# later runner pod can start instantly: it exercises the exact same pull
# mechanism a runner pod itself uses, not a side-door import.
TF_RUNNER_IMAGE="ghcr.io/flux-iac/tf-runner:$TOFU_CONTROLLER_VERSION"
log "tofu-controller: warm the runner image ($TF_RUNNER_IMAGE) via a disposable pod"
# Full pod manifest (rather than `kubectl run ... --command`, which has no
# flag surface for securityContext) so the warm-up pod itself satisfies the
# Kubernetes `restricted` Pod Security Standard (dogfood note 2, finding 4:
# flux-system already carries pod-security.kubernetes.io/warn=restricted -
# flux's own install labels its namespace this way - so every pod created in
# it is checked against `restricted`, cosmetic-only on a plain kind cluster
# but a hard admission-time rejection on a restricted-enforcing one). The
# pod's only job is to exec `true` and exit - it touches no filesystem path
# that could care which non-root UID it runs as.
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: tf-runner-warm
  namespace: flux-system
spec:
  restartPolicy: Never
  securityContext:
    runAsNonRoot: true
    runAsUser: 65532
    seccompProfile:
      type: RuntimeDefault
  containers:
    - name: tf-runner-warm
      image: $TF_RUNNER_IMAGE
      command: ["true"]
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop: ["ALL"]
EOF
poll "tf-runner-warm pod Succeeded (image pulled)" bash -c \
	"[ \"\$(kubectl get pod tf-runner-warm -n flux-system -o jsonpath='{.status.phase}' 2>/dev/null)\" = 'Succeeded' ]"
kubectl delete pod tf-runner-warm -n flux-system --ignore-not-found
