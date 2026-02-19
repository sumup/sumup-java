package generator

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	base "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// inlineSchemaInfo tracks metadata about inline schemas discovered while
// walking through OpenAPI operations and models.
type inlineSchemaInfo struct {
	className   string
	schema      *base.Schema
	description string
	processed   bool
}

// typeResolver derives Java-friendly type information from OpenAPI schema
// references. It keeps track of inline models so they can be rendered later.
type typeResolver struct {
	params          Params
	schemaTypes     map[string]string
	inlineTypes     map[*base.Schema]string
	inlineSchemas   []*inlineSchemaInfo
	inlineNameUsage map[string]int
}

// newTypeResolver returns a resolver that knows how to reference components and
// captures inline schema names as it uncovers them.
func newTypeResolver(doc *v3.Document, params Params) *typeResolver {
	resolver := &typeResolver{
		params:          params,
		schemaTypes:     make(map[string]string),
		inlineTypes:     make(map[*base.Schema]string),
		inlineSchemas:   []*inlineSchemaInfo{},
		inlineNameUsage: make(map[string]int),
	}
	if doc.Components != nil && doc.Components.Schemas != nil && doc.Components.Schemas.Len() > 0 {
		for name := range doc.Components.Schemas.KeysFromOldest() {
			className := pascalCase(name, "")
			resolver.schemaTypes[name] = params.modelPackage() + "." + className
			resolver.inlineNameUsage[className]++
		}
	}
	return resolver
}

// schemaClassName resolves a schema component name to the fully-qualified Java
// class name, registering it as inline if the component does not exist.
func (r *typeResolver) schemaClassName(name string) string {
	if cls, ok := r.schemaTypes[name]; ok {
		return cls
	}
	className := pascalCase(name, "")
	r.inlineNameUsage[className]++
	return r.params.modelPackage() + "." + className
}

// javaType converts an OpenAPI schema reference into a Java type, optionally
// recording extra inline models that must be rendered.
type javaType struct {
	Name        string
	Imports     []string
	TypeRefExpr string
	IsVoid      bool
}

// javaType infers the Java representation for the provided schema proxy,
// falling back to generic map/list types whenever the schema cannot be mapped
// to a concrete SumUp model.
func (r *typeResolver) javaType(ref *base.SchemaProxy, context ...string) javaType {
	if ref == nil {
		return r.genericMap()
	}
	if ref.IsReference() {
		name := componentNameFromRef(ref.GetReference())
		if name != "" {
			fqn := r.schemaClassName(name)
			return javaType{
				Name:        fqn,
				TypeRefExpr: fmt.Sprintf("new TypeReference<%s>() {}", fqn),
			}
		}
	}
	schema := ref.Schema()
	if schema == nil {
		return r.genericMap()
	}
	if enumValues, ok := enumValuesForSchema(schema); ok && len(enumValues) > 0 {
		name := r.registerInlineSchema(schema, context)
		fqn := r.params.modelPackage() + "." + name
		return javaType{
			Name:        fqn,
			TypeRefExpr: fmt.Sprintf("new TypeReference<%s>() {}", fqn),
		}
	}
	switch {
	case schemaHasType(schema, "string"):
		return r.stringType(schema)
	case schemaHasType(schema, "integer"):
		return r.integerType(schema)
	case schemaHasType(schema, "number"):
		return r.numberType(schema)
	case schemaHasType(schema, "boolean"):
		return javaType{
			Name:        "Boolean",
			TypeRefExpr: "new com.fasterxml.jackson.core.type.TypeReference<Boolean>() {}",
		}
	case schemaHasType(schema, "array") || (schema.Items != nil && schema.Items.IsA()):
		if schema.Items != nil && schema.Items.IsA() && schema.Items.A != nil {
			elem := r.javaType(schema.Items.A, append(context, "Item")...)
			imports := append([]string{}, elem.Imports...)
			imports = append(imports, "java.util.List")
			name := fmt.Sprintf("java.util.List<%s>", elem.Name)
			return javaType{
				Name:        name,
				Imports:     uniqueStrings(imports),
				TypeRefExpr: fmt.Sprintf("new TypeReference<%s>() {}", name),
			}
		}
		return r.genericList()
	case schemaHasType(schema, "object"):
		return r.objectType(schema, context)
	}

	if schema.AdditionalProperties != nil {
		if schema.AdditionalProperties.IsA() && schema.AdditionalProperties.A != nil {
			valueType := r.javaType(schema.AdditionalProperties.A, append(context, "Value")...)
			imports := append([]string{}, valueType.Imports...)
			imports = append(imports, "java.util.Map")
			name := fmt.Sprintf("java.util.Map<String, %s>", valueType.Name)
			return javaType{
				Name:        name,
				Imports:     uniqueStrings(imports),
				TypeRefExpr: fmt.Sprintf("new TypeReference<%s>() {}", name),
			}
		}
		if schema.AdditionalProperties.IsB() && schema.AdditionalProperties.B {
			return r.genericMap()
		}
	}
	if schema.Properties != nil && schema.Properties.Len() > 0 {
		return r.inlineObjectType(schema, context)
	}
	if len(schema.AllOf) > 0 {
		for _, item := range schema.AllOf {
			if item == nil {
				continue
			}
			return r.javaType(item, context...)
		}
	}
	if len(schema.OneOf) > 0 {
		for _, item := range schema.OneOf {
			if item == nil {
				continue
			}
			return r.javaType(item, context...)
		}
	}
	if len(schema.AnyOf) > 0 {
		for _, item := range schema.AnyOf {
			if item == nil {
				continue
			}
			return r.javaType(item, context...)
		}
	}
	return r.genericMap()
}

