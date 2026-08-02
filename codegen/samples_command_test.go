package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestReadSDKVersion(t *testing.T) {
	t.Parallel()
	filename := filepath.Join(t.TempDir(), "VERSION")
	if err := os.WriteFile(filename, []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatalf("write version: %v", err)
	}
	version, err := readSDKVersion(filename)
	if err != nil {
		t.Fatalf("read version: %v", err)
	}
	if version != "1.2.3" {
		t.Fatalf("version = %q, want 1.2.3", version)
	}
}

func TestWriteSamples(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	if err := writeSamples("", []byte("samples\n"), &stdout); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if stdout.String() != "samples\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}

	filename := filepath.Join(t.TempDir(), "nested", "samples.json")
	if err := writeSamples(filename, []byte("samples\n"), &bytes.Buffer{}); err != nil {
		t.Fatalf("write file: %v", err)
	}
	contents, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read samples: %v", err)
	}
	if string(contents) != "samples\n" {
		t.Fatalf("contents = %q", contents)
	}
}
