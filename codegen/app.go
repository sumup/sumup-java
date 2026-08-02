package main

import (
	"github.com/urfave/cli/v2"
)

// App wires the CLI commands exposed by the codegen binary.
func App() *cli.App {
	return &cli.App{
		Name:                 "sumup-java-codegen",
		Usage:                "Generate the SumUp Java SDK from openapi.json",
		EnableBashCompletion: true,
		DefaultCommand:       "generate",
		Commands: []*cli.Command{
			GenerateCommand(),
			SamplesCommand(),
		},
	}
}
