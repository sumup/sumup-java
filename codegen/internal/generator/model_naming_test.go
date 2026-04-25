package generator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateModelFlattensAllOfIntoComponent(t *testing.T) {
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
      "Merchant": {
        "allOf": [
          {
            "type": "object",
            "required": ["merchant_code"],
            "properties": {
              "merchant_code": { "type": "string" }
            }
          },
          {
            "$ref": "#/components/schemas/Timestamps"
          }
        ],
        "title": "Merchant"
      },
      "Timestamps": {
        "type": "object",
        "properties": {
          "created_at": { "type": "string", "format": "date-time", "readOnly": true }
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

	merchantPath := filepath.Join(outputDir, "com", "test", "sdk", "models", "Merchant.java")
	content, err := os.ReadFile(merchantPath)
	if err != nil {
		t.Fatalf("read generated Merchant model: %v", err)
	}
	generated := string(content)

	assertContains(t, generated, "public record Merchant(")
	assertContains(t, generated, "String merchantCode")
	assertContains(t, generated, "java.time.OffsetDateTime createdAt")
	assertNotContains(t, generated, "Merchant2")
	assertFileDoesNotExist(t, filepath.Join(outputDir, "com", "test", "sdk", "models", "Merchant2.java"))
}

func TestGenerateClientUsesOpenEnumsForInlineParameterEnums(t *testing.T) {
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
    "/v1/transactions": {
      "get": {
        "operationId": "ListTransactions",
        "parameters": [
          {
            "name": "order",
            "in": "query",
            "schema": {
              "type": "string",
              "enum": ["ascending", "descending"]
            }
          },
          {
            "name": "types",
            "in": "query",
            "schema": {
              "type": "array",
              "items": {
                "type": "string",
                "enum": ["PAYMENT", "REFUND"]
              }
            }
          }
        ],
        "responses": {
          "204": {
            "description": "No content"
          }
        },
        "tags": ["Transactions"]
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

	clientPath := filepath.Join(outputDir, "com", "test", "sdk", "clients", "TransactionsClient.java")
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("read generated Transactions client: %v", err)
	}
	generated := string(content)

	assertContains(t, generated, "public ListTransactionsQueryParams order(com.test.sdk.models.ListTransactionsOrder value)")
	assertContains(t, generated, "public ListTransactionsQueryParams types(java.util.List<com.test.sdk.models.ListTransactionsTypesItem> value)")
	assertContains(t, generated, "com.test.sdk.models.ListTransactionsOrder")
	assertContains(t, generated, "com.test.sdk.models.ListTransactionsTypesItem")
	assertFileDoesNotExist(t, filepath.Join(outputDir, "com", "test", "sdk", "models", "Order.java"))
	assertFileDoesNotExist(t, filepath.Join(outputDir, "com", "test", "sdk", "models", "TypesItem.java"))

	orderPath := filepath.Join(outputDir, "com", "test", "sdk", "models", "ListTransactionsOrder.java")
	orderContent, err := os.ReadFile(orderPath)
	if err != nil {
		t.Fatalf("read generated ListTransactionsOrder model: %v", err)
	}
	orderGenerated := string(orderContent)

	assertContains(t, orderGenerated, "public final class ListTransactionsOrder")
	assertContains(t, orderGenerated, `public static final ListTransactionsOrder ASCENDING = new ListTransactionsOrder("ascending");`)
	assertContains(t, orderGenerated, "public static ListTransactionsOrder of(String value)")
	assertContains(t, orderGenerated, "return value == null ? null : new ListTransactionsOrder(value);")
	assertNotContains(t, orderGenerated, "public enum ListTransactionsOrder")
}

func assertFileDoesNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected %s to not exist", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}
