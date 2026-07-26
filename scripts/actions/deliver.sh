# deliver - register the git-source action's content with Flux (the
# GitRepository CR the emitter's Kustomizations already name), apply only
# the two emitted Flux Kustomization CRs, and let kustomize-controller /
# helm-controller reconcile the rest down to a healthy Cluster - never a
# direct apply of compiled objects (docs/plans/01-week-one.md's
# Flux-reconciliation wiring).
#
# week-three plan, slice 4 (closing the week-two slice-4 live finding): the
# emitted infra Kustomization now carries spec.healthChecks naming the CNPG
# operator's HelmRelease (internal/adapters/flux/flux.go, emitCNPGOperator),
# so its own Ready condition means the operator is genuinely serving, not
# merely that its manifests applied. The app Kustomization's existing
# dependsOn on the infra Kustomization (emitAppKustomization) therefore
# gates on that real readiness by itself - Flux requires ALL of a
# dependency's health checks to pass before a dependent applies. This
# closes the gap: no more explicit operator/HelmRelease/Deployment wait,
# no more forcing either Kustomization's reconcile on demand. What's left
# below is a bounded wait on the app Kustomization's own outcome (it is
# gated by the compiled dependsOn+healthChecks, not re-implemented here)
# and, further down, the legitimate bounded wait on the Cluster's own
# final healthy state (rule 44: waiting for an outcome is fine: re-doing
# the ordering procedurally is not).
#
# dogfood note 3 (docs/dogfood.md, 2026-07-26): the first human cold run
# hit a raw Kubernetes NotFound here instead of a remedy, because the
# piecemeal QUICKSTART sequence it followed omitted minio-install. Cheap,
# up-front prerequisite checks (scripts/lib/common.sh) now cover every
# environment prerequisite this script itself depends on before doing
# anything - Flux (its first kubectl apply targets flux-system), Istio
# (guarantees.mtls has nothing to enforce against without it - otherwise
# only surfaces later as a generic poll timeout), MinIO (the durability
# guarantee's declared external, dogfood note 3's own failure), and the
# in-cluster git source register_git_source is about to register a
# GitRepository against (found live, one step over: without it the
# GitRepository just DNS-fails silently through register_git_source's
# whole bounded poll budget) - so a cold operator gets a named
# prerequisite and its remedy, never a raw API error or a wait that
# reads as a hang.
require_repo_root
require_flux_prereq
require_istio_prereq
require_minio_prereq
require_gitserver_prereq

log "flux: git push  # register the git source, named/namespaced for the emitter's sourceRef"
register_git_source

log "apply ONLY the emitted Flux Kustomization CRs - kustomize-controller/helm-controller reconcile the rest"
kubectl apply -f "$OUT/flux/infra-cnpg-operator.yaml"
kubectl apply -f "$OUT/flux/apps-week-one.yaml"

log "flux delivers app layer: namespace, Cluster CR, zero-trust"
echo "(compiled ordering alone: the app Kustomization's dependsOn on"
echo " cnpg-operator won't apply until cnpg-operator's own healthChecks -"
echo " the operator's HelmRelease - report Ready, so this poll observes"
echo " Flux's own reconciliation, it does not drive it)"
poll "Kustomization week-one reconciled by flux (gated on cnpg-operator health via dependsOn+healthChecks)" bash -c \
	"[ \"\$(kubectl get kustomization week-one -n flux-system -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}' 2>/dev/null)\" = 'True' ]"

log "environment prerequisite: create the declared credentials secret"
echo "(CNPG's bootstrap.initdb.secret only consumes a pre-existing secret - "
echo " see internal/domain/secret.go; the week-one namespace already exists,"
echo " created by flux reconciling the Kustomization above)"
kubectl create secret generic orders-db-app -n week-one \
	--from-literal=username=orders-db \
	--from-literal=password="$(openssl rand -hex 16)" \
	--dry-run=client -o yaml | kubectl apply -f -

log "environment prerequisite: create the declared external's credentials secret"
echo "(same reason as orders-db-app above - CNPG's barmanObjectStore.s3Credentials"
echo " only consumes a pre-existing secret; minio-install already stood up MinIO"
echo " and its bucket earlier in the orchestrator - see acceptance.sh - but the"
echo " credentials secret itself must land here, now that the week-one namespace"
echo " exists, not inside minio-install)"
minio-secret

log "wait: Cluster reaches healthy state"
poll "Cluster healthy" bash -c \
	"[ \"\$(kubectl get cluster orders-db -n week-one -o jsonpath='{.status.phase}' 2>/dev/null)\" = 'Cluster in healthy state' ]"
