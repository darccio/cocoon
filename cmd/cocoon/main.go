// Command cocoon generates pure Go packages from trusted Rust libraries.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/darccio/cocoon/internal/cli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	err := cli.Run(ctx, os.Args[1:], os.Stdout)
	cancel()
	if err != nil {
		if _, writeErr := fmt.Fprintln(os.Stderr, "cocoon:", err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}
