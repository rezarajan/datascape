package ports

import "github.com/rezarajan/datascape/internal/domain"

// Loader parses raw declaration bytes into a domain.Stack. It reports
// parse-time problems (malformed syntax, unknown or reserved fields) —
// domain-level structural validation is Stack.Validate, called
// separately so both error classes can be aggregated into one report
// (golden rule 33).
type Loader interface {
	Load(raw []byte) (domain.Stack, error)
}
