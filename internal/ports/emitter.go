// Package ports declares the interfaces the compiler core depends on.
// Ports import only domain — never adapters (golden rule 8).
package ports

import "github.com/rezarajan/datascape/internal/domain"

// Emitter is the compiler core's output boundary: one target-specific
// emitter per GitOps target. v1 has exactly one implementation (the Flux
// emitter), kept exactly this thin per golden rule 11 until a second
// target genuinely arrives.
type Emitter interface {
	Emit(stack domain.Stack) (Manifests, error)
}

// Manifests is the compiled, target-specific output an Emitter produces:
// a set of files keyed by their path relative to the compile output root,
// plus any conditional-guarantee notices the CLI must surface visibly.
type Manifests struct {
	Files map[string][]byte
	// Conditionals names every guarantee that compiled labeled
	// CONDITIONAL rather than unconditionally satisfied (problem
	// definition Amendment 2, B3: a durability/freshness-family
	// guarantee may compile across the trust boundary, but only labeled,
	// never silently). The label itself is also compiled directly into
	// Files (an annotation on the emitted objects) — this is the same
	// fact surfaced a second time, visibly, to whoever ran the compile.
	Conditionals []ConditionalGuarantee
}

// ConditionalGuarantee names one guarantee that compiled labeled
// CONDITIONAL, and why.
type ConditionalGuarantee struct {
	// Component is the declaring component's name.
	Component string
	// Guarantee names the guarantee family (e.g. "durability
	// (guarantees.rpo)").
	Guarantee string
	// Label is the exact annotation key: value compiled onto the
	// emitted objects for this guarantee.
	Label string
	// Reason states why the guarantee is only conditionally satisfied
	// (e.g. `crosses the trust boundary to external store "name"`).
	Reason string
}
