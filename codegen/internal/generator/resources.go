package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func renderApiVersionResource(apiVersion string, params Params) error {
	apiVersion = strings.TrimSpace(apiVersion)
	if apiVersion == "" {
		return fmt.Errorf("missing api version in spec info")
	}
	targetDir := filepath.Join(params.ResourceDir, params.basePackagePath())
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create api version resource directory: %w", err)
	}
	target := filepath.Join(targetDir, "api-version.txt")
	if err := os.WriteFile(target, []byte(apiVersion), 0o644); err != nil {
		return fmt.Errorf("write api version resource: %w", err)
	}
	return nil
}
