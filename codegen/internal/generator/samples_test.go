package generator

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"go.yaml.in/yaml/v4"
)

func TestBuildSamples(t *testing.T) {
	t.Parallel()
	catalog, err := BuildSamples(Params{SpecPath: "../../../openapi.json"}, "test")
	if err != nil {
		t.Fatalf("build samples: %v", err)
	}
	if catalog.SchemaVersion != 1 || catalog.Language != "java" {
		t.Fatalf("catalog metadata = %#v", catalog)
	}
	if catalog.SDK.Module != "com.sumup:sumup-sdk" || catalog.SDK.Version != "test" {
		t.Fatalf("SDK metadata = %#v", catalog.SDK)
	}
	if catalog.OpenAPIVersion == "" {
		t.Fatal("OpenAPIVersion is empty")
	}
	if len(catalog.Samples) == 0 {
		t.Fatal("catalog has no samples")
	}
	if catalog.Samples[0].ID == "" || catalog.Samples[0].Source == "" {
		t.Fatalf("incomplete sample = %#v", catalog.Samples[0])
	}
}

func TestBuildSamplesDeterministic(t *testing.T) {
	t.Parallel()
	first, err := BuildSamples(Params{SpecPath: "../../../openapi.json"}, "test")
	if err != nil {
		t.Fatalf("build first catalog: %v", err)
	}
	second, err := BuildSamples(Params{SpecPath: "../../../openapi.json"}, "test")
	if err != nil {
		t.Fatalf("build second catalog: %v", err)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if string(firstJSON) != string(secondJSON) {
		t.Fatal("sample generation is not deterministic")
	}
}

func TestJavaSampleRendererPrefersRequestExampleOverPropertyExamples(t *testing.T) {
	t.Parallel()

	propertySchema := base.CreateSchemaProxy(&base.Schema{
		Type:    []string{"string"},
		Example: &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "property-example"},
	})
	renderer := javaSampleRenderer{registry: map[string]schemaModel{
		"Request": {
			Name:       "Request",
			HasBuilder: true,
			Fields: []schemaField{
				{WireName: "selected", Name: "selected", Type: "String", Required: true, Schema: propertySchema},
				{WireName: "missing", Name: "missing", Type: "String", Required: true, Schema: propertySchema},
			},
		},
	}}

	expression := renderer.value("Request", nil, map[string]any{"selected": "request-example"}, true, true, 0)
	if !strings.Contains(expression, `.selected("request-example")`) {
		t.Fatalf("request example was not selected:\n%s", expression)
	}
	if strings.Contains(expression, "property-example") {
		t.Fatalf("renderer drilled into property examples after selecting a request example:\n%s", expression)
	}
	if !strings.Contains(expression, `.missing("example")`) {
		t.Fatalf("omitted required field did not receive a neutral fallback:\n%s", expression)
	}
}
