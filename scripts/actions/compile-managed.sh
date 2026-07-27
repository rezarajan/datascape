# compile-managed - compile the managed scenario's PHASE-ONE (unpinned)
# declaration, $MANAGED_STACK_UNPINNED, to $MANAGED_OUT. Phase one is
# unpinned by construction: the endpoint host tofu-controller will
# provision does not exist yet, and d7s never fabricates one (rule 50) -
# pin-managed materializes phase two from $MANAGED_STACK once the
# written-outputs secret carries the live host. Delegates the actual
# compile+determinism+refusal-check work to compile-and-verify
# (docs/plans/02-week-two.md Revision 3, slice 6's DRY principle: the
# negative probes are stack-agnostic compiler contract tests, not
# specific to the managed scenario) - EXPECT_FILE names the one artifact
# THIS stack must produce, the Terraform CR, rather than the self-hosted
# scenario's default (its Cluster CR).
require_repo_root

EXPECT_FILE="$MANAGED_OUT/apps/$MANAGED_NAMESPACE/$MANAGED_COMPONENT-terraform.yaml" \
	STACK="$MANAGED_STACK_UNPINNED" \
	OUT="$MANAGED_OUT" \
	compile-and-verify
