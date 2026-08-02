package generator

import (
	"encoding/json"
	"slices"
	"strings"
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
	if catalog.OpenAPIVersion != "1.0.0" {
		t.Fatalf("OpenAPIVersion = %q", catalog.OpenAPIVersion)
	}
	if len(catalog.Samples) != 47 {
		t.Fatalf("samples = %d, want 47", len(catalog.Samples))
	}
	if !slices.IsSortedFunc(catalog.Samples, func(a, b Sample) int {
		return strings.Compare(a.ID, b.ID)
	}) {
		t.Fatal("samples are not sorted by ID")
	}
	seen := make(map[string]struct{}, len(catalog.Samples))
	operations := make(map[string]struct{}, len(catalog.Samples))
	namedExamples := 0
	for _, sample := range catalog.Samples {
		if _, exists := seen[sample.ID]; exists {
			t.Fatalf("duplicate sample ID %q", sample.ID)
		}
		seen[sample.ID] = struct{}{}
		operations[sample.OperationID] = struct{}{}
		if sample.Example != "" {
			namedExamples++
		}
		if !strings.Contains(sample.Source, "public static void main(String[] args) throws Exception") {
			t.Fatalf("sample %q is not a complete Java program", sample.ID)
		}
	}
	if len(operations) != 40 || namedExamples != 9 {
		t.Fatalf("catalog coverage = %d operations and %d named examples, want 40 and 9", len(operations), namedExamples)
	}

	hosted := sampleByID(t, catalog.Samples, "CreateCheckout.HostedCheckout")
	if !strings.Contains(hosted.Source, "CheckoutCreateRequest.builder()") ||
		!strings.Contains(hosted.Source, `.checkoutReference("b50pr914-6k0e-3091-a592-890010285b3d")`) {
		t.Fatalf("named request example was not rendered:\n%s", hosted.Source)
	}
	encoded, err := json.Marshal(hosted)
	if err != nil {
		t.Fatalf("marshal sample: %v", err)
	}
	if !strings.Contains(string(encoded), `"sample":`) || strings.Contains(string(encoded), `"source":`) {
		t.Fatalf("portal JSON contract changed: %s", encoded)
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

func sampleByID(t *testing.T, samples []Sample, id string) Sample {
	t.Helper()
	for _, sample := range samples {
		if sample.ID == id {
			return sample
		}
	}
	t.Fatalf("sample %q not found", id)
	return Sample{}
}
