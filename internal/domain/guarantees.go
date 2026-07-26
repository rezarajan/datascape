package domain

import (
	"fmt"
	"time"
)

// Guarantees declares the guarantee families a component must meet.
// Every declared guarantee ships as a triple — compile-time check,
// emitted infrastructure, conformance probe — or not at all; there is no
// best-effort tier (golden rules 34, 37, 50; problem definition
// Amendment 1, "the guarantee primitive").
type Guarantees struct {
	// MTLS turns on the transport-security guarantee: mesh mTLS
	// (PeerAuthentication STRICT) plus a default-deny AuthorizationPolicy
	// compiled only from declared wiring (golden rule 53). Presence is
	// the only signal the schema exposes — there is no field to turn it
	// off once declared (golden rule 50: security properties have no
	// best-effort tier).
	MTLS *MTLSGuarantee

	// RPO declares the durability/recovery guarantee: the maximum
	// acceptable window of data loss, compiled to a backup schedule that
	// honors it. v1 has no way to declare a backup destination, so this
	// guarantee fails compilation closed unconditionally, on every
	// placement (golden rules 34/35) — enforced in Validate below, not
	// the compiler core: the guarantee can never be satisfied regardless
	// of target, so there is nothing target-dependent for the compiler
	// core or an emitter to check on the live path. The emitter-level
	// satisfiability check and backup emitter remain in the tree, gated
	// behind this refusal, for the week a destination becomes declarable
	// (owner decision, week-one plan "Owner decisions — 2026-07-26");
	// verified this still fires for placement: managed too (week-two
	// plan) since this check does not branch on placement at all.
	RPO *RPOGuarantee
}

// MTLSGuarantee is an empty marker: its presence in the declaration turns
// the guarantee on. It carries no fields, so the schema has no way to
// express "mTLS disabled" once declared.
type MTLSGuarantee struct{}

// RPOGuarantee is a recovery point objective: after a failure, data loss
// must never exceed Target.
type RPOGuarantee struct {
	Target time.Duration
}

// Validate reports every structural problem with g, aggregated rather
// than stopping at the first (golden rule 33). guarantees.rpo refuses
// unconditionally here, on every placement (golden rules 34, 35): v1 has
// no way to declare a backup destination, so the durability guarantee's
// conformance probe could never pass, for any declared target. The gated
// emitter-level machinery (checkRPOSatisfiable, the ScheduledBackup emitter in
// internal/adapters/flux/durability.go) stays in the tree, unit-tested,
// for the week a destination becomes declarable — this refusal simply
// never lets the compile path reach it.
func (g Guarantees) Validate(component string) []error {
	var errs []error
	if g.RPO != nil {
		errs = append(errs, fmt.Errorf(
			"postgres component %q: guarantees.rpo is planned, not yet available — "+
				"v1 has no backup destination declarable, so the durability guarantee's "+
				"conformance probe could never pass; remove guarantees.rpo (a declarable "+
				"destination is planned for the week-two+ skeleton)",
			component))
	}
	return errs
}
