// Package domain holds the declaration model: pure data and validation
// logic. It imports nothing outside the standard library — ports and
// adapters depend on domain, never the reverse (golden rule 8).
package domain

import "fmt"

// Stack is the root of the declaration model: a named data platform to
// compile.
type Stack struct {
	Name       string
	Components []Component
}

// ComponentKind identifies which concrete component schema a Component is.
type ComponentKind string

// KindPostgres is the only component kind week one declares.
const KindPostgres ComponentKind = "postgres"

// Component is implemented by every declarable component kind (e.g.
// Postgres). A component that also implements validator contributes its
// own errors to Stack.Validate.
type Component interface {
	ComponentName() string
	Kind() ComponentKind
}

type validator interface {
	Validate() []error
}

// Validate aggregates every problem in the stack into one report rather
// than failing on the first (golden rule 33: validate-time completeness).
func (s Stack) Validate() []error {
	var errs []error
	if s.Name == "" {
		errs = append(errs, fmt.Errorf("stack: name is required"))
	}
	if len(s.Components) == 0 {
		errs = append(errs, fmt.Errorf("stack %q: at least one component is required", s.Name))
	}
	seen := make(map[string]bool, len(s.Components))
	for _, c := range s.Components {
		name := c.ComponentName()
		if seen[name] {
			errs = append(errs, fmt.Errorf("stack %q: duplicate component name %q", s.Name, name))
		}
		seen[name] = true
		if v, ok := c.(validator); ok {
			errs = append(errs, v.Validate()...)
		}
	}
	return errs
}
