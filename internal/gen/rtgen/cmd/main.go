// Command rtgen reproduces the runtime's shared checked adapters.
package main

import (
	"fmt"
	"os"

	"dario.cat/cocoon/internal/gen/rtgen"
)

func main() {
	err := generate()
	if err != nil {
		if _, writeErr := fmt.Fprintln(os.Stderr, "rtgen:", err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}

func generate() error {
	existing, err := os.ReadFile("instance.go")
	if err != nil {
		return fmt.Errorf("read runtime source: %w", err)
	}
	output, err := rtgen.Update(existing)
	if err != nil {
		return err
	}
	if err := os.WriteFile("instance.go", output, 0o600); err != nil { // #nosec G703 -- Fixed repository path, independent of source contents.
		return fmt.Errorf("write checked boundary: %w", err)
	}
	return nil
}
