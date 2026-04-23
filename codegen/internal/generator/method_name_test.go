package generator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateClientUsesCodegenMethodName(t *testing.T) {
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
  "paths": {
    "/v0.1/merchants/{merchant_code}/readers/{reader_id}/terminate": {
      "post": {
        "operationId": "CreateReaderTerminate",
        "summary": "Terminate a Reader Checkout",
        "parameters": [
          {
            "name": "merchant_code",
            "in": "path",
            "required": true,
            "schema": { "type": "string" }
          },
          {
            "name": "reader_id",
            "in": "path",
            "required": true,
            "schema": { "type": "string" }
          }
        ],
        "responses": {
          "204": {
            "description": "Terminated"
          }
        },
        "tags": ["Readers"],
        "x-codegen": {
          "method_name": "terminate_checkout"
        }
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

	clientPath := filepath.Join(outputDir, "com", "test", "sdk", "clients", "ReadersClient.java")
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("read generated client: %v", err)
	}
	generated := string(content)

	assertContains(t, generated, "void terminateCheckout(String merchantCode, String readerId)")
	assertContains(t, generated, "Operation ID: CreateReaderTerminate")
	assertNotContains(t, generated, "createReaderTerminate")
}
