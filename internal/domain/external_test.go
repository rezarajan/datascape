package domain_test

import (
	"strings"
	"testing"

	"github.com/rezarajan/datascape/internal/domain"
)

func validExternal() domain.External {
	return domain.External{
		Name: "backups",
		ObjectStore: &domain.ObjectStoreExternal{
			Endpoint:    "https://minio.example.com",
			Bucket:      "d7s-backups",
			Credentials: domain.SecretRef{Name: "backups-credentials"},
		},
	}
}

func TestExternalValidateAcceptsValid(t *testing.T) {
	if errs := validExternal().Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestExternalValidateRequiresName(t *testing.T) {
	e := validExternal()
	e.Name = ""
	errs := e.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "name is required") {
		t.Fatalf("expected a name-required error, got %v", errs)
	}
}

// TestExternalValidateRequiresObjectStore proves v1's only external shape
// (objectStore) is required — a name-only external with no store shape
// refuses rather than silently emitting nothing useful (golden rule 34).
func TestExternalValidateRequiresObjectStore(t *testing.T) {
	e := domain.External{Name: "backups"}
	errs := e.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "no store declared") {
		t.Fatalf("expected a no-store-declared error, got %v", errs)
	}
}

// TestExternalValidateObjectStoreAggregatesRequiredFields proves every
// missing objectStore field is reported together (golden rule 33), not
// one at a time.
func TestExternalValidateObjectStoreAggregatesRequiredFields(t *testing.T) {
	e := domain.External{Name: "backups", ObjectStore: &domain.ObjectStoreExternal{}}
	errs := e.Validate()
	if len(errs) != 3 {
		t.Fatalf("expected 3 aggregated errors (endpoint, bucket, credentials), got %v", errs)
	}
}

func TestExternalValidateRequiresEndpoint(t *testing.T) {
	e := validExternal()
	e.ObjectStore.Endpoint = ""
	errs := e.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "objectStore.endpoint is required") {
		t.Fatalf("expected an endpoint-required error, got %v", errs)
	}
}

func TestExternalValidateRequiresBucket(t *testing.T) {
	e := validExternal()
	e.ObjectStore.Bucket = ""
	errs := e.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "objectStore.bucket is required") {
		t.Fatalf("expected a bucket-required error, got %v", errs)
	}
}

// TestExternalValidateRequiresCredentialsSecretRefNeverInline proves
// credentials.secretRef.name is required — there is no field on
// ObjectStoreExternal representable as an inline credential value (golden
// rule 51: unrepresentable at the schema level, not merely refused).
func TestExternalValidateRequiresCredentialsSecretRefNeverInline(t *testing.T) {
	e := validExternal()
	e.ObjectStore.Credentials = domain.SecretRef{}
	errs := e.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "credentials.secretRef.name is required") {
		t.Fatalf("expected a credentials-required error, got %v", errs)
	}
}

// TestExternalValidateRegionOptional proves region is optional (many
// S3-compatible stores, e.g. self-hosted MinIO, don't require one).
func TestExternalValidateRegionOptional(t *testing.T) {
	e := validExternal()
	e.ObjectStore.Region = ""
	if errs := e.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors with region unset, got %v", errs)
	}
}
