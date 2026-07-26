// Command d7s is the composition root: the only place that selects a
// concrete target emitter (golden rule 8).
package main

import (
	"fmt"
	"os"

	"github.com/rezarajan/datascape/internal/adapters/flux"
	"github.com/rezarajan/datascape/internal/compiler"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	_ = compiler.New(flux.New())
	return fmt.Errorf("d7s: no subcommand implemented yet — planned, not yet available")
}
