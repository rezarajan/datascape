package domain

import "fmt"

// External is d7s's source(): a named, schema-carrying declaration of a
// resource d7s did not compile and will never provision or mutate — the
// only legal crossing point of the trust boundary (problem definition
// Amendment 2). It participates in the wiring graph (a component may
// reference it by name, e.g. guarantees.rpo.backupTo) and in conditional
// guarantees, but it never receives emitted infrastructure itself: an
// external declaration alone compiles to nothing (verified in
// internal/adapters/flux — an External with no component referencing it
// changes zero emitted bytes).
type External struct {
	Name string

	// ObjectStore is v1's only external shape: an S3-compatible object
	// store. A future external kind (e.g. an external database) would
	// add a sibling pointer here, mirroring how Postgres is today the
	// only Component kind.
	ObjectStore *ObjectStoreExternal
}

// ObjectStoreExternal is an S3-compatible object store external
// declaration. Region is optional per S3 semantics (many S3-compatible
// stores, e.g. self-hosted MinIO, don't require one). Credentials is
// always a Kubernetes Secret reference — never an inline value (golden
// rule 51) — d7s never reads, provisions, or mutates the secret it
// names.
type ObjectStoreExternal struct {
	Endpoint    string
	Bucket      string
	Region      string
	Credentials SecretRef
}

// Validate reports every structural problem with e, aggregated rather
// than stopping at the first (golden rule 33).
func (e External) Validate() []error {
	var errs []error
	if e.Name == "" {
		errs = append(errs, fmt.Errorf("external: name is required"))
	}
	if e.ObjectStore == nil {
		errs = append(errs, fmt.Errorf(
			"external %q: no store declared — v1's only external shape is objectStore (endpoint, bucket, credentials.secretRef)",
			e.Name))
		return errs
	}
	os := e.ObjectStore
	if os.Endpoint == "" {
		errs = append(errs, fmt.Errorf("external %q: objectStore.endpoint is required", e.Name))
	}
	if os.Bucket == "" {
		errs = append(errs, fmt.Errorf("external %q: objectStore.bucket is required", e.Name))
	}
	if os.Credentials.Name == "" {
		errs = append(errs, fmt.Errorf(
			"external %q: objectStore.credentials.secretRef.name is required (a Kubernetes Secret name — d7s never accepts an inline credential value)",
			e.Name))
	}
	return errs
}
