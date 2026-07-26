// Package compiler is the compiler core (golden rule 9's named plane):
// it validates a declaration and drives a target emitter. Cross-component
// wiring logic lives here, never precipitated into an emitter adapter —
// though week one has exactly one component kind, so there is no
// cross-component wiring to exercise yet (a known, deliberate gap, not
// an implicit promise — golden rule 7).
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
