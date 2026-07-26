package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/rezarajan/datascape/internal/domain"
)

// TestStackValidateExternalAlone proves an external declaration validates
// cleanly on its own, with no component referencing it — the declaration
// model accepts it structurally; whether it emits anything is the Flux
// emitter's concern (internal/adapters/flux/flux_test.go), not domain
// validation's.
func TestStackValidateExternalAlone(t *testing.T) {
	s := domain.Stack{
		Name: "week-three",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
			},
		},
		Externals: []domain.External{validExternal()},
	}
	if errs := s.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

// TestStackValidateExternalStructuralErrorsAggregate proves a malformed
// external's own errors surface through Stack.Validate, aggregated with
// everything else (golden rule 33).
func TestStackValidateExternalStructuralErrorsAggregate(t *testing.T) {
	s := domain.Stack{
		Name: "week-three",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
			},
		},
		Externals: []domain.External{{Name: "backups"}},
	}
	errs := s.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "no store declared") {
		t.Fatalf("expected the external's own structural error to surface, got %v", errs)
	}
}

// TestStackValidateDuplicateExternalNameRefused mirrors the duplicate
// component-name check for externals.
func TestStackValidateDuplicateExternalNameRefused(t *testing.T) {
	ext := validExternal()
	s := domain.Stack{
		Name: "week-three",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
			},
		},
		Externals: []domain.External{ext, ext},
	}
	errs := s.Validate()
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "duplicate external name") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a duplicate-external-name error, got %v", errs)
	}
}

// TestStackValidateRPOReferencingUndeclaredExternalRefused proves the
// cross-declaration referential check (week-three plan, slices 1+2): a
// component's guarantees.rpo.backupTo naming an external that was never
// declared refuses at the stack level, with a remedy (golden rules 34,
// 35), before compilation ever reaches an emitter that would otherwise
// fail to resolve it.
func TestStackValidateRPOReferencingUndeclaredExternalRefused(t *testing.T) {
	s := domain.Stack{
		Name: "week-three",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
				Guarantees: domain.Guarantees{
					RPO: &domain.RPOGuarantee{Target: time.Hour, BackupTo: "backups"},
				},
			},
		},
		// No Externals declared at all.
	}
	errs := s.Validate()
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), `references undeclared external "backups"`) &&
			strings.Contains(e.Error(), "declare it in the stack's external block") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an undeclared-external error naming the remedy, got %v", errs)
	}
}

// TestStackValidateRPOReferencingDeclaredExternalCompiles proves the
// positive case: once the named external is declared, the cross-check
// passes (rule 49: the check shown able to fail and pass).
func TestStackValidateRPOReferencingDeclaredExternalCompiles(t *testing.T) {
	s := domain.Stack{
		Name: "week-three",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
				Guarantees: domain.Guarantees{
					RPO: &domain.RPOGuarantee{Target: time.Hour, BackupTo: "backups"},
				},
			},
		},
		Externals: []domain.External{validExternal()},
	}
	if errs := s.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}
