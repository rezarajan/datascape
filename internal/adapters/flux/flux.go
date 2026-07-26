// Package flux is the Flux target emitter adapter: it compiles a Stack
// into Flux-consumable Kubernetes manifests. It implements ports.Emitter
// and imports only domain and ports (golden rule 8).
package flux

import (
	"github.com/rezarajan/datascape/internal/domain"
	"github.com/rezarajan/datascape/internal/ports"
)

// Emitter compiles a Stack to Flux manifests.
type Emitter struct{}

// New builds a Flux Emitter.
func New() *Emitter {
	return &Emitter{}
}

// Emit implements ports.Emitter. The manifest set is filled in as
// declaration coverage grows (namespace, CNPG operator, Cluster CR, ...).
func (e *Emitter) Emit(stack domain.Stack) (ports.Manifests, error) {
	return ports.Manifests{Files: map[string][]byte{}}, nil
}
