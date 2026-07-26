# minio-secret - materialize the app-namespace credentials Secret
# examples/week-one/stack.yaml's declared external names
# (credentials.secretRef.name: backups-credentials), from the MinIO root
# credentials minio-install generated and stored in MinIO's OWN
# namespace. Invoked from deliver.sh, once the app namespace ("week-one")
# exists (Flux reconciling the compiled Kustomization creates it) — the
# same per-run pattern, and the same reason, as neon-secret being invoked
# from deliver-managed.sh rather than folded into tofu-install: kubectl
# cannot create a Secret in a namespace that doesn't exist yet, and this
# namespace is d7s-compiled output, not a harness-created one.
#
# Keyed under internal/adapters/flux/durability.go's fixed convention
# (ACCESS_KEY_ID / ACCESS_SECRET_KEY) — the same keys minio-install
# already stored the values under, so this is a straight copy across
# namespaces, never a re-shape.
require_repo_root

log "backups-credentials secret: materialize in week-one from minio's root credentials"
ACCESS_KEY_ID=$(kubectl get secret "$MINIO_ROOT_CREDS_SECRET_NAME" -n "$MINIO_NS" \
	-o "jsonpath={.data.$OBJECT_STORE_ACCESS_KEY_ID_KEY}" | base64 -d)
SECRET_ACCESS_KEY=$(kubectl get secret "$MINIO_ROOT_CREDS_SECRET_NAME" -n "$MINIO_NS" \
	-o "jsonpath={.data.$OBJECT_STORE_SECRET_ACCESS_KEY_KEY}" | base64 -d)
kubectl create secret generic "$OBJECT_STORE_CREDENTIALS_SECRET_NAME" -n week-one \
	--from-literal="$OBJECT_STORE_ACCESS_KEY_ID_KEY=$ACCESS_KEY_ID" \
	--from-literal="$OBJECT_STORE_SECRET_ACCESS_KEY_KEY=$SECRET_ACCESS_KEY" \
	--dry-run=client -o yaml | kubectl apply -f -
