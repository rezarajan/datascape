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
// a set of files keyed by their path relative to the compile output root.
type Manifests struct {
	Files map[string][]byte
}
