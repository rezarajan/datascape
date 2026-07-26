// Package compiler is the compiler core (golden rule 9's named plane):
// it validates a declaration and drives a target emitter. Week-three plan
// slices 1+2 introduced the first real cross-declaration reference
// (Postgres.Guarantees.RPO.BackupTo naming a sibling External) — its
// referential validity (does the name resolve at all?) is checked in
// Stack.Validate (internal/domain/stack.go), the declaration model's own
// aggregator, since that check is target-agnostic; resolving a validated
// reference into a target's actual emitted shape (the CNPG
// barmanObjectStore fields) is target-specific and stays in the Flux
// emitter (internal/adapters/flux/durability.go) — never precipitated
// into this package, which still does no target-specific shaping of its
// own.
package compiler

import (
	"errors"

	"github.com/rezarajan/datascape/internal/domain"
	"github.com/rezarajan/datascape/internal/ports"
)

// Compiler compiles a Stack by driving the configured target Emitter.
type Compiler struct {
	Emitter ports.Emitter
}

// New builds a Compiler over the given target emitter. Only the
// composition root selects which concrete emitter to pass in.
func New(emitter ports.Emitter) *Compiler {
	return &Compiler{Emitter: emitter}
}

// Compile validates stack, aggregating every problem into one report
// before emitting anything (golden rule 33), then compiles it via the
// target emitter.
func (c *Compiler) Compile(stack domain.Stack) (ports.Manifests, error) {
	if errs := stack.Validate(); len(errs) > 0 {
		return ports.Manifests{}, errors.Join(errs...)
	}
	return c.Emitter.Emit(stack)
}
