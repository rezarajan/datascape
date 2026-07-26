# compile-managed - compile examples/week-two/managed-stack.yaml to
# $MANAGED_OUT. Delegates the actual compile+determinism+refusal-check
# work to compile-and-verify (docs/plans/02-week-two.md Revision 3, slice
# 6's DRY principle: the negative probes are stack-agnostic compiler
# contract tests, not specific to the managed scenario) - EXPECT_FILE
# names the one artifact THIS stack must produce, the Terraform CR,
# rather than the self-hosted scenario's default (its Cluster CR).
require_repo_root

EXPECT_FILE="$MANAGED_OUT/apps/$MANAGED_NAMESPACE/orders-db-terraform.yaml" \
	STACK="$MANAGED_STACK" \
	OUT="$MANAGED_OUT" \
	compile-and-verify
