#!/usr/bin/env bash
# Acceptance harness (docs/plans/01-week-one.md): the documented
# scenario, exercised exactly as an operator would type it, against a
# real kind cluster with Flux + Istio ambient + CloudNativePG. Same
# script locally and in CI (TASK_PROGRESS.md, 2026-07-26 re-plan).
#
# Environment prerequisites this week (not compiled by d7s — week-one
# plan, "explicitly NOT this week"):
#   - Flux and Istio ambient installed on the target cluster.
#   - The postgres component's credentials Secret exists before the
#     compiled Cluster CR is applied (CNPG's bootstrap.initdb.secret
#     only consumes a pre-existing secret; see internal/domain/secret.go).
#   - A StorageClass with reclaimPolicy: Retain, if the deployment needs
#     golden rule 28's retain-on-delete honored — CNPG has no field of
#     its own for this, it is purely a StorageClass property. This
#     script does not set one up; PVC retention-on-delete is therefore
#     not exercised here (a known, deliberate gap — golden rule 7).
set -euo pipefail
cd "$(dirname "$0")/.."

CLUSTER_NAME="${CLUSTER_NAME:-d7s-acceptance}"
TIMEOUT="${TIMEOUT:-300s}"
POLL_ATTEMPTS="${POLL_ATTEMPTS:-40}"
POLL_INTERVAL="${POLL_INTERVAL:-5}"
STACK="examples/week-one/stack.yaml"
OUT="./out"

log() { printf '\n==> %s\n' "$1"; }

poll() {
	# poll <description> <command...> — retries under an honest bounded
	# deadline (golden rule 44: no fixed-duration sleeps; one knob scales
	# every wait in this script).
	local desc="$1"
	shift
	local i
	for ((i = 1; i <= POLL_ATTEMPTS; i++)); do
		if "$@"; then
			return 0
		fi
		sleep "$POLL_INTERVAL"
	done
	echo "TIMEOUT waiting for: $desc" >&2
	return 1
}

cleanup() {
	log "tearing down cluster $CLUSTER_NAME (ephemeral per run)"
	kind delete cluster --name "$CLUSTER_NAME" >/dev/null 2>&1 || true
}
trap cleanup EXIT

log "compile (deterministic, read-only) — the documented invocation"
go build -o /tmp/d7s ./cmd/d7s
/tmp/d7s compile "$STACK" -o "$OUT"
test -f "$OUT/apps/week-one/orders-db-cluster.yaml"

log "determinism: two compiles are byte-identical (rules 22, 45)"
/tmp/d7s compile "$STACK" -o /tmp/d7s-out-2
diff -rq "$OUT" /tmp/d7s-out-2
rm -rf /tmp/d7s-out-2

log "negative probe: unsatisfiable RPO refuses to compile (rules 34, 35, 49)"
cat >/tmp/d7s-bad-rpo.yaml <<'EOF'
apiVersion: d7s.dev/v1alpha1
kind: Stack
name: bad-rpo
components:
  - kind: postgres
    name: db
    placement: self-hosted
    credentials:
      secretRef:
        name: db-app
    guarantees:
      rpo: 2m
EOF
if /tmp/d7s compile /tmp/d7s-bad-rpo.yaml -o /tmp/d7s-bad-rpo-out 2>/tmp/d7s-bad-rpo.err; then
	echo "expected compilation to refuse an unsatisfiable RPO" >&2
	exit 1
fi
grep -q "cannot be honored" /tmp/d7s-bad-rpo.err
rm -f /tmp/d7s-bad-rpo.yaml /tmp/d7s-bad-rpo.err
rm -rf /tmp/d7s-bad-rpo-out

log "kind cluster: create (fresh, ephemeral)"
kind delete cluster --name "$CLUSTER_NAME" >/dev/null 2>&1 || true
kind create cluster --name "$CLUSTER_NAME"
kubectl wait --for=condition=Ready node --all --timeout="$TIMEOUT"