// objectType handles schemas that look like objects by either emitting inline
// models or falling back to generic map types.
func (r *typeResolver) objectType(schema *base.Schema, context []string) javaType {
	if schema.AdditionalProperties != nil {
		if schema.AdditionalProperties.IsA() && schema.AdditionalProperties.A != nil {
			valueType := r.javaType(schema.AdditionalProperties.A, append(context, "Value")...)
			imports := append([]string{}, valueType.Imports...)
			imports = append(imports, "java.util.Map")
			name := fmt.Sprintf("java.util.Map<String, %s>", valueType.Name)
			return javaType{
				Name:        name,
				Imports:     uniqueStrings(imports),
				TypeRefExpr: fmt.Sprintf("new TypeReference<%s>() {}", name),
			}
		}
		if schema.AdditionalProperties.IsB() && schema.AdditionalProperties.B {
			return r.genericMap()
		}
	}
	if (schema.Properties == nil || schema.Properties.Len() == 0) && len(schema.AllOf) == 0 {
		return r.genericMap()
	}
	return r.inlineObjectType(schema, context)
}

// genericList is the fallback javaType for arrays with unknown element types.
func (r *typeResolver) genericList() javaType {
	return javaType{
		Name:        "java.util.List<Object>",
		Imports:     []string{"java.util.List"},
		TypeRefExpr: "new TypeReference<java.util.List<Object>>() {}",
	}
}

// inlineObjectType registers the schema as an inline model and returns its
// fully-qualified name for use in generated sources.
func (r *typeResolver) inlineObjectType(schema *base.Schema, context []string) javaType {
	name := r.registerInlineSchema(schema, context)
	fqn := r.params.modelPackage() + "." + name
	return javaType{
		Name:        fqn,
		TypeRefExpr: fmt.Sprintf("new TypeReference<%s>() {}", fqn),
	}
}

// registerInlineSchema stores schemas that should become top-level models and
// assigns them deterministic, unique names.
func (r *typeResolver) registerInlineSchema(schema *base.Schema, context []string) string {
	if existing, ok := r.inlineTypes[schema]; ok {
		return existing
	}

	baseName := strings.TrimSpace(schema.Title)
	if baseName == "" && len(context) > 0 {
		baseName = strings.Join(context, " ")
	}
	if baseName == "" {
		baseName = "InlineObject"
	}
	className := pascalCase(baseName, "")
	if count := r.inlineNameUsage[className]; count > 0 {
		className = fmt.Sprintf("%s%d", className, count+1)
	}
	r.inlineNameUsage[className]++

	r.inlineTypes[schema] = className
	r.inlineSchemas = append(r.inlineSchemas, &inlineSchemaInfo{
		className:   className,
		schema:      schema,
		description: schema.Description,
	})
	return className
}

