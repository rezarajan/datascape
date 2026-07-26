# deliver - register the git-source action's content with Flux (the
# GitRepository CR the emitter's Kustomizations already name), apply only
# the two emitted Flux Kustomization CRs, and drive the on-demand
# reconcile sequencing down to a healthy Cluster: kustomize-controller /
# helm-controller do the rest, never a direct apply of compiled objects
# (docs/plans/01-week-one.md's Flux-reconciliation wiring).
require_repo_root

log "flux: git push  # register the git source, named/namespaced for the emitter's sourceRef"
register_git_source

log "apply ONLY the emitted Flux Kustomization CRs - kustomize-controller/helm-controller reconcile the rest"
kubectl apply -f "$OUT/flux/infra-cnpg-operator.yaml"
kubectl apply -f "$OUT/flux/apps-week-one.yaml"

log "flux delivers infra layer: cnpg operator (namespace, HelmRepository, HelmRelease)"
poll "Kustomization cnpg-operator reconciled by flux" bash -c \
	"flux reconcile kustomization cnpg-operator -n flux-system --with-source >/dev/null 2>&1; \
	 [ \"\$(kubectl get kustomization cnpg-operator -n flux-system -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}' 2>/dev/null)\" = 'True' ]"
poll "CNPG operator HelmRelease ready" bash -c \
	"[ \"\$(kubectl get helmrelease cnpg-operator -n cnpg-system -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}' 2>/dev/null)\" = 'True' ]"
kubectl wait --for=condition=Available deployment -n cnpg-system --all --timeout="$TIMEOUT"

log "flux delivers app layer: namespace, Cluster CR, zero-trust"
echo "(the emitted Kustomization's dependsOn already waits on the infra layer;"
echo " forced here now that CNPG's CRDs actually exist - rule 44: this poll is"
echo " the wait, no fixed-duration sleep for that gap)"
poll "Kustomization week-one reconciled by flux" bash -c \
	"flux reconcile kustomization week-one -n flux-system --with-source >/dev/null 2>&1; \
	 [ \"\$(kubectl get kustomization week-one -n flux-system -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}' 2>/dev/null)\" = 'True' ]"

log "environment prerequisite: create the declared credentials secret"
echo "(CNPG's bootstrap.initdb.secret only consumes a pre-existing secret - "
echo " see internal/domain/secret.go; the week-one namespace already exists,"
echo " created by flux reconciling the Kustomization above)"
kubectl create secret generic orders-db-app -n week-one \
	--from-literal=username=orders-db \
	--from-literal=password="$(openssl rand -hex 16)" \
	--dry-run=client -o yaml | kubectl apply -f -

log "wait: Cluster reaches healthy state"
poll "Cluster healthy" bash -c \
	"[ \"\$(kubectl get cluster orders-db -n week-one -o jsonpath='{.status.phase}' 2>/dev/null)\" = 'Cluster in healthy state' ]"
