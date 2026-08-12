package generator

import (
	"encoding/json"
	"testing"
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
