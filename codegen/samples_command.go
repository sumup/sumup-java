package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sumup/sumup-java/codegen/internal/generator"
	"github.com/urfave/cli/v2"
)

// SamplesCommand generates the Java code-sample catalog consumed by documentation sites.
func SamplesCommand() *cli.Command {
	return &cli.Command{
		Name:  "samples",
		Usage: "Generate Java code samples as a JSON catalog",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "spec",
				Usage:       "Path to the OpenAPI spec",
				Value:       "",
				DefaultText: "openapi.json",
				EnvVars:     []string{"SUMUP_OPENAPI_SPEC"},
			},
			&cli.StringFlag{
				Name:  "package",
				Usage: "Base Java package for generated sources",
				Value: "com.sumup.sdk",
			},
			&cli.StringFlag{
				Name:    "out",
				Aliases: []string{"o"},
				Usage:   "Path of the output JSON file (defaults to stdout)",
			},
			&cli.StringFlag{
				Name:  "sdk-version",
				Usage: "SDK version represented by the samples",
			},
			&cli.PathFlag{
				Name:  "sdk-version-file",
				Usage: "File containing the SDK version",
				Value: filepath.Join("..", "VERSION"),
			},
		},
		Action: func(ctx *cli.Context) error {
			sdkVersion := strings.TrimSpace(ctx.String("sdk-version"))
			if sdkVersion == "" {
				version, err := readSDKVersion(ctx.Path("sdk-version-file"))
				if err != nil {
					return err
				}
				sdkVersion = version
			}

			catalog, err := generator.BuildSamples(generator.Params{
				SpecPath:    ctx.String("spec"),
				BasePackage: ctx.String("package"),
			}, sdkVersion)
			if err != nil {
				return fmt.Errorf("generate samples: %w", err)
			}
			encoded, err := json.MarshalIndent(catalog, "", "  ")
			if err != nil {
				return fmt.Errorf("encode samples: %w", err)
			}
			encoded = append(encoded, '\n')

			stdout := ctx.App.Writer
			if stdout == nil {
				stdout = os.Stdout
			}
			return writeSamples(ctx.String("out"), encoded, stdout)
		},
	}
}

func readSDKVersion(filename string) (string, error) {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("read SDK version: %w", err)
	}
	version := strings.TrimSpace(string(contents))
	if version == "" {
		return "", fmt.Errorf("SDK version file %q is empty", filename)
	}
	return version, nil
}

func writeSamples(out string, encoded []byte, stdout io.Writer) error {
	if out == "" {
		if _, err := stdout.Write(encoded); err != nil {
			return fmt.Errorf("write samples: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("create samples directory: %w", err)
	}
	if err := os.WriteFile(out, encoded, 0o644); err != nil {
		return fmt.Errorf("write samples: %w", err)
	}
	return nil
}
