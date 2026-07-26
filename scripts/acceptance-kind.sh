#!/usr/bin/env bash
# Acceptance harness (docs/plans/01-week-one.md; Flux-delivery wiring
# per docs/plans/02-week-two.md Revision 2, slice 4): the documented
# scenario, exercised exactly as an operator would type it, against a
# real kind cluster with Flux + Istio ambient + CloudNativePG. The
# compiled out/ tree is delivered the documented GitOps way — pushed to
# an in-cluster git source, then reconciled by kustomize-controller /
# helm-controller from the emitted Flux Kustomization CRs — never
# kubectl-applied directly; a guard near the end fails loudly if any
# compiled object was ever touched by a manager other than
# kustomize-controller. Same script locally and in CI (TASK_PROGRESS.md,
# 2026-07-26 re-plan); both run it through the pinned toolchain in
# flake.nix:
#   nix develop --command scripts/acceptance-kind.sh
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
#
# The git source itself (a bare repo served over the git smart-HTTP
# protocol by a throwaway in-cluster pod, built from debian-slim —
# Alpine's git package ships no git-http-backend binary, checked live
# before wiring this in) is harness scaffolding, not a declared
# environment prerequisite and not compiled output: it exists only for
# this run's lifetime, same as the kind cluster itself.
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

log "negative probe: any declared RPO refuses to compile (rules 34, 35, 49)"
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
	echo "expected compilation to refuse a declared RPO" >&2
	exit 1
fi
grep -q "durability guarantee's conformance probe could never pass" /tmp/d7s-bad-rpo.err
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

