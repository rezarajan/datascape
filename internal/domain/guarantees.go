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
	// honors it, plus a wired backup destination (week-three plan,
	// slices 1+2: the destination is now declarable). BackupTo names a
	// declared external object store (problem definition Amendment 2) —
	// required, since v1 has exactly one way to declare a destination.
	// A durability guarantee wired to a declared external compiles, but
	// only labeled CONDITIONAL (Amendment 2, B3: it crosses the trust
	// boundary, so the claim is only as strong as the external's own
	// gate) — the emitter carries the label into every object it emits
	// for it (internal/adapters/flux/durability.go). placement: managed
	// still refuses (internal/domain/postgres.go): the managed emitter
	// has no destination wiring this week.
	RPO *RPOGuarantee
}

// MTLSGuarantee is an empty marker: its presence in the declaration turns
// the guarantee on. It carries no fields, so the schema has no way to
// express "mTLS disabled" once declared.
type MTLSGuarantee struct{}

// RPOGuarantee is a recovery point objective: after a failure, data loss
// must never exceed Target, backed up to the external object store named
// by BackupTo.
type RPOGuarantee struct {
	Target time.Duration
	// BackupTo names a declared external (Stack.Externals) this
	// guarantee's backups are written to. Required — v1 has exactly one
	// destination shape declarable (an S3-compatible external object
	// store) and no default. Whether the name actually resolves to a
	// declared external is a stack-level, not a per-guarantee,
	// question — checked in Stack.Validate (internal/domain/stack.go),
	// since a lone Postgres component has no visibility into its
	// siblings' declarations.
	BackupTo string
}

// Validate reports every structural problem with g, aggregated rather
// than stopping at the first (golden rule 33). guarantees.rpo refuses
// here only when no destination is named — BackupTo is required because
// v1 has exactly one declarable destination shape and no default (golden
// rules 34, 35). Whether BackupTo actually names a declared external is
// checked at the stack level (Stack.Validate), not here.
func (g Guarantees) Validate(component string) []error {
	var errs []error
	if g.RPO != nil && g.RPO.BackupTo == "" {
		errs = append(errs, fmt.Errorf(
			"postgres component %q: guarantees.rpo requires backupTo naming a declared external "+
				"object store — declare an external block (endpoint, bucket, credentials.secretRef) "+
				"and set guarantees.rpo.backupTo to its name, or remove guarantees.rpo",
			component))
	}
	return errs
}
