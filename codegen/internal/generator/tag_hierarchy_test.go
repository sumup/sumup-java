package generator

import (
	"github.com/pb33f/libopenapi"
	"testing"
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

	sdk, err := buildModel(&model.Model, Params{BasePackage: "com.sumup.sdk"})
	if err != nil {
		t.Fatal(err)
	}
	for _, client := range sdk.Clients {
		switch client.TagName {
		case "Readers":
			if len(client.Children) != 1 || client.Children[0].AccessorName != "checkouts" || len(client.Methods) != 2 {
				t.Fatalf("expected Checkouts and both legacy operations: %+v", client)
			}
		case "ReaderCheckouts":
			if client.AccessPath != "readers().checkouts()" || len(client.Methods) != 1 {
				t.Fatalf("expected only the checkout operation on the nested client: %+v", client)
			}
		}
	}
	if len(sdk.Clients) != 2 {
		t.Fatalf("expected two clients, got %d", len(sdk.Clients))
	}
}
