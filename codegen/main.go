package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

// main wires the CLI app and exits with a non-zero status on failure.
func main() {
	if err := App().Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "program exited: %v\n", err)
		os.Exit(1)
	}
}

// init configures Go's default structured logger so dependencies can emit
// consistent logs without reconfiguring slog in every package.
func init() {
	// Configure a minimal structured logger so subpackages can emit consistent logs.
	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
}
