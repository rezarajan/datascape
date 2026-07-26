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
	// honors it. A target the compiler cannot honor fails compilation
	// with the remedy in the error (golden rules 34/35) — enforced by
	// the compiler core, not here.
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
// than stopping at the first (golden rule 33).
func (g Guarantees) Validate(component string) []error {
	var errs []error
	if g.RPO != nil && g.RPO.Target <= 0 {
		errs = append(errs, fmt.Errorf(
			"postgres component %q: guarantees.rpo must be a positive duration, got %q — declare e.g. \"1h\" or \"15m\"",
			component, g.RPO.Target))
	}
	return errs
}
