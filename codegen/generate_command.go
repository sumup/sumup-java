package main

import (
	"path/filepath"

	"github.com/sumup/sumup-java/codegen/internal/generator"
	"github.com/urfave/cli/v2"
)

// GenerateCommand wires the CLI action that renders the Java SDK from the
// project OpenAPI specification.
func GenerateCommand() *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Generate the SumUp Java SDK using the custom generator",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "spec",
				Usage:       "Path to the OpenAPI spec",
				Value:       "",
				DefaultText: "openapi.json",
				EnvVars:     []string{"SUMUP_OPENAPI_SPEC"},
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "Root directory where Java sources are written",
				Value:       "",
				DefaultText: filepath.Join("src", "main", "java"),
			},
			&cli.StringFlag{
				Name:  "package",
				Usage: "Base Java package for generated sources",
				Value: "com.sumup.sdk",
			},
		},
		Action: func(ctx *cli.Context) error {
			params := generator.Params{
				SpecPath:    ctx.String("spec"),
				OutputDir:   ctx.String("output"),
				BasePackage: ctx.String("package"),
			}

			return generator.Run(ctx.Context, params)
		},
	}
}
