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
	// Externals declares d7s's source()s: named resources d7s never
	// provisions or mutates (problem definition Amendment 2). An
	// external alone emits nothing — it becomes observable only when a
	// component wires to it by name (e.g. guarantees.rpo.backupTo).
	Externals []External
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

// externalReferencer is implemented by any component that can reference
// a declared external by name (e.g. Postgres.ExternalRefs, for
// guarantees.rpo.backupTo). Stack.Validate cross-checks every name
// returned here against Stack.Externals — a stack-level, component-kind-
// agnostic check, since no single component has visibility into its
// siblings' declarations (golden rule 9: this cross-declaration
// referential check lives in the declaration model, not precipitated
// into a component's own Validate; the target-specific work of actually
// resolving a validated reference into emitted infrastructure — e.g. the
// CNPG barmanObjectStore shape — stays in the Flux emitter,
// internal/adapters/flux/durability.go, since that shape is target-
// specific and the declaration model must stay target-agnostic).
type externalReferencer interface {
	ExternalRefs() []string
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

	declaredExternals := make(map[string]bool, len(s.Externals))
	seenExternal := make(map[string]bool, len(s.Externals))
	for _, e := range s.Externals {
		if seenExternal[e.Name] {
			errs = append(errs, fmt.Errorf("stack %q: duplicate external name %q", s.Name, e.Name))
		}
		seenExternal[e.Name] = true
		errs = append(errs, e.Validate()...)
		if e.Name != "" {
			declaredExternals[e.Name] = true
		}
	}

	for _, c := range s.Components {
		er, ok := c.(externalReferencer)
		if !ok {
			continue
		}
		for _, ref := range er.ExternalRefs() {
			if !declaredExternals[ref] {
				errs = append(errs, fmt.Errorf(
					"stack %q: component %q references undeclared external %q — "+
						"declare it in the stack's external block, or remove the reference",
					s.Name, c.ComponentName(), ref))
			}
		}
	}

	return errs
}
