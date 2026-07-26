# probe-managed - the managed seam's conformance probe (rule 41): an
# in-cluster pod connects to the real Neon endpoint over TLS
# (sslmode=require) and runs SQL, using ONLY the written-outputs secret
# (the component's DECLARED credentials.secretRef.name) - never a value
# d7s compiled or this script hardcoded. envFrom maps the secret's data
# keys (host, port, database, username, password - see
# internal/adapters/flux/terraform.go's neonConfigTemplate output blocks)
# straight to identically-named environment variables in the container.
require_repo_root

log "positive probe: declared consumer connects to Neon over TLS using ONLY the written-outputs secret, runs SQL"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: managed-probe-client
  namespace: $MANAGED_NAMESPACE
spec:
  restartPolicy: Never
  containers:
    - name: psql
      image: postgres:17
      command: ["sleep", "3600"]
      envFrom:
        - secretRef:
            name: orders-db-app
EOF
kubectl wait --for=condition=Ready pod/managed-probe-client -n "$MANAGED_NAMESPACE" --timeout="$TIMEOUT"
# shellcheck disable=SC2016
kubectl exec -n "$MANAGED_NAMESPACE" managed-probe-client -- sh -c \
	'psql "host=$host port=$port dbname=$database user=$username password=$password sslmode=require" -c "SELECT 1;"'
