# pin-managed - the exact-host pin ceremony's SECOND phase (week-four
# plan Revision 2; both example files in examples/week-two/ document the
# same ceremony from the operator's seat). Runs after deliver-managed has
# brought the Terraform CR to Ready, so the written-outputs secret
# carries the live endpoint host tofu-controller provisioned:
#
#   1. read the host from the secret's own "host" key - the DECLARED
#      credentials.secretRef.name, never a value this script hardcodes;
#   2. materialize the pinned declaration from $MANAGED_STACK (the
#      committed phase-two template, whose placeholder host its header
#      comment explains) into a throwaway file;
#   3. recompile - only now does d7s compile the exact-host ServiceEntry
#      + AuthorizationPolicy declared consumers need (domain validation
#      refuses allowedConsumers without the pin, naming this ceremony);
#   4. republish and wait for Flux to reconcile the pinned output - the
#      plan is the git diff, phase two included (git-source is
#      re-invoked wholesale: same scratch-repo publish, fresh commit).
require_repo_root

log "pin ceremony: read the provisioned endpoint host from the written-outputs secret"
PINNED_HOST="$(kubectl get secret "$MANAGED_CREDENTIALS_SECRET" -n "$MANAGED_NAMESPACE" -o jsonpath='{.data.host}' | base64 -d)"
if [ -z "$PINNED_HOST" ]; then
	echo "refusing to pin: secret $MANAGED_NAMESPACE/$MANAGED_CREDENTIALS_SECRET carries no host key - the Terraform CR must reach Ready (writeOutputsToSecret) before the pin ceremony - remedy: run 'nix run .#deliver-managed' first; the 'nix run .#acceptance-managed' orchestrator orders this for you" >&2
	exit 1
fi

log "pin ceremony: materialize the pinned declaration (endpointHost: $PINNED_HOST)"
PINNED_STACK="$(mktemp -t d7s-pinned-stack.XXXXXX)"
sed "s|endpointHost: .*|endpointHost: $PINNED_HOST|" "$MANAGED_STACK" >"$PINNED_STACK"

log "pin ceremony: recompile with the pinned host (the exact-host ServiceEntry must now compile)"
EXPECT_FILE="$MANAGED_OUT/apps/$MANAGED_NAMESPACE/$MANAGED_COMPONENT-neon-serviceentry.yaml" \
	STACK="$PINNED_STACK" \
	OUT="$MANAGED_OUT" \
	compile-and-verify
rm -f "$PINNED_STACK"

log "pin ceremony: republish the pinned output through the same git source"
PRE_PIN_REVISION="$(kubectl get gitrepository d7s -n flux-system -o jsonpath='{.status.artifact.revision}')"
git-source

log "pin ceremony: flux reconciles the pinned revision"
flux reconcile source git d7s -n flux-system >/dev/null
poll "git source serves the pinned revision (artifact revision changed)" bash -c \
	"[ \"\$(kubectl get gitrepository d7s -n flux-system -o jsonpath='{.status.artifact.revision}')\" != '$PRE_PIN_REVISION' ]"
flux reconcile kustomization "$MANAGED_NAMESPACE" -n flux-system --with-source >/dev/null 2>&1 || true
poll "pinned exact-host ServiceEntry applied by flux" bash -c \
	"kubectl get serviceentry $MANAGED_COMPONENT-neon -n $MANAGED_NAMESPACE >/dev/null 2>&1"
