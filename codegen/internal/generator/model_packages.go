package generator

import (
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// collectModelPackages scopes models used by one subresource to that resource.
// Models used by multiple resources and runtime models stay in the base package.
func collectModelPackages(doc *v3.Document, params Params, parents map[string]resourceParent) map[*base.Schema]string {
	if len(parents) == 0 {
		return nil
	}
	shared := params.modelPackage()
	packages := make(map[*base.Schema]string)
	component := func(name string) *base.SchemaProxy {
		if doc.Components == nil || doc.Components.Schemas == nil {
			return nil
		}
		return doc.Components.Schemas.GetOrZero(name)
	}
	var visit func(*base.SchemaProxy, string)
	visit = func(proxy *base.SchemaProxy, pkg string) {
		schema := schemaFromProxy(proxy)
		if schema == nil {
			return
		}
		if previous := packages[schema]; previous != "" {
			if previous == pkg || previous == shared {
				return
			}
			pkg = shared
		}
		packages[schema] = pkg
		if name, ok := strings.CutPrefix(proxy.GetReference(), "#/components/schemas/"); ok {
			visit(component(name), pkg)
		}
		visitSchemaChildren(schema, func(child *base.SchemaProxy) { visit(child, pkg) })
	}

	// The handwritten transport and generated event types import these models.
	for _, name := range []string{"Problem", "Event", "EventType", "EventObject"} {
		visit(component(name), shared)
	}
	if doc.Paths != nil {
		for _, item := range doc.Paths.PathItems.FromOldest() {
			if item == nil {
				continue
			}
			for _, op := range item.GetOperations().FromOldest() {
				if op == nil {
					continue
				}
				tag := firstTag(op.Tags)
				pkg := shared
				if parents[tag].tag != "" {
					pkg += "." + resourcePackageSuffix(tag, parents)
				}
				claim := func(schema *base.SchemaProxy) { visit(schema, pkg) }
				for _, param := range item.Parameters {
					visitParameterSchemas(param, claim)
				}
				visitOperationSchemas(op, claim)
			}
		}
	}
	for _, item := range doc.Webhooks.FromOldest() {
		if item == nil || item.Post == nil {
			continue
		}
		op := item.Post
		visitOperationSchemas(op, func(schema *base.SchemaProxy) { visit(schema, shared) })
		if name, err := eventObjectSchemaName(op); err == nil {
			visit(component(name), shared)
		}
	}
	return packages
}
