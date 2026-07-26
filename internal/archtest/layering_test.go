// Package archtest mechanically enforces the hexagonal layering golden
// rule 8: domain imports nothing, ports import only domain, adapters
// implement ports, and only the composition root knows concrete adapters.
// "Discipline decays; nothing stops an import" — this test does.
package archtest

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

const module = "github.com/rezarajan/datascape"

type goPackage struct {
	ImportPath string
	Imports    []string
}

func loadPackages(t *testing.T) []goPackage {
	t.Helper()
	// An absolute module pattern, not "./...": the test binary's working
	// directory is this package's own directory, so a relative pattern
	// would silently scope the check to archtest alone.
	out, err := exec.Command("go", "list", "-json", module+"/...").Output()
	if err != nil {
		t.Fatalf("go list -json %s/...: %v", module, err)
	}
	var pkgs []goPackage
	dec := json.NewDecoder(strings.NewReader(string(out)))
	for dec.More() {
		var p goPackage
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("decode go list output: %v", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs
}

// layer classifies a package under the module by its hexagonal role.
// Packages outside the named planes (e.g. this test package itself) are
// "other" and are not subject to the layering rules below.
func layer(importPath string) string {
	rel := strings.TrimPrefix(importPath, module+"/")
	switch {
	case rel == "internal/domain" || strings.HasPrefix(rel, "internal/domain/"):
		return "domain"
	case rel == "internal/ports" || strings.HasPrefix(rel, "internal/ports/"):
		return "ports"
	case rel == "internal/compiler" || strings.HasPrefix(rel, "internal/compiler/"):
		return "compiler"
	case strings.HasPrefix(rel, "internal/adapters/"):
		return "adapters"
	case strings.HasPrefix(rel, "cmd/"):
		return "root"
	default:
		return "other"
	}
}

// allowed lists, for each layer, which other in-module layers it may
// import. A layer may always import itself (a sibling sub-package).
var allowed = map[string]map[string]bool{
	"domain":   {},
	"ports":    {"domain": true},
	"compiler": {"domain": true, "ports": true},
	"adapters": {"domain": true, "ports": true},
	"root":     {"domain": true, "ports": true, "compiler": true, "adapters": true},
}

// TestLayering enforces inward-pointing dependencies mechanically.
func TestLayering(t *testing.T) {
	for _, p := range loadPackages(t) {
		from := layer(p.ImportPath)
		if from == "other" {
			continue
		}
		for _, imp := range p.Imports {
			if !strings.HasPrefix(imp, module+"/") {
				continue // stdlib / third-party: not a layering concern
			}
			to := layer(imp)
			switch {
			case to == "root":
				t.Errorf("%s (%s) imports %s (composition root) — nothing may import the root", p.ImportPath, from, imp)
			case to == "other":
				// not a named plane; no rule to enforce
			case from == to:
				// a layer may import its own sub-packages
			case !allowed[from][to]:
				t.Errorf("%s (%s) imports %s (%s) — violates hexagonal layering (golden rule 8)", p.ImportPath, from, imp, to)
			}
		}
	}
}
