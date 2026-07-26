# minio-install - the acceptance stack's declared `external` object
# store, stood up by the harness in its own namespace exactly like Flux
# or Istio above: environment scaffolding d7s never provisions or
# mutates (problem definition Amendment 2 — external by provenance; week-
# three plan, slice 3). Plain HTTP, no TLS cert to manage — an
# in-cluster-only Service nothing outside the cluster ever reaches, and
# examples/week-one/stack.yaml's declared external names it the same way
# (http://minio.d7s-harness-minio.svc:9000).
#
# Only stands up MinIO itself and records its generated root credentials
# in ITS OWN namespace's Secret. It deliberately does NOT also
# materialize the app-namespace credentials Secret the declared
# external's credentials.secretRef names: `kubectl create secret -n
# week-one` would fail outright here since Flux hasn't reconciled that
# namespace into existence yet (delivery happens later in the
# orchestrator) — the identical timing constraint neon-secret/
# deliver-managed already solved for the managed scenario's credentials
# secret. scripts/actions/minio-secret.sh is that second step, invoked
# from deliver.sh once the app namespace exists (see its own comment,
# and deliver.sh's "environment prerequisite" block).
require_repo_root

log "minio: namespace $MINIO_NS"
kubectl create namespace "$MINIO_NS" --dry-run=client -o yaml | kubectl apply -f -

log "minio: root credentials (generated once per run, never logged)"
MINIO_ROOT_USER="d7s-minio-$(openssl rand -hex 4)"
MINIO_ROOT_PASSWORD="$(openssl rand -hex 16)"
kubectl create secret generic "$MINIO_ROOT_CREDS_SECRET_NAME" -n "$MINIO_NS" \
	--from-literal="$OBJECT_STORE_ACCESS_KEY_ID_KEY=$MINIO_ROOT_USER" \
	--from-literal="$OBJECT_STORE_SECRET_ACCESS_KEY_KEY=$MINIO_ROOT_PASSWORD" \
	--dry-run=client -o yaml | kubectl apply -f -

log "minio: deployment + service"
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: minio
  namespace: $MINIO_NS
spec:
  replicas: 1
  selector:
    matchLabels: {app: minio}
  template:
    metadata:
      labels: {app: minio}
    spec:
      containers:
        - name: minio
          image: $MINIO_IMAGE
          args: ["server", "/data"]
          env:
            - name: MINIO_ROOT_USER
              valueFrom:
                secretKeyRef:
                  name: $MINIO_ROOT_CREDS_SECRET_NAME
                  key: $OBJECT_STORE_ACCESS_KEY_ID_KEY
            - name: MINIO_ROOT_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: $MINIO_ROOT_CREDS_SECRET_NAME
                  key: $OBJECT_STORE_SECRET_ACCESS_KEY_KEY
          ports:
            - containerPort: 9000
          readinessProbe:
            httpGet: {path: /minio/health/ready, port: 9000}
            initialDelaySeconds: 2
            periodSeconds: 3
---
apiVersion: v1
kind: Service
metadata:
  name: $MINIO_SERVICE
  namespace: $MINIO_NS
spec:
  selector: {app: minio}
  ports:
    - port: 9000
      targetPort: 9000
EOF
kubectl wait --for=condition=Available deployment/minio -n "$MINIO_NS" --timeout="$TIMEOUT"

log "minio: create the declared bucket ($MINIO_BUCKET) via a one-shot mc job"
kubectl delete job minio-mc-mb -n "$MINIO_NS" --ignore-not-found >/dev/null 2>&1
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: minio-mc-mb
  namespace: $MINIO_NS
spec:
  backoffLimit: 6
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: mc
          image: $MC_IMAGE
          env:
            - name: MINIO_ROOT_USER
              valueFrom:
                secretKeyRef:
                  name: $MINIO_ROOT_CREDS_SECRET_NAME
                  key: $OBJECT_STORE_ACCESS_KEY_ID_KEY
            - name: MINIO_ROOT_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: $MINIO_ROOT_CREDS_SECRET_NAME
                  key: $OBJECT_STORE_SECRET_ACCESS_KEY_KEY
            - name: MC_HOST_minio
              value: "http://\$(MINIO_ROOT_USER):\$(MINIO_ROOT_PASSWORD)@$MINIO_SERVICE.$MINIO_NS.svc.cluster.local:9000"
          command: ["mc", "mb", "--ignore-existing", "minio/$MINIO_BUCKET"]
EOF

if ! poll "minio bucket $MINIO_BUCKET created (mc job completed)" bash -c \
	"[ \"\$(kubectl get job minio-mc-mb -n $MINIO_NS -o jsonpath='{.status.succeeded}' 2>/dev/null)\" = '1' ]"; then
	log "DIAGNOSTICS: minio-mc-mb did not complete - capturing state before failing"
	kubectl describe job minio-mc-mb -n "$MINIO_NS" || true
	kubectl logs job/minio-mc-mb -n "$MINIO_NS" --all-containers --tail=200 || true
	exit 1
fi
