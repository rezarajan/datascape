# deliver-managed - register the git-source action's content with flux
# (the GitRepository CR the emitter's Kustomizations already name), apply
# only the managed stack's emitted Flux Kustomization CR, and drive the
# on-demand reconcile down to a Ready Terraform CR: kustomize-controller
# creates the namespace + Terraform CR, tofu-controller reconciles the
# OpenTofu config from there - never a direct apply of compiled objects,
# the same discipline as the self-hosted scenario's deliver action
# (docs/plans/02-week-two.md, slice 5: "the SAME git-source + Flux path
# as the self-hosted scenario").
#
# require_gitserver_prereq (scripts/lib/common.sh): this action registers
# a git source exactly like deliver.sh does, and hits the identical trap
# without it - the GitRepository DNS-fails silently through
# register_git_source's whole bounded poll budget instead of naming the
# missing prerequisite. Fail closed with the remedy instead, before ever
# registering the GitRepository.
require_repo_root
require_gitserver_prereq

log "flux: git push  # register the git source, named/namespaced for the emitter's sourceRef"
register_git_source

log "apply ONLY the emitted Flux Kustomization CR for the managed stack"
kubectl apply -f "$MANAGED_OUT/flux/apps-$MANAGED_NAMESPACE.yaml"

log "flux delivers the managed app layer: namespace + terraform CR"
poll "Kustomization $MANAGED_NAMESPACE reconciled by flux" bash -c \
	"flux reconcile kustomization $MANAGED_NAMESPACE -n flux-system --with-source >/dev/null 2>&1; \
	 [ \"\$(kubectl get kustomization $MANAGED_NAMESPACE -n flux-system -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}' 2>/dev/null)\" = 'True' ]"

log "environment prerequisite: materialize the neon-api-key secret the Terraform CR references"
echo "(the same per-run pattern as the self-hosted scenario's CNPG credentials secret -"
echo " see internal/domain/secret.go and scripts/actions/deliver.sh; the namespace above"
echo " already exists, created by flux reconciling the Kustomization)"
neon-secret

log "wait: Terraform CR reaches Ready (tofu-controller reconciled the OpenTofu config)"
if ! poll "Terraform $MANAGED_COMPONENT Ready" bash -c \
	"[ \"\$(kubectl get terraform $MANAGED_COMPONENT -n $MANAGED_NAMESPACE -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}' 2>/dev/null)\" = 'True' ]"; then
	# A bounded wait that times out silently is undiagnosable by design
	# (found live, 2026-07-26) - capture everything teardown is about to
	# make unrecoverable, BEFORE failing this action and letting the
	# orchestrator's EXIT trap tear the cluster down.
	log "DIAGNOSTICS: Terraform $MANAGED_COMPONENT did not reach Ready - capturing state before teardown"
	echo "--- kubectl describe terraform $MANAGED_COMPONENT -n $MANAGED_NAMESPACE ---"
	kubectl describe terraform "$MANAGED_COMPONENT" -n "$MANAGED_NAMESPACE" || true
	echo "--- kubectl get pods -n $MANAGED_NAMESPACE -o wide ---"
	kubectl get pods -n "$MANAGED_NAMESPACE" -o wide || true
	for pod in $(kubectl get pods -n "$MANAGED_NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
		echo "--- kubectl describe pod/$pod -n $MANAGED_NAMESPACE ---"
		kubectl describe pod "$pod" -n "$MANAGED_NAMESPACE" || true
		echo "--- kubectl logs pod/$pod -n $MANAGED_NAMESPACE --all-containers ---"
		kubectl logs "$pod" -n "$MANAGED_NAMESPACE" --all-containers --tail=300 || true
	done
	echo "--- kubectl get events -n $MANAGED_NAMESPACE --sort-by=.lastTimestamp ---"
	kubectl get events -n "$MANAGED_NAMESPACE" --sort-by=.lastTimestamp || true
	echo "--- kubectl logs deploy/tofu-controller -n flux-system --tail=40 ---"
	kubectl logs deploy/tofu-controller -n flux-system --tail=40 || true
	exit 1
fi
