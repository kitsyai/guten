package main

import (
	"fmt"
	"os"

	"github.com/kitsyai/guten/cli/internal/library"
)

// cmdNew implements `guten new <name> [--from <builtin>] [--lib-dir <dir>]`:
// it scaffolds a template.json + part file(s) + sample.json into the user
// library, optionally cloned from an existing bundle (builtin, gutenkit, or
// another user template).
func cmdNew(args []string) error {
	if len(args) == 0 || len(args[0]) == 0 || args[0][0] == '-' {
		return fmt.Errorf("usage: guten new <name> [--from <builtin>] [--lib-dir <dir>]")
	}
	name := args[0]
	var from, libDir string
	for i := 1; i < len(args); i++ {
		a := args[i]
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("missing value for %s", a)
			}
			i++
			return args[i], nil
		}
		var err error
		switch a {
		case "--from":
			from, err = next()
		case "--lib-dir":
			libDir, err = next()
		default:
			return fmt.Errorf("unknown flag %q", a)
		}
		if err != nil {
			return err
		}
	}
	dir, err := library.NewUserTemplate(name, from, libDir)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "created %s\n", dir)
	return nil
}
