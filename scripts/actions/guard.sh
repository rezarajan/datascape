# guard - the full seven-object managedFields assertion: every compiled
# object must have been delivered by flux (kustomize-controller), never
# by a direct kubectl apply. Fails loudly with the remedy (rules 34, 35)
# if either check fails for any of the seven objects.
require_repo_root

log "guard: every compiled object was delivered by flux, never by a direct apply"
assert_flux_managed() {
	# fails loudly (rule 35: remedy in the message) if a compiled object
	# was never touched by kustomize-controller, or was ALSO touched by
	# a kubectl field manager - either means something other than flux
	# reconciliation of the git source put it there.
	local resource="$1" name="$2" ns="$3"
	local -a get_args=(get "$resource" "$name" -o "jsonpath={.metadata.managedFields[*].manager}")
	[ -n "$ns" ] && get_args+=(-n "$ns")
	local managers
	managers="$(kubectl "${get_args[@]}")"
	case " $managers " in
	*" kustomize-controller "*) ;;
	*)
		echo "guard failed: $resource/$name (ns=${ns:-<cluster-scoped>}) was never" \
			"reconciled by kustomize-controller (managers seen: $managers) - " \
			"remedy: apply only the emitted Flux Kustomization CRs and let flux" \
			"reconcile the compiled objects, never kubectl apply them directly" >&2
		exit 1
		;;
	esac
	case " $managers " in
	*" kubectl-client-side-apply "*|*" kubectl-edit "*|*" kubectl "*)
		echo "guard failed: $resource/$name (ns=${ns:-<cluster-scoped>}) was ALSO" \
			"touched by a kubectl field manager (managers seen: $managers) - " \
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
