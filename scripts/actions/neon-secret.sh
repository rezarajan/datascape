# neon-secret - materialize the neon-api-key Kubernetes Secret the
# compiled Terraform CR's runner pod reads NEON_API_KEY and the Neon
# project id from (week-two plan Revision 3: "the harness materializes
# the neon-api-key Kubernetes Secret from it at runtime, mirroring how
# the CNPG credentials secret is already created per run"; Revision 4:
# the project id joins the same Secret, since it reaches the OpenTofu
# config via the same varsFrom trust path as the key). Assumes
# $MANAGED_NAMESPACE already exists (the managed Kustomization's
# namespace object, applied and reconciled before this runs) - kubectl
# refuses to create a secret in a namespace that doesn't exist yet.
#
# require_neon_api_key does the actual API-key process: NEON_API_KEY from
# the environment, falling back to a gitignored ./.env, never printed -
# and exits with SKIP_EXIT_CODE (never 0 or 1) if neither is set, so a
# caller further up (the managed orchestrator) can tell "skipped" apart
# from "failed" by exit code alone. discover_neon_project_id resolves
# NEON_PROJECT_ID the same way if it isn't already set (an explicit env/
# .env override always wins over discovery).
require_repo_root
require_neon_api_key

if [ -z "${NEON_PROJECT_ID:-}" ]; then
	log "discover the Neon project id NEON_API_KEY is scoped to"
	NEON_PROJECT_ID=$(discover_neon_project_id)
fi

log "neon-api-key secret: materialize in $MANAGED_NAMESPACE (apiKey + projectId; values never logged)"
kubectl create secret generic "$NEON_API_KEY_SECRET_NAME" -n "$MANAGED_NAMESPACE" \
	--from-literal="$NEON_API_KEY_SECRET_KEY=$NEON_API_KEY" \
	--from-literal="$NEON_PROJECT_ID_SECRET_KEY=$NEON_PROJECT_ID" \
	--dry-run=client -o yaml | kubectl apply -f -