log "flux: install controllers"
flux install

log "istio: install ambient profile"
istioctl install --set profile=ambient -y

log "git add out/ && git commit && git push  # the plan is the git diff"
echo "(simulated: this script applies out/ directly — see out/'s own"
echo " Flux Kustomization objects for the real GitOps wiring shape)"

log "apply infra layer: cnpg operator (namespace, HelmRepository, HelmRelease)"
kubectl apply -f "$OUT/infra/cnpg-operator/namespace.yaml"
kubectl apply -f "$OUT/infra/cnpg-operator/helmrepository.yaml"
kubectl apply -f "$OUT/infra/cnpg-operator/helmrelease.yaml"
poll "CNPG operator HelmRelease ready" bash -c \
	"[ \"\$(kubectl get helmrelease cnpg-operator -n cnpg-system -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}' 2>/dev/null)\" = 'True' ]"
kubectl wait --for=condition=Available deployment -n cnpg-system --all --timeout="$TIMEOUT"

log "environment prerequisite: create the declared credentials secret"
echo "(CNPG's bootstrap.initdb.secret only consumes a pre-existing secret —"
echo " see internal/domain/secret.go)"
kubectl apply -f "$OUT/apps/week-one/namespace.yaml"
kubectl create secret generic orders-db-app -n week-one \
	--from-literal=username=orders-db \
	--from-literal=password="$(openssl rand -hex 16)" \
	--dry-run=client -o yaml | kubectl apply -f -

log "apply app layer: Cluster CR, zero-trust, durability"
kubectl apply -f "$OUT/apps/week-one/orders-db-cluster.yaml"
kubectl apply -f "$OUT/apps/week-one/peerauthentication.yaml"
kubectl apply -f "$OUT/apps/week-one/orders-db-authorizationpolicy.yaml"
kubectl apply -f "$OUT/apps/week-one/orders-db-scheduledbackup.yaml"

log "wait: Cluster reaches healthy state"
poll "Cluster healthy" bash -c \
	"[ \"\$(kubectl get cluster orders-db -n week-one -o jsonpath='{.status.phase}' 2>/dev/null)\" = 'Cluster in healthy state' ]"

PASSWORD=$(kubectl get secret orders-db-app -n week-one -o jsonpath='{.data.password}' | base64 -d)

log "positive probe: declared consumer connects over mTLS and runs SQL (rule 41)"
kubectl create serviceaccount probe-client -n week-one --dry-run=client -o yaml | kubectl apply -f -
kubectl run probe-client -n week-one --image=postgres:17 --restart=Never \
	--overrides="{\"spec\":{\"serviceAccountName\":\"probe-client\"}}" -- sleep 3600
kubectl wait --for=condition=Ready pod/probe-client -n week-one --timeout="$TIMEOUT"
kubectl exec -n week-one probe-client -- env PGPASSWORD="$PASSWORD" \
	psql -h orders-db-rw -U orders-db -d orders-db -c "SELECT 1;"

log "negative probe: off-mesh plaintext client is REFUSED (rule 49)"
kubectl run offmesh-client -n default --image=postgres:17 --restart=Never -- sleep 3600
kubectl wait --for=condition=Ready pod/offmesh-client -n default --timeout="$TIMEOUT"
if kubectl exec -n default offmesh-client -- env PGPASSWORD="$PASSWORD" \
	psql -h orders-db-rw.week-one -U orders-db -d orders-db -c "SELECT 1;" 2>/dev/null; then
	echo "expected the off-mesh client to be refused, but it connected" >&2
	exit 1
fi

log "durability probe: ScheduledBackup fires, a Backup object appears"
poll "Backup object appears" bash -c \
	"[ \"\$(kubectl get backup -n week-one --no-headers 2>/dev/null | wc -l)\" -gt 0 ]"
kubectl get backup -n week-one

log "acceptance scenario PASSED"
