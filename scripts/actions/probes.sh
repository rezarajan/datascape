# probes - the positive mTLS conformance probe (a declared consumer
# connects and runs SQL, rule 41) and the negative off-mesh refusal probe
# (rule 49): an undeclared, off-mesh plaintext client must be refused.
require_repo_root

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
