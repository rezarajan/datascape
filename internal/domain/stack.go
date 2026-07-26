// Package domain holds the declaration model: pure data and validation
// logic. It imports nothing outside the standard library — ports and
// adapters depend on domain, never the reverse (golden rule 8).
package domain

// Stack is the root of the declaration model: a named data platform to
// compile.
type Stack struct {
	Name       string
	Components []Component
}

// Component is implemented by every declarable component kind (e.g.
// Postgres).
type Component interface {
	ComponentName() string
}