log "git source: stand up the lightest viable in-cluster git server (smart HTTP)"
GITSERVER_NS="d7s-harness-git"
GITSERVER_IMAGE="d7s-gitserver:harness"
GITSERVER_BUILD_DIR="$(mktemp -d)"
cat >"$GITSERVER_BUILD_DIR/Dockerfile" <<'DOCKERFILE'
# Harness-only git server: not compiled output, not product code. It
# exists solely to give Flux's source-controller a real git source to
# clone from, over the git smart-HTTP protocol (the only transports
# Flux's GitRepository CRD accepts are http/https/ssh — no git:// or
# file://, checked against the upstream CRD schema before this was
# written).
FROM debian:12-slim
RUN apt-get update -qq \
	&& DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends \
	   git nginx-light fcgiwrap ca-certificates \
	&& rm -rf /var/lib/apt/lists/* \
	&& mkdir -p /srv/git
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
CMD ["sh", "-c", "fcgiwrap -s unix:/run/fcgiwrap.sock & exec nginx -g 'daemon off;'"]
DOCKERFILE
cat >"$GITSERVER_BUILD_DIR/nginx.conf" <<'NGINXCONF'
user root;
worker_processes 1;
pid /run/nginx.pid;
events { worker_connections 64; }
http {
	server {
		listen 80 default_server;
		location ~ ^/d7s\.git(/.*)?$ {
			root /srv/git;
			client_max_body_size 0;
			include fastcgi_params;
			fastcgi_param SCRIPT_FILENAME /usr/lib/git-core/git-http-backend;
			fastcgi_param GIT_HTTP_EXPORT_ALL "";
			fastcgi_param GIT_PROJECT_ROOT /srv/git;
			fastcgi_param PATH_INFO $uri;
			fastcgi_pass unix:/run/fcgiwrap.sock;
		}
	}
}
NGINXCONF
docker build -q -t "$GITSERVER_IMAGE" "$GITSERVER_BUILD_DIR" >/dev/null
kind load docker-image "$GITSERVER_IMAGE" --name "$CLUSTER_NAME"
docker rmi "$GITSERVER_IMAGE" >/dev/null 2>&1 || true
rm -rf "$GITSERVER_BUILD_DIR"

kubectl create namespace "$GITSERVER_NS" --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: d7s-gitserver
  namespace: $GITSERVER_NS
spec:
  replicas: 1
  selector:
    matchLabels: {app: d7s-gitserver}
  template:
    metadata:
      labels: {app: d7s-gitserver}
    spec:
      containers:
        - name: gitserver
          image: $GITSERVER_IMAGE
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: d7s-gitserver
  namespace: $GITSERVER_NS
spec:
  selector: {app: d7s-gitserver}
  ports:
    - port: 80
      targetPort: 80
EOF
kubectl wait --for=condition=Available deployment/d7s-gitserver -n "$GITSERVER_NS" --timeout="$TIMEOUT"
GITSERVER_POD=$(kubectl get pod -n "$GITSERVER_NS" -l app=d7s-gitserver -o jsonpath='{.items[0].metadata.name}')

log "git source: git add out/ && git commit  # the plan is the git diff"
GITCONTENT_DIR="$(mktemp -d)"
mkdir -p "$GITCONTENT_DIR/repo/out"
cp -r "$OUT"/. "$GITCONTENT_DIR/repo/out/"
git -C "$GITCONTENT_DIR/repo" -c init.defaultBranch=main init -q
git -C "$GITCONTENT_DIR/repo" add -A
git -C "$GITCONTENT_DIR/repo" -c user.email=harness@d7s.dev -c user.name=d7s-harness \
	commit -q -m "compiled output (acceptance harness run)"
# kubectl cp's tar-based extraction lands at
# <dest-dirname>/<src-basename> — naming the local bare clone "d7s.git"
# to match the target path exactly (no trailing-slash "copy contents"
# ambiguity to get wrong).
git clone --bare -q "$GITCONTENT_DIR/repo" "$GITCONTENT_DIR/d7s.git"
kubectl cp "$GITCONTENT_DIR/d7s.git" "$GITSERVER_NS/$GITSERVER_POD:/srv/git/d7s.git"
rm -rf "$GITCONTENT_DIR"

log "flux: git push  # register the git source, named/namespaced for the emitter's sourceRef"
kubectl apply -f - <<EOF
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: d7s
  namespace: flux-system
spec:
  interval: 1m
  url: http://d7s-gitserver.$GITSERVER_NS.svc.cluster.local/d7s.git
  ref:
    branch: main
EOF
poll "GitRepository d7s ready (flux cloned the git source)" bash -c \
	"flux reconcile source git d7s -n flux-system >/dev/null 2>&1; \
	 [ \"\$(kubectl get gitrepository d7s -n flux-system -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}' 2>/dev/null)\" = 'True' ]"

log "apply ONLY the emitted Flux Kustomization CRs — kustomize-controller/helm-controller reconcile the rest"
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
echo " forced here now that CNPG's CRDs actually exist — rule 44: this poll is"
echo " the wait, no fixed-duration sleep for that gap)"
poll "Kustomization week-one reconciled by flux" bash -c \
	"flux reconcile kustomization week-one -n flux-system --with-source >/dev/null 2>&1; \
	 [ \"\$(kubectl get kustomization week-one -n flux-system -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}' 2>/dev/null)\" = 'True' ]"

log "environment prerequisite: create the declared credentials secret"
echo "(CNPG's bootstrap.initdb.secret only consumes a pre-existing secret —"
echo " see internal/domain/secret.go; the week-one namespace already exists,"
echo " created by flux reconciling the Kustomization above)"
kubectl create secret generic orders-db-app -n week-one \
	--from-literal=username=orders-db \
	--from-literal=password="$(openssl rand -hex 16)" \
	--dry-run=client -o yaml | kubectl apply -f -

log "wait: Cluster reaches healthy state"
poll "Cluster healthy" bash -c \
	"[ \"\$(kubectl get cluster orders-db -n week-one -o jsonpath='{.status.phase}' 2>/dev/null)\" = 'Cluster in healthy state' ]"

log "guard: every compiled object was delivered by flux, never by a direct apply"
assert_flux_managed() {
	# fails loudly (rule 35: remedy in the message) if a compiled object
	# was never touched by kustomize-controller, or was ALSO touched by
	# a kubectl field manager — either means something other than flux
	# reconciliation of the git source put it there.
	local resource="$1" name="$2" ns="$3"
	local -a get_args=(get "$resource" "$name" -o jsonpath={.metadata.managedFields[*].manager})
	[ -n "$ns" ] && get_args+=(-n "$ns")
	local managers
	managers="$(kubectl "${get_args[@]}")"
	case " $managers " in
	*" kustomize-controller "*) ;;
	*)
		echo "guard failed: $resource/$name (ns=${ns:-<cluster-scoped>}) was never" \
			"reconciled by kustomize-controller (managers seen: $managers) —" \
			"remedy: apply only the emitted Flux Kustomization CRs and let flux" \
			"reconcile the compiled objects, never kubectl apply them directly" >&2
		exit 1
		;;
	esac
	case " $managers " in
	*" kubectl-client-side-apply "*|*" kubectl-edit "*|*" kubectl "*)
		echo "guard failed: $resource/$name (ns=${ns:-<cluster-scoped>}) was ALSO" \
			"touched by a kubectl field manager (managers seen: $managers) —" \
			"remedy: a compiled object must reach the cluster only through flux" \
			"reconciliation of the git source, never a direct kubectl apply" >&2
		exit 1
		;;
	esac
}
assert_flux_managed namespace cnpg-system ""
assert_flux_managed helmrepositories.source.toolkit.fluxcd.io cloudnative-pg cnpg-system
assert_flux_managed helmreleases.helm.toolkit.fluxcd.io cnpg-operator cnpg-system
assert_flux_managed namespace week-one ""
assert_flux_managed clusters.postgresql.cnpg.io orders-db week-one
assert_flux_managed peerauthentications.security.istio.io default week-one
assert_flux_managed authorizationpolicies.security.istio.io orders-db week-one

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

log "acceptance scenario PASSED"
