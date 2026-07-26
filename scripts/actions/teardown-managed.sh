# teardown-managed - the operator act golden rule 28 requires: the
# compiled Terraform CR never sets destroyResourcesOnDeletion (retain by
# default - internal/adapters/flux/terraform.go), so an ordinary delete
# of the CR would ABANDON the real Neon branch rather than destroy it. A
# leaked branch is a harness defect (week-two plan Revision 4), never
# just a cost concern. This action explicitly patches the CR to enable
# destruction before deleting it, waits (bounded, rule 44, on
# DESTROY_POLL_ATTEMPTS - see common.sh) for tofu-controller's destroy
# finalizer to actually finish BEFORE the cluster is ever torn down, then
# verifies via Neon's own API that no branch named after the harness-
# created component remains in the prerequisite project - never assumed
# from the CR's own deletion succeeding, which only proves the Kubernetes
# object is gone, not that the real branch was destroyed. Revision 4
# supersedes the earlier project-per-stack design (the owner's key is
# project-scoped and cannot create/delete projects at all - it CAN manage
# branches within its own project, which is exactly the operation this
# check performs).
#
# Every step below accumulates into $status rather than exiting early
# (found live, 2026-07-26: an early unguarded `poll` timeout on the CR-
# deletion wait tripped `set -e` and skipped BOTH the Neon-API leak check
# AND the `teardown` call at the very end, leaking the kind cluster
# itself). `teardown` (kind cluster deletion) always runs LAST, no matter
# what happened above it - the one part of this action that must never be
# skippable, and must never run before a pending destroy finalizer
# completes - and the accumulated status is what this action finally
# exits with, so a real problem still surfaces to the orchestrator/CI.
require_repo_root

status=0