// stringType returns the Java representation for OpenAPI string schemas.
func (r *typeResolver) stringType(schema *base.Schema) javaType {
	switch schema.Format {
	case "date-time":
		return javaType{
			Name:        "java.time.OffsetDateTime",
			Imports:     []string{"java.time.OffsetDateTime"},
			TypeRefExpr: "new TypeReference<java.time.OffsetDateTime>() {}",
		}
	case "date":
		return javaType{
			Name:        "java.time.LocalDate",
			Imports:     []string{"java.time.LocalDate"},
			TypeRefExpr: "new TypeReference<java.time.LocalDate>() {}",
		}
	case "uuid":
		return javaType{
			Name:        "java.util.UUID",
			Imports:     []string{"java.util.UUID"},
			TypeRefExpr: "new TypeReference<java.util.UUID>() {}",
		}
	default:
		return javaType{
			Name:        "String",
			TypeRefExpr: "new TypeReference<String>() {}",
		}
	}
}

// integerType returns the boxed Java number type for integer schemas.
func (r *typeResolver) integerType(schema *base.Schema) javaType {
	switch schema.Format {
	case "int32":
		return javaType{
			Name:        "Integer",
			TypeRefExpr: "new TypeReference<Integer>() {}",
		}
	default:
		return javaType{
			Name:        "Long",
			TypeRefExpr: "new TypeReference<Long>() {}",
		}
	}
}

// numberType maps floating point schemas to their Java equivalents.
func (r *typeResolver) numberType(schema *base.Schema) javaType {
	switch schema.Format {
	case "float":
		return javaType{
			Name:        "Float",
			TypeRefExpr: "new TypeReference<Float>() {}",
		}
	default:
		return javaType{
			Name:        "Double",
			TypeRefExpr: "new TypeReference<Double>() {}",
		}
	}
}

// genericMap returns the canonical fallback map type for unknown schema shapes.
func (r *typeResolver) genericMap() javaType {
	return javaType{
		Name:        "java.util.Map<String, Object>",
		Imports:     []string{"java.util.Map"},
		TypeRefExpr: "new TypeReference<java.util.Map<String, Object>>() {}",
	}
}

// inlineSchemaModels converts the inline schemas tracked by the resolver into
// schemaModel values so the renderer can emit classes for them.
func (r *typeResolver) inlineSchemaModels(params Params) []schemaModel {
	var models []schemaModel
	for {
		added := false
		for _, info := range r.inlineSchemas {
			if info.processed {
				continue
			}
			if enumValues, ok := enumValuesForSchema(info.schema); ok {
				imports := sortedImports(map[string]struct{}{
					"com.fasterxml.jackson.annotation.JsonCreator": {},
					"com.fasterxml.jackson.annotation.JsonValue":   {},
				})
				models = append(models, schemaModel{
					Name:             info.className,
					ClassName:        info.className,
					Package:          params.modelPackage(),
					DescriptionLines: splitComment(info.description),
					Imports:          imports,
					IsEnum:           true,
					EnumValues:       enumValues,
				})
				info.processed = true
				added = true
				continue
			}
			fields, additionalProps, imports, hasRequired := buildSchemaFields(info.className, base.CreateSchemaProxy(info.schema), r)
			if additionalProps != nil {
				imports = withAdditionalPropertiesImports(imports)
			}
			hasBuilder := shouldGenerateBuilder(fields, additionalProps)
			if hasRequired && hasBuilder {
				imports = uniqueStrings(append(imports, "java.util.Objects"))
			}
			models = append(models, schemaModel{
				Name:             info.className,
				ClassName:        info.className,
				Package:          params.modelPackage(),
				DescriptionLines: splitComment(info.description),
				Fields:           fields,
				AdditionalProps:  additionalProps,
				Imports:          imports,
				HasRequired:      hasRequired,
				HasBuilder:       hasBuilder,
			})
			info.processed = true
			added = true
		}
		if !added {
			break
		}
	}
	return models
}

// componentNameFromRef extracts the schema component name from a JSON pointer.
func componentNameFromRef(ref string) string {
	if ref == "" {
		return ""
	}
	baseName := filepath.Base(ref)
	if baseName != "" {
		return baseName
	}
	parts := strings.Split(ref, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// uniqueStrings returns a sorted set of strings while preserving only distinct
// values. It is used heavily for import lists so generated files remain tidy.
func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	sort.Strings(values)
	out := make([]string, 0, len(values))
	var last string
	for i, v := range values {
		if i == 0 || v != last {
			out = append(out, v)
			last = v
		}
	}
	return out
}
