# probe-managed - the managed seam's conformance probe (rule 41): an
# in-cluster pod connects to the real Neon endpoint over TLS
# (sslmode=require) and runs SQL, using ONLY the written-outputs secret
# (the component's DECLARED credentials.secretRef.name) - never a value
# d7s compiled or this script hardcoded. envFrom maps the secret's data
# keys (host, port, database, username, password - see
# internal/adapters/flux/terraform.go's neonConfigTemplate output blocks)
# straight to identically-named environment variables in the container.
require_repo_root

# The declared consumer's OWN ServiceAccount (allowedConsumers:
# probe-client in the pinned declaration) - consumer-owned identity, not
# a d7s-compiled object, so the probe creates it exactly as a real
# consumer team would. The compiled AuthorizationPolicy on the exact-host
# ServiceEntry admits only this principal; the negative leg below proves
# the denial for everyone else (rule 49: the check can fail).
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: probe-client
  namespace: $MANAGED_NAMESPACE
EOF

log "positive probe: declared consumer connects to Neon over TLS using ONLY the written-outputs secret, runs SQL"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: managed-probe-client
  namespace: $MANAGED_NAMESPACE
spec:
  restartPolicy: Never
  serviceAccountName: probe-client
  containers:
    - name: psql
      image: postgres:17
      command: ["sleep", "3600"]
      envFrom:
        - secretRef:
            name: $MANAGED_CREDENTIALS_SECRET
EOF
kubectl wait --for=condition=Ready pod/managed-probe-client -n "$MANAGED_NAMESPACE" --timeout="$TIMEOUT"
# shellcheck disable=SC2016
kubectl exec -n "$MANAGED_NAMESPACE" managed-probe-client -- sh -c \
	'psql "host=$host port=$port dbname=$database user=$username password=$password sslmode=require" -c "SELECT 1;"'

log "negative probe: an UNDECLARED identity is refused at the compiled egress gate (rule 49)"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: managed-probe-undeclared
  namespace: $MANAGED_NAMESPACE
spec:
  restartPolicy: Never
  containers:
    - name: psql
      image: postgres:17
      command: ["sleep", "3600"]
      envFrom:
        - secretRef:
            name: $MANAGED_CREDENTIALS_SECRET
EOF
kubectl wait --for=condition=Ready pod/managed-probe-undeclared -n "$MANAGED_NAMESPACE" --timeout="$TIMEOUT"
# The default-ServiceAccount pod carries the SAME credentials - only its
# identity differs - so a successful SELECT here would mean the compiled
# allow-list is decorative. connect_timeout keeps the denied path from
# consuming psql's own much longer default.
# shellcheck disable=SC2016
if kubectl exec -n "$MANAGED_NAMESPACE" managed-probe-undeclared -- sh -c \
	'psql "host=$host port=$port dbname=$database user=$username password=$password sslmode=require connect_timeout=15" -c "SELECT 1;"' 2>/dev/null; then
	echo "NEGATIVE PROBE FAILED OPEN: an undeclared identity reached the managed endpoint - the compiled egress gate is not enforcing" >&2
	exit 1
fi
log "confirmed: undeclared identity REFUSED at the compiled egress gate"
