// Command d7s is the composition root: the only place that selects
// concrete adapters (golden rule 8).
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rezarajan/datascape/internal/adapters/flux"
	yamlloader "github.com/rezarajan/datascape/internal/adapters/yaml"
	"github.com/rezarajan/datascape/internal/compiler"
	"github.com/rezarajan/datascape/internal/ports"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("d7s: usage: d7s compile <file> -o <dir>")
	}
	switch args[0] {
	case "compile":
		return runCompile(args[1:])
	default:
		return fmt.Errorf("d7s: unknown subcommand %q — planned, not yet available", args[0])
	}
}

// runCompile parses its own arguments rather than using flag.FlagSet:
// the documented, acceptance-tested invocation is
// "d7s compile <file> -o <dir>" (rule 41 — the flag follows the
// positional file argument), which Go's flag package cannot parse since
// it stops at the first non-flag argument.
func runCompile(args []string) error {
	const usage = "d7s compile: usage: d7s compile <file> -o <dir>"

	var file, out string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", "--o":
			if i+1 >= len(args) {
				return fmt.Errorf("%s", usage)
			}
			out = args[i+1]
			i++
		default:
			if file != "" {
				return fmt.Errorf("%s", usage)
			}
			file = args[i]
		}
	}
	if file == "" || out == "" {
		return fmt.Errorf("%s", usage)
	}

	raw, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("d7s compile: read %s: %w", file, err)
	}

	stack, err := yamlloader.New().Load(raw)
	if err != nil {
		return fmt.Errorf("d7s compile: %w", err)
	}

	manifests, err := compiler.New(flux.New()).Compile(stack)
	if err != nil {
		return fmt.Errorf("d7s compile: %w", err)
	}

	return writeManifests(out, manifests)
}

// writeManifests is the only side effect d7s compile performs: writing
// local files. It never touches a cluster (golden rule: no mutating
// verbs against any backend).
func writeManifests(dir string, manifests ports.Manifests) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("d7s compile: clear output dir %s: %w", dir, err)
	}
	for path, contents := range manifests.Files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("d7s compile: create %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, contents, 0o644); err != nil {
			return fmt.Errorf("d7s compile: write %s: %w", full, err)
		}
	}
	return nil
}
