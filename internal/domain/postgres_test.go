package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/rezarajan/datascape/internal/domain"
)

func validPostgres() domain.Postgres {
	return domain.Postgres{
		Name:        "orders-db",
		Placement:   domain.PlacementSelfHosted,
		Credentials: domain.SecretRef{Name: "orders-db-app"},
		Guarantees: domain.Guarantees{
			MTLS: &domain.MTLSGuarantee{},
		},
	}
}

func TestPostgresValidateAcceptsValid(t *testing.T) {
	if errs := validPostgres().Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestPostgresValidateManagedPlacementFailsClosed(t *testing.T) {
	p := validPostgres()
	p.Placement = domain.PlacementManaged
	errs := p.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %v", errs)
	}
	if got := errs[0].Error(); !strings.Contains(got, "planned, not yet available") {
		t.Errorf("error %q does not carry the remedy (golden rule 35)", got)
	}
}

func TestPostgresValidateAggregatesAllErrors(t *testing.T) {
	// Every field wrong at once: validate-time completeness (rule 33)
	// means every problem is reported together, not one at a time.
	p := domain.Postgres{}
	errs := p.Validate()
	// name, placement, credentials => at least 3 distinct problems.
	if len(errs) < 3 {
		t.Fatalf("expected aggregated errors for name/placement/credentials, got %v", errs)
	}
}

func TestPostgresValidateUnknownPlacementRefused(t *testing.T) {
	p := validPostgres()
	p.Placement = "on-prem"
	errs := p.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "unknown placement") {
		t.Fatalf("expected an unknown-placement error, got %v", errs)
	}
}

// TestPostgresValidateRPOFailsClosed proves guarantees.rpo refuses to
// compile in the same fail-closed style as placement: managed (golden
// rules 34, 35; owner decision, week-one plan "Owner decisions —
// 2026-07-26"): v1 has no declarable backup destination, so the
// durability guarantee's conformance probe could never pass, for any
// declared target — not only an unsatisfiable one.
func TestPostgresValidateRPOFailsClosed(t *testing.T) {
	p := validPostgres()
	p.Guarantees.RPO = &domain.RPOGuarantee{Target: time.Hour}
	errs := p.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %v", errs)
	}
	got := errs[0].Error()
	if !strings.Contains(got, "planned, not yet available") {
		t.Errorf("error %q does not carry the remedy (golden rule 35)", got)
	}
	if !strings.Contains(got, "remove guarantees.rpo") {
		t.Errorf("error %q does not name the concrete remedy", got)
	}
	if !strings.Contains(got, "conformance probe could never pass") {
		t.Errorf("error %q does not state why (golden rule 34)", got)
	}
}

// TestPostgresValidateWithoutRPOCompiles proves the same declaration,
// minus guarantees.rpo, compiles cleanly (rule 49: the check shown able
// to fail and pass).
func TestPostgresValidateWithoutRPOCompiles(t *testing.T) {
	p := validPostgres()
	if errs := p.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors without guarantees.rpo, got %v", errs)
	}
}

func TestPostgresValidateAllowedConsumersRequireMTLS(t *testing.T) {
	p := validPostgres()
	p.Guarantees.MTLS = nil
	p.AllowedConsumers = []domain.AllowedConsumer{{ServiceAccount: "probe-client"}}
	errs := p.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "allowedConsumers declared without guarantees.mtls") {
		t.Fatalf("expected an allowedConsumers-without-mtls error, got %v", errs)
	}
}

func TestPostgresValidateAllowedConsumerRequiresServiceAccount(t *testing.T) {
	p := validPostgres()
	p.AllowedConsumers = []domain.AllowedConsumer{{}}
	errs := p.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "serviceAccount is required") {
		t.Fatalf("expected a serviceAccount-required error, got %v", errs)
	}
}

func TestPostgresValidateAcceptsDeclaredConsumer(t *testing.T) {
	p := validPostgres()
	p.AllowedConsumers = []domain.AllowedConsumer{{ServiceAccount: "probe-client"}}
	if errs := p.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}
