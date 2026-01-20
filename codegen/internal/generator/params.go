package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Params configure a generator run.
type Params struct {
	SpecPath    string
	OutputDir   string
	ResourceDir string
	BasePackage string
}

// normalize fills missing configuration with defaults and resolves absolute
// paths, ensuring downstream rendering code can operate deterministically.
func (p *Params) normalize() error {
	if p.BasePackage == "" {
		p.BasePackage = "com.sumup.sdk"
	}
	if p.OutputDir == "" {
		p.OutputDir = filepath.Join("..", "src", "main", "java")
	}
	var err error
	if p.SpecPath == "" {
		p.SpecPath = filepath.Join("..", "openapi.json")
	}
	if p.SpecPath, err = filepath.Abs(p.SpecPath); err != nil {
		return fmt.Errorf("resolve spec path: %w", err)
	}
	if p.OutputDir, err = filepath.Abs(p.OutputDir); err != nil {
		return fmt.Errorf("resolve output directory: %w", err)
	}
	if p.ResourceDir == "" {
		p.ResourceDir = filepath.Join(filepath.Dir(p.OutputDir), "resources")
	}
	if p.ResourceDir, err = filepath.Abs(p.ResourceDir); err != nil {
		return fmt.Errorf("resolve resource directory: %w", err)
	}
	return nil
}

// validate ensures the OpenAPI specification exists and the target directory is
// writable (creating it if needed).
func (p *Params) validate() error {
	if _, err := os.Stat(p.SpecPath); err != nil {
		return fmt.Errorf("spec not found: %w", err)
	}
	fi, err := os.Stat(p.OutputDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(p.OutputDir, 0o755); err != nil {
				return fmt.Errorf("create output directory: %w", err)
			}
		} else {
			return fmt.Errorf("output directory: %w", err)
		}
	} else if !fi.IsDir() {
		return fmt.Errorf("output directory %s is not a directory", p.OutputDir)
	}
	fi, err = os.Stat(p.ResourceDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(p.ResourceDir, 0o755); err != nil {
				return fmt.Errorf("create resource directory: %w", err)
			}
		} else {
			return fmt.Errorf("resource directory: %w", err)
		}
	} else if !fi.IsDir() {
		return fmt.Errorf("resource directory %s is not a directory", p.ResourceDir)
	}
	return nil
}

// basePackagePath translates the base Java package into a filesystem path
// relative to the configured output directory.
func (p Params) basePackagePath() string {
	return filepath.Join(strings.Split(p.BasePackage, ".")...)
}

// clientPackage returns the Java package path where API clients live.
func (p Params) clientPackage() string {
	return p.BasePackage + ".clients"
}

// clientPackagePath returns the filesystem path that backs the client package.
func (p Params) clientPackagePath() string {
	return filepath.Join(p.basePackagePath(), "clients")
}

// modelPackage returns the Java package for generated shared models.
func (p Params) modelPackage() string {
	return p.BasePackage + ".models"
}

// modelPackagePath returns the filesystem path to the models package.
func (p Params) modelPackagePath() string {
	return filepath.Join(p.basePackagePath(), "models")
}
