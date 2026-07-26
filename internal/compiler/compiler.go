// Package compiler is the compiler core (golden rule 9's named plane):
// it drives a target emitter over a validated declaration. Cross-component
// wiring logic lives here, never precipitated into an emitter adapter.
package compiler

import (
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

// Compile validates and compiles stack, returning the target's manifests.
func (c *Compiler) Compile(stack domain.Stack) (ports.Manifests, error) {
	return c.Emitter.Emit(stack)
}