# dump_runner_diagnostics - captures tofu-controller's own logs plus
# whatever runner pod exists in the stack namespace (found live,
# 2026-07-26: a bounded wait that times out silently is undiagnosable by
# design - this is what confirmed the destroy leg's real cost, a fresh
# runner pod cold-starting and re-running `tofu init`, rather than
# guessing).
dump_runner_diagnostics() {
	echo "--- kubectl logs deploy/tofu-controller -n flux-system --tail=60 ---"
	kubectl logs deploy/tofu-controller -n flux-system --tail=60 || true
	echo "--- kubectl get pods -n $MANAGED_NAMESPACE -o wide ---"
	kubectl get pods -n "$MANAGED_NAMESPACE" -o wide || true
	for pod in $(kubectl get pods -n "$MANAGED_NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
		echo "--- kubectl describe pod/$pod -n $MANAGED_NAMESPACE ---"
		kubectl describe pod "$pod" -n "$MANAGED_NAMESPACE" || true
		echo "--- kubectl logs pod/$pod -n $MANAGED_NAMESPACE --all-containers ---"
		kubectl logs "$pod" -n "$MANAGED_NAMESPACE" --all-containers --tail=300 || true
	done
}

if kubectl get terraform "$MANAGED_COMPONENT" -n "$MANAGED_NAMESPACE" >/dev/null 2>&1; then
	# Branch on whether the CR ever actually planned/applied anything
	# (found live, 2026-07-26: a CR Stalled before its first successful
	# reconcile - status.observedGeneration stays -1, never advancing
	# past the controller's initial state - has no Terraform state and
	# no real Neon resources to destroy; its own finalizer can't run a
	# destroy plan for a plan that never existed, which is exactly the
	# wedge that made the patch->delete->wait path hang until timeout).
	# A never-planned CR is removed directly (an explicit operator act,
	# safe only because nothing was ever created); a CR that DID plan/
	# apply gets the full destroy path - the Neon-API check further down
	# is the ground truth either way, not this branch's own guess.
	observed_gen=$(kubectl get terraform "$MANAGED_COMPONENT" -n "$MANAGED_NAMESPACE" -o jsonpath='{.status.observedGeneration}' 2>/dev/null || true)
	if [ "$observed_gen" = "-1" ] || [ -z "$observed_gen" ]; then
		log "Terraform $MANAGED_COMPONENT never reached a plan/apply (observedGeneration=$observed_gen) - nothing to destroy"
		echo "removing its finalizer directly is safe here: no Terraform state and no real Neon branch was ever created" >&2
		kubectl patch terraform "$MANAGED_COMPONENT" -n "$MANAGED_NAMESPACE" --type=merge -p '{"metadata":{"finalizers":[]}}' || true
		kubectl delete terraform "$MANAGED_COMPONENT" -n "$MANAGED_NAMESPACE" --ignore-not-found --wait=false
		if ! poll "Terraform $MANAGED_COMPONENT fully deleted (finalizer removed directly)" bash -c \
			"! kubectl get terraform '$MANAGED_COMPONENT' -n '$MANAGED_NAMESPACE' >/dev/null 2>&1"; then
			echo "teardown warning: Terraform $MANAGED_COMPONENT still present after its finalizer was removed - unexpected, investigate; the cluster is still torn down regardless" >&2
			status=1
		fi
	else
		log "operator act: enable destroyResourcesOnDeletion on the Terraform CR before deleting it"
		kubectl patch terraform "$MANAGED_COMPONENT" -n "$MANAGED_NAMESPACE" --type=merge \
			-p '{"spec":{"destroyResourcesOnDeletion":true}}'

		log "delete the Terraform CR - tofu-controller's finalizer runs a destroy plan first"
		kubectl delete terraform "$MANAGED_COMPONENT" -n "$MANAGED_NAMESPACE" --ignore-not-found --wait=false
		if ! poll_n "$DESTROY_POLL_ATTEMPTS" "$POLL_INTERVAL" \
			"Terraform $MANAGED_COMPONENT fully deleted (destroy finalizer completed)" bash -c \
			"! kubectl get terraform '$MANAGED_COMPONENT' -n '$MANAGED_NAMESPACE' >/dev/null 2>&1"; then
			log "DIAGNOSTICS: destroy finalizer did not complete within DESTROY_POLL_ATTEMPTS - capturing state"
			dump_runner_diagnostics
			echo "teardown warning: Terraform $MANAGED_COMPONENT's destroy finalizer did not complete in time - the Kubernetes object may be stuck; the Neon-API check below still runs to catch a real leaked branch, and the cluster is still torn down regardless (never before this point, so the finalizer always got its full budget first)" >&2
			status=1
		fi
	fi
else
	log "no Terraform CR found in $MANAGED_NAMESPACE - nothing to destroy (an earlier step never reached delivery)"
fi

if [ -n "${NEON_API_KEY:-}" ]; then
	log "verify via Neon's API: the harness-created branch is gone"
	if [ -z "${NEON_PROJECT_ID:-}" ]; then
		NEON_PROJECT_ID=$(discover_neon_project_id || true)
	fi
	if [ -n "${NEON_PROJECT_ID:-}" ]; then
		branches_json=$(curl -sf -m 30 "https://console.neon.tech/api/v2/projects/$NEON_PROJECT_ID/branches" \
			-H "Authorization: Bearer $NEON_API_KEY" || true)
		if [ -z "$branches_json" ]; then
			echo "teardown warning: could not list Neon branches for project $NEON_PROJECT_ID (API call failed) - cannot confirm no leak" >&2
			status=1
		else
			remaining=$(printf '%s' "$branches_json" | jq -r --arg name "$MANAGED_COMPONENT" '.branches[] | select(.name==$name) | .id')
			if [ -n "$remaining" ]; then
				echo "teardown defect: Neon branch '$MANAGED_COMPONENT' (id: $remaining) still exists in project $NEON_PROJECT_ID after teardown" >&2
				echo "last-resort remediation: deleting it via the Neon API directly - self-cleaning, never self-absolving (this run still fails)" >&2
				delete_http_code=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE \
					"https://console.neon.tech/api/v2/projects/$NEON_PROJECT_ID/branches/$remaining" \
					-H "Authorization: Bearer $NEON_API_KEY" || true)
				if [ "$delete_http_code" = "200" ] || [ "$delete_http_code" = "202" ]; then
					echo "last-resort remediation succeeded: branch $remaining deleted via the API (http $delete_http_code) - the defect above is still reported, investigate why the destroy finalizer did not remove it" >&2
				else
					echo "last-resort remediation FAILED: DELETE returned http $delete_http_code - branch $remaining may still be leaked, remedy: delete it manually via the Neon console/API" >&2
				fi
				status=1
			else
				log "confirmed: no leaked Neon branch named $MANAGED_COMPONENT remains in project $NEON_PROJECT_ID"
			fi
		fi
	else
		echo "could not resolve NEON_PROJECT_ID - skipping the Neon-API leak check" >&2
	fi
else
	echo "NEON_API_KEY not set in this teardown's environment - skipping the Neon-API leak check (nothing to verify against)" >&2
fi

teardown

exit "$status"
