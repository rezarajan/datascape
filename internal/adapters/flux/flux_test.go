package flux_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/rezarajan/datascape/internal/adapters/flux"
	"github.com/rezarajan/datascape/internal/domain"
)

func exampleStack() domain.Stack {
	return domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
				Guarantees: domain.Guarantees{
					MTLS: &domain.MTLSGuarantee{},
					RPO:  &domain.RPOGuarantee{Target: time.Hour},
				},
				AllowedConsumers: []domain.AllowedConsumer{
					{ServiceAccount: "probe-client"},
				},
			},
		},
	}
}

// TestEmitGoldenFiles pins compiled output against the checked-in golden
// tree (golden rule 45): the acceptance example, compiled once here and
// committed under testdata/golden, must match byte for byte.
func TestEmitGoldenFiles(t *testing.T) {
	manifests, err := flux.New().Emit(exampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}

	want := readGoldenDir(t, filepath.Join("testdata", "golden", "week-one"))

	if len(manifests.Files) != len(want) {
		t.Fatalf("emitted %d files, golden has %d\nemitted: %v\ngolden: %v",
			len(manifests.Files), len(want), sortedKeys(manifests.Files), sortedKeys(want))
	}
	for path, wantBytes := range want {
		gotBytes, ok := manifests.Files[path]
		if !ok {
			t.Errorf("golden file %s was not emitted", path)
			continue
		}
		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("emitted %s differs from golden:\n--- got ---\n%s\n--- want ---\n%s", path, gotBytes, wantBytes)
		}
	}
}

// TestEmitDeterministic compiles the same declaration twice and requires
// byte-identical output (golden rules 22, 45).
func TestEmitDeterministic(t *testing.T) {
	a, err := flux.New().Emit(exampleStack())
	if err != nil {
		t.Fatalf("Emit (first): %v", err)
	}
	b, err := flux.New().Emit(exampleStack())
	if err != nil {
		t.Fatalf("Emit (second): %v", err)
	}
	if len(a.Files) != len(b.Files) {
		t.Fatalf("file count differs across compiles: %d vs %d", len(a.Files), len(b.Files))
	}
	for path, want := range a.Files {
		got, ok := b.Files[path]
		if !ok {
			t.Fatalf("second compile is missing file %s", path)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("file %s is not byte-identical across compiles", path)
		}
	}
}

// TestEmitWithoutMTLSGuaranteeEmitsNoZeroTrustObjects proves presence is
// the only signal (golden rule 50): a component that doesn't declare
// the mtls guarantee gets no PeerAuthentication or AuthorizationPolicy
// at all, rather than a permissive or disabled variant of either.
func TestEmitWithoutMTLSGuaranteeEmitsNoZeroTrustObjects(t *testing.T) {
	stack := domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
			},
		},
	}
	manifests, err := flux.New().Emit(stack)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	for path := range manifests.Files {
		if filepath.Base(path) == "peerauthentication.yaml" || filepath.Base(path) == "orders-db-authorizationpolicy.yaml" {
			t.Errorf("emitted %s without a declared mtls guarantee", path)
		}
	}
}

// TestEmitWithoutRPOGuaranteeEmitsNoBackupObjects mirrors the mTLS case
// for durability: no ScheduledBackup and no Cluster.spec.backup unless
// the RPO guarantee is declared.
func TestEmitWithoutRPOGuaranteeEmitsNoBackupObjects(t *testing.T) {
	stack := domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
			},
		},
	}
	manifests, err := flux.New().Emit(stack)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	for path := range manifests.Files {
		if filepath.Base(path) == "orders-db-scheduledbackup.yaml" {
			t.Errorf("emitted %s without a declared rpo guarantee", path)
		}
	}
	if bytes.Contains(manifests.Files["apps/week-one/orders-db-cluster.yaml"], []byte("backup")) {
		t.Error("Cluster CR mentions backup without a declared rpo guarantee")
	}
}

// TestEmitUnsatisfiableRPORefused proves the durability guarantee's
// compile-time check can fail (golden rule 49's "proven by the negative
// probe" spirit, exercised here at compile time): an RPO below what the
// emitter can honor refuses compilation with the remedy in the error
// (golden rules 34, 35), and nothing is written for that call.
func TestEmitUnsatisfiableRPORefused(t *testing.T) {
	stack := domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
				Guarantees: domain.Guarantees{
					RPO: &domain.RPOGuarantee{Target: 2 * time.Minute},
				},
			},
		},
	}
	manifests, err := flux.New().Emit(stack)
	if err == nil {
		t.Fatal("expected an error for an unsatisfiable RPO, got nil")
	}
	if !strings.Contains(err.Error(), "cannot be honored") {
		t.Errorf("error %q does not explain why the RPO can't be honored", err.Error())
	}
	if !strings.Contains(err.Error(), "declare a larger value") {
		t.Errorf("error %q does not carry a remedy (golden rule 35)", err.Error())
	}
	if len(manifests.Files) != 0 {
		t.Errorf("expected no files written on refusal, got %v", sortedKeys(manifests.Files))
	}
}

// unknownComponent is a domain.Component of a kind the flux emitter does
// not implement, used to prove the unimplemented-kind path refuses
// loudly (golden rule 34) instead of silently skipping it.
type unknownComponent struct{}

func (unknownComponent) ComponentName() string      { return "mystery" }
func (unknownComponent) Kind() domain.ComponentKind { return "kafka" }

func TestEmitUnknownComponentKindRefused(t *testing.T) {
	stack := domain.Stack{
		Name:       "week-one",
		Components: []domain.Component{unknownComponent{}},
	}
	_, err := flux.New().Emit(stack)
	if err == nil {
		t.Fatal("expected an error for an unimplemented component kind, got nil")
	}
}

func readGoldenDir(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = b
		return nil
	})
	if err != nil {
		t.Fatalf("read golden dir %s: %v", dir, err)
	}
	return files
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
