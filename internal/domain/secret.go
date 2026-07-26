package domain

// SecretRef names a Kubernetes Secret that d7s never provisions, mutates,
// or reads the value of. Secrets are references, never inline values —
// unrepresentable at the schema level (golden rule 51).
type SecretRef struct {
	Name string
}
