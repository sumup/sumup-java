package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateScalarReferences(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	specPath := filepath.Join(tmp, "openapi.yaml")
	spec := `openapi: 3.1.0
info: {title: Scalars, version: '1'}
paths:
  /readers/{id}:
    post:
      operationId: createReader
      x-codegen: {method_name: create}
      tags: [Readers]
      parameters:
        - name: id
          in: path
          required: true
          schema: {$ref: '#/components/schemas/Name'}
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Request'}
      responses:
        '200':
          description: Name
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Name'}
components:
  schemas:
    Name: {type: string, description: Reader display name., example: Counter 1}
    Count: {type: integer, format: int32}
    Total: {type: integer, format: int64}
    Latitude: {type: number, format: float}
    Amount: {type: number, format: double}
    Enabled: {type: boolean}
    Date: {type: string, format: date}
    Timestamp: {type: string, format: date-time}
    Identifier: {type: string, format: uuid}
    NullableName: {type: [string, 'null']}
    Status: {type: string, enum: [active, inactive]}
    Request:
      type: object
      required: [name, nullableName]
      properties:
        name: {$ref: '#/components/schemas/Name'}
        nullableName: {$ref: '#/components/schemas/NullableName'}
        count: {$ref: '#/components/schemas/Count'}
        total: {$ref: '#/components/schemas/Total'}
        latitude: {$ref: '#/components/schemas/Latitude'}
        amount: {$ref: '#/components/schemas/Amount'}
        enabled: {$ref: '#/components/schemas/Enabled'}
        date: {$ref: '#/components/schemas/Date'}
        timestamp: {$ref: '#/components/schemas/Timestamp'}
        identifier: {$ref: '#/components/schemas/Identifier'}
        status: {$ref: '#/components/schemas/Status'}
        names:
          type: array
          items: {$ref: '#/components/schemas/Name'}
        counts:
          type: object
          additionalProperties: {$ref: '#/components/schemas/Count'}
`
	if err := os.WriteFile(specPath, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}
	params := Params{SpecPath: specPath, OutputDir: filepath.Join(tmp, "java"), ResourceDir: filepath.Join(tmp, "resources")}
	if err := Run(t.Context(), params); err != nil {
		t.Fatal(err)
	}
	read := func(path string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(params.OutputDir, "com/sumup/sdk", path))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	request := read("models/Request.java")
	for _, field := range []string{
		"String name", "String nullableName", "Integer count", "Long total", "Float latitude", "Double amount",
		"Boolean enabled", "java.time.LocalDate date", "java.time.OffsetDateTime timestamp", "java.util.UUID identifier",
		"com.sumup.sdk.models.Status status", "java.util.List<String> names", "java.util.Map<String, Integer> counts",
		"Reader display name.", `Objects.requireNonNull(name, "name")`,
	} {
		assertContains(t, request, field)
	}
	assertNotContains(t, request, `Objects.requireNonNull(nullableName`)
	for _, name := range []string{"Name", "NullableName", "Count", "Total", "Latitude", "Amount", "Enabled", "Date", "Timestamp", "Identifier"} {
		assertFileDoesNotExist(t, filepath.Join(params.OutputDir, "com/sumup/sdk/models", name+".java"))
	}
	assertContains(t, read("models/Status.java"), "public static final Status ACTIVE")
	assertContains(t, read("clients/ReadersClient.java"), "public String create(")
	assertContains(t, read("clients/ReadersClient.java"), "String id")
	assertContains(t, read("clients/ReadersAsyncClient.java"), "CompletableFuture<String> create(")

	catalog, err := BuildSamples(params, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Samples) != 1 {
		t.Fatalf("expected one sample, got %d", len(catalog.Samples))
	}
	assertContains(t, catalog.Samples[0].Source, `.name("Counter 1")`)
	assertNotContains(t, catalog.Samples[0].Source, "new com.sumup.sdk.models.Name")
}
