package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
)

func TestNestedReaderResources(t *testing.T) {
	t.Parallel()

	doc, err := libopenapi.NewDocument([]byte(`
openapi: 3.1.0
info: {title: Readers, version: 1.0.0}
tags:
  - name: Readers
  - name: ReaderCheckouts
    x-displayName: Checkouts
x-tagGroups:
  - name: Readers
    tags: [ReaderCheckouts]
paths:
  /readers/checkouts:
    get:
      operationId: ListReaders
      tags: [Readers]
      responses:
        '204': {description: OK}
    post:
      operationId: CreateReaderCheckout
      tags: [ReaderCheckouts]
      x-codegen: {method_name: create}
      responses:
        '204': {description: OK}
components: {schemas: {}}
`))
	if err != nil {
		t.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}

	params := Params{BasePackage: "com.sumup.sdk", OutputDir: t.TempDir()}
	sdk, err := buildModel(&model.Model, params)
	if err != nil {
		t.Fatal(err)
	}
	for _, client := range sdk.Clients {
		switch client.TagName {
		case "Readers":
			if len(client.Children) != 1 || client.Children[0].AccessorName != "checkouts" || len(client.Methods) != 1 || client.Methods[0].MethodName != "listReaders" {
				t.Fatalf("expected Checkouts and the parent operation: %+v", client)
			}
		case "ReaderCheckouts":
			if client.AccessPath != "readers().checkouts()" || len(client.Methods) != 1 || client.Methods[0].MethodName != "create" {
				t.Fatalf("expected only the checkout operation on the nested client: %+v", client)
			}
		}
	}
	if len(sdk.Clients) != 2 {
		t.Fatalf("expected two clients, got %d", len(sdk.Clients))
	}
	if err := renderClients(sdk, params); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(params.OutputDir, params.clientPackagePath())
	for _, variant := range []string{"Client", "AsyncClient"} {
		childClass := "ReaderCheckouts" + variant
		child, err := os.ReadFile(filepath.Join(root, "readers", "checkouts", childClass+".java"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(child), "package com.sumup.sdk.clients.readers.checkouts;") {
			t.Errorf("%s declares the wrong package", childClass)
		}
		parent, err := os.ReadFile(filepath.Join(root, "Readers"+variant+".java"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(parent), "import com.sumup.sdk.clients.readers.checkouts."+childClass+";") {
			t.Errorf("Readers%s does not import %s", variant, childClass)
		}
	}
}
