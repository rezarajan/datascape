# compile-and-verify - the documented `d7s compile` invocation, plus the
# compile-time contract checks that don't need a cluster: determinism
# (rules 22, 45), the rpo-with-an-undeclared-destination refusal (week-
# three plan, slices 1+2: rpo now compiles CONDITIONAL once backupTo
# names a *declared* external - the negative probe here is an unresolved
# reference, not "any declared rpo"), and the mtls+managed refusal
# (rules 34, 35, 49). Produces $OUT from $STACK, consumed by the
# git-source and deliver actions later in whichever orchestrated run
# invoked it. EXPECT_FILE names the one artifact that run's stack must
# produce - self-hosted's Cluster CR by default, overridden to the
# managed placement's Terraform CR by the managed orchestrator - so this
# action stays shared between both scenarios rather than forked in two
# (docs/plans/02-week-two.md Revision 3, slice 6's DRY principle).
require_repo_root

EXPECT_FILE="${EXPECT_FILE:-$OUT/apps/week-one/orders-db-cluster.yaml}"

log "compile (deterministic, read-only) - the documented invocation"
go build -o /tmp/d7s ./cmd/d7s
/tmp/d7s compile "$STACK" -o "$OUT"
test -f "$EXPECT_FILE"

log "determinism: two compiles are byte-identical (rules 22, 45)"
/tmp/d7s compile "$STACK" -o /tmp/d7s-out-2
diff -rq "$OUT" /tmp/d7s-out-2
rm -rf /tmp/d7s-out-2

log "negative probe: rpo.backupTo naming an undeclared external refuses to compile (rules 34, 35, 49)"
cat >/tmp/d7s-bad-rpo.yaml <<'EOF'
apiVersion: d7s.dev/v1alpha1
kind: Stack
name: bad-rpo
components:
  - kind: postgres
    name: db
    placement: self-hosted
    credentials:
      secretRef:
        name: db-app
    guarantees:
      rpo:
        target: 1h
        backupTo: nonexistent
EOF
if /tmp/d7s compile /tmp/d7s-bad-rpo.yaml -o /tmp/d7s-bad-rpo-out 2>/tmp/d7s-bad-rpo.err; then
	echo "expected compilation to refuse rpo.backupTo naming an undeclared external" >&2
	exit 1
fi
grep -q 'references undeclared external "nonexistent"' /tmp/d7s-bad-rpo.err
rm -f /tmp/d7s-bad-rpo.yaml /tmp/d7s-bad-rpo.err
rm -rf /tmp/d7s-bad-rpo-out

log "negative probe: guarantees.mtls + placement: managed refuses to compile (week-two plan)"
cat >/tmp/d7s-bad-managed-mtls.yaml <<'EOF'
apiVersion: d7s.dev/v1alpha1
kind: Stack
name: bad-managed-mtls
components:
  - kind: postgres
    name: db
    placement: managed
    credentials:
      secretRef:
        name: db-app
    guarantees:
      mtls: {}
EOF
if /tmp/d7s compile /tmp/d7s-bad-managed-mtls.yaml -o /tmp/d7s-bad-managed-mtls-out 2>/tmp/d7s-bad-managed-mtls.err; then
	echo "expected compilation to refuse guarantees.mtls + placement: managed" >&2
	exit 1
fi
grep -q "guarantees.mtls + placement: managed refuses to compile" /tmp/d7s-bad-managed-mtls.err
rm -f /tmp/d7s-bad-managed-mtls.yaml /tmp/d7s-bad-managed-mtls.err
rm -rf /tmp/d7s-bad-managed-mtls-out
