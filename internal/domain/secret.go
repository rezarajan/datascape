package domain

// SecretRef names a Kubernetes Secret that d7s never provisions, mutates,
// or reads the value of. Secrets are references, never inline values —
// unrepresentable at the schema level (golden rule 51).
//
// For a Postgres component's Credentials, the named Secret must already
// exist (with "username"/"password" keys) before the compiled Cluster
// CR is applied: CNPG's bootstrap.initdb.secret only consumes a
// pre-existing secret, it does not generate one for a name the caller
// supplied. Found by running the acceptance harness against a live
// CNPG operator (golden rule 40) — the role and database were created
// with no password set until the secret existed first.
type SecretRef struct {
	Name string
}
