# compile-and-verify - the documented `d7s compile` invocation, plus the
# two compile-time contract checks that don't need a cluster: determinism
# (rules 22, 45) and the any-RPO refusal (rules 34, 35, 49). Produces
# $OUT, consumed by the git-source and deliver actions later in the
# orchestrated run.
require_repo_root

log "compile (deterministic, read-only) - the documented invocation"
go build -o /tmp/d7s ./cmd/d7s
/tmp/d7s compile "$STACK" -o "$OUT"
test -f "$OUT/apps/week-one/orders-db-cluster.yaml"

log "determinism: two compiles are byte-identical (rules 22, 45)"
/tmp/d7s compile "$STACK" -o /tmp/d7s-out-2
diff -rq "$OUT" /tmp/d7s-out-2
rm -rf /tmp/d7s-out-2

log "negative probe: any declared RPO refuses to compile (rules 34, 35, 49)"
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
      rpo: 2m
EOF
if /tmp/d7s compile /tmp/d7s-bad-rpo.yaml -o /tmp/d7s-bad-rpo-out 2>/tmp/d7s-bad-rpo.err; then
	echo "expected compilation to refuse a declared RPO" >&2
	exit 1
fi
grep -q "durability guarantee's conformance probe could never pass" /tmp/d7s-bad-rpo.err
rm -f /tmp/d7s-bad-rpo.yaml /tmp/d7s-bad-rpo.err
rm -rf /tmp/d7s-bad-rpo-out
