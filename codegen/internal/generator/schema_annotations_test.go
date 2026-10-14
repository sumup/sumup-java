package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAnnotatedSchemas(t *testing.T) {
	t.Parallel()
	out := t.TempDir()
	params := Params{SpecPath: "testdata/schema-annotations.json", OutputDir: out, ResourceDir: filepath.Join(out, "resources")}
	if err := Run(t.Context(), params); err != nil {
		t.Fatal(err)
	}
	checks := map[string][]string{
		"com/sumup/sdk/models/Address.java":         {"String country"},
		"com/sumup/sdk/models/Merchant.java":        {"String phoneNumber"},
		"com/sumup/sdk/models/ExtendedAddress.java": {"String country", "String label"},
	}
	for file, snippets := range checks {
		data, err := os.ReadFile(filepath.Join(out, file))
		if err != nil {
			t.Fatal(err)
		}
		for _, snippet := range snippets {
			if !strings.Contains(string(data), snippet) {
				t.Errorf("%s missing %q", file, snippet)
			}
		}
	}
}
