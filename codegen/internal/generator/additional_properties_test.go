package generator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateModelWithPropertiesAndAdditionalProperties(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	specPath := filepath.Join(tmp, "openapi.json")
	outputDir := filepath.Join(tmp, "src", "main", "java")
	resourceDir := filepath.Join(tmp, "src", "main", "resources")

	spec := `{
  "openapi": "3.0.3",
  "info": {
    "title": "test",
    "version": "1.0.0"
  },
  "paths": {},
  "components": {
    "schemas": {
      "Problem": {
        "type": "object",
        "properties": {
          "type": { "type": "string" },
          "title": { "type": "string" }
        },
        "required": ["type"],
        "additionalProperties": true
      }
    }
  }
}`
	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	params := Params{
		SpecPath:    specPath,
		OutputDir:   outputDir,
		ResourceDir: resourceDir,
		BasePackage: "com.test.sdk",
	}
	if err := Run(context.Background(), params); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	modelPath := filepath.Join(outputDir, "com", "test", "sdk", "models", "Problem.java")
	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read generated model: %v", err)
	}
	generated := string(content)

	assertContains(t, generated, "@JsonDeserialize(builder = Problem.Builder.class)")
	assertContains(t, generated, "java.util.Map<String, Object> additionalProperties")
	assertContains(t, generated, "@JsonAnyGetter")
	assertContains(t, generated, "@JsonAnySetter")
	assertContains(t, generated, "public Builder additionalProperty(String name, Object value)")
}

func assertContains(t *testing.T, content, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Fatalf("expected generated output to contain %q", want)
	}
}
