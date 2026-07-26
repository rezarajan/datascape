# durability-probe - the RPO/durability guarantee's conformance probe
# (week-three plan, slice 3): the leg that has failed by design since
# week one — a CNPG Backup object actually reaching phase `completed`
# against a real, declared external store — goes green here. Existence
# alone was never the claim (Cluster.spec.backup + the ScheduledBackup
# CR have compiled since week-three slices 1+2); completion is the part
# that needed a real MinIO to prove.
#
# Also asserts the CONDITIONAL label (problem definition Amendment 2,
# B3) is visible on the live, delivered Cluster and ScheduledBackup — not
# only in the golden-file bytes (internal/adapters/flux/flux_test.go).
# This half of the probe can fail (golden rule 49): presence is the only
# signal, and there is nothing stopping a future emitter change from
# quietly dropping the annotation while everything else still compiles.
require_repo_root

log "probe: compiled Cluster/ScheduledBackup carry the CONDITIONAL durability label (rule 49 - this check can fail)"
cluster_label=$(kubectl get cluster orders-db -n week-one -o jsonpath='{.metadata.annotations.d7s\.dev/guarantee-durability}')
if [ "$cluster_label" != "conditional-on-external" ]; then
	echo "expected Cluster orders-db to carry annotation d7s.dev/guarantee-durability=conditional-on-external, got: ${cluster_label:-<absent>}" >&2
	exit 1
fi
sb_label=$(kubectl get scheduledbackups.postgresql.cnpg.io orders-db -n week-one -o jsonpath='{.metadata.annotations.d7s\.dev/guarantee-durability}')
if [ "$sb_label" != "conditional-on-external" ]; then
	echo "expected ScheduledBackup orders-db to carry annotation d7s.dev/guarantee-durability=conditional-on-external, got: ${sb_label:-<absent>}" >&2
	exit 1
fi

log "probe: a CNPG Backup reaches phase completed (the leg that has failed since week one)"
if ! poll "CNPG Backup completed against the declared external store" bash -c \
	"kubectl get backups.postgresql.cnpg.io -n week-one -o jsonpath='{.items[*].status.phase}' 2>/dev/null | grep -qw completed"; then
	log "DIAGNOSTICS: no CNPG Backup reached phase completed - capturing state before teardown"
	echo "--- kubectl get backups.postgresql.cnpg.io -n week-one -o yaml ---"
	kubectl get backups.postgresql.cnpg.io -n week-one -o yaml || true
	echo "--- kubectl describe backups.postgresql.cnpg.io -n week-one ---"
	kubectl describe backups.postgresql.cnpg.io -n week-one || true
	echo "--- kubectl describe cluster orders-db -n week-one ---"
	kubectl describe cluster orders-db -n week-one || true
	echo "--- cnpg operator logs (cnpg-system) ---"
	for pod in $(kubectl get pods -n cnpg-system -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
		echo "--- kubectl logs pod/$pod -n cnpg-system --all-containers ---"
		kubectl logs "$pod" -n cnpg-system --all-containers --tail=300 || true
	done
	exit 1
fi
