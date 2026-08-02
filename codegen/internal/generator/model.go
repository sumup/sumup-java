package generator

import (
	"fmt"
	"sort"
	"strings"

	base "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

// sdkModel represents the aggregate information needed to render the SDK.
type sdkModel struct {
	BasePackage   string
	ClientPackage string
	ModelPackage  string
	Clients       []clientModel
	Schemas       []schemaModel
}

// clientModel encapsulates the data necessary to render a Java API client.
type clientModel struct {
	TagName             string
	TagDescriptionLines []string
	ClassName           string
	AccessorName        string
	FieldName           string
	Package             string
	Methods             []operationModel
}

// operationModel stores the derived metadata for one OpenAPI operation.
type operationModel struct {
	OperationID          string
	MethodName           string
	SummaryLines         []string
	DescriptionLines     []string
	HttpMethod           string
	Path                 string
	PathParams           []parameterModel
	RequiredQueryParams  []parameterModel
	OptionalQueryParams  []parameterModel
	RequiredHeaderParams []parameterModel
	OptionalHeaderParams []parameterModel
	QueryStruct          *parameterGroupModel
	HeaderStruct         *parameterGroupModel
	RequestBodyType      javaType
	RequestRequired      bool
	RequestDescription   string
	HasRequestBody       bool
	ResponseType         javaType
	HasQueryParams       bool
	HasOptionalQuery     bool
	HasHeaderParams      bool
	HasOptionalHeaders   bool
	HasOptionalArgs      bool
	TagName              string
	Operation            *v3.Operation
	RequestSchema        *base.SchemaProxy
}

// parameterGroupModel holds information about optional parameter structs
// generated for query or header bags.
type parameterGroupModel struct {
	Kind        string
	ClassName   string
	VarName     string
	MapValue    string
	Fields      []parameterModel
	ImportType  string
	Description string
}

// parameterModel describes a single operation argument.
type parameterModel struct {
	Name        string
	FieldName   string
	Description string
	Required    bool
	Type        javaType
	Location    string
	Schema      *base.SchemaProxy
	Parameter   *v3.Parameter
}

// schemaModel represents the information required to render a POJO model.
type schemaModel struct {
	Name             string
	ClassName        string
	Package          string
	DescriptionLines []string
	Fields           []schemaField
	AdditionalProps  *additionalPropertiesModel
	Imports          []string
	HasRequired      bool
	HasBuilder       bool
	IsEnum           bool
	EnumValues       []enumValueModel
}

// schemaField stores metadata for a field within a schemaModel.
type schemaField struct {
	WireName         string
	Name             string
	Type             string
	DescriptionLines []string
	Required         bool
	ReadOnly         bool
	Schema           *base.SchemaProxy
}

// additionalPropertiesModel describes synthetic map storage used when object
// schemas define both fixed properties and additionalProperties.
type additionalPropertiesModel struct {
	FieldName        string
	Type             string
	ValueType        string
	DescriptionLines []string
}

// enumValueModel captures a single enum constant and its wire value.
type enumValueModel struct {
	Name      string
	WireValue string
}

const (
	parameterInPath   = "path"
	parameterInQuery  = "query"
	parameterInHeader = "header"
)

// buildModel walks the OpenAPI document and produces the aggregated SDK model
// used by renderers to generate source files.
func buildModel(doc *v3.Document, params Params) (sdkModel, error) {
	resolver := newTypeResolver(doc, params)
	tagDescriptions := map[string][]string{}
	if doc.Tags != nil {
		for _, tag := range doc.Tags {
			if tag == nil {
				continue
			}
			tagDescriptions[firstTag([]string{tag.Name})] = splitComment(strings.TrimSpace(tag.Description))
		}
	}
	groups := map[string][]operationModel{}
	if doc.Paths != nil && doc.Paths.PathItems != nil && doc.Paths.PathItems.Len() > 0 {
		for path, item := range doc.Paths.PathItems.FromOldest() {
			if item == nil {
				continue
			}
			ops := item.GetOperations()
			if ops == nil {
				continue
			}
			for method, operation := range ops.FromOldest() {
				if operation == nil {
					continue
				}
				opModel := convertOperation(method, path, item, operation, resolver)
				tag := opModel.TagName
				groups[tag] = append(groups[tag], opModel)
			}
		}
	}

	tagNames := make([]string, 0, len(groups))
	for tag := range groups {
		tagNames = append(tagNames, tag)
	}
	sort.Strings(tagNames)

	clients := make([]clientModel, 0, len(tagNames))
	accessorUsage := map[string]int{}

	for _, tag := range tagNames {
		ops := groups[tag]
		sort.SliceStable(ops, func(i, j int) bool {
			if ops[i].MethodName == ops[j].MethodName {
				return ops[i].OperationID < ops[j].OperationID
			}
			return ops[i].MethodName < ops[j].MethodName
		})
		ensureUniqueMethodNames(ops)

		className := pascalCase(tag, "Client")
		accessor := camelCase(tag, "client")
		count := accessorUsage[accessor]
		accessorUsage[accessor] = count + 1
		fieldName := accessor
		if count > 0 {
			suffix := fmt.Sprintf("%d", count+1)
			accessor += suffix
			fieldName += suffix
			className += suffix
		}

		clients = append(clients, clientModel{
			TagName:             tag,
			TagDescriptionLines: tagDescriptions[tag],
			ClassName:           className,
			AccessorName:        accessor,
			FieldName:           fieldName,
			Package:             params.clientPackage(),
			Methods:             ops,
		})
	}

	schemas := buildSchemas(doc, params, resolver)
	schemas = append(schemas, resolver.inlineSchemaModels(params)...)

	return sdkModel{
		BasePackage:   params.BasePackage,
		ClientPackage: params.clientPackage(),
		ModelPackage:  params.modelPackage(),
		Clients:       clients,
		Schemas:       schemas,
	}, nil
}

// ensureUniqueMethodNames deduplicates generated method names within a client
// by appending numeric suffixes when duplicates surface.
func ensureUniqueMethodNames(ops []operationModel) {
	counts := make(map[string]int, len(ops))
	for i := range ops {
		name := ops[i].MethodName
		count := counts[name]
		if count > 0 {
			ops[i].MethodName = fmt.Sprintf("%s%d", name, count+1)
		}
		counts[name] = count + 1
	}
}

// convertOperation translates an OpenAPI operation into the normalized
// operationModel used by templates.
func convertOperation(method, path string, item *v3.PathItem, op *v3.Operation, resolver *typeResolver) operationModel {
	sanitizedID := sanitizeOperationID(op.OperationId)
	model := operationModel{
		OperationID:      sanitizedID,
		MethodName:       operationMethodName(op, sanitizedID),
		SummaryLines:     splitComment(strings.TrimSpace(op.Summary)),
		DescriptionLines: splitComment(strings.TrimSpace(op.Description)),
		HttpMethod:       strings.ToUpper(method),
		Path:             path,
		TagName:          firstTag(op.Tags),
		Operation:        op,
	}

	params := collectParameters(item, op)
	model.PathParams = filterParams(params, parameterInPath, resolver, sanitizedID)

	queryParams := filterParams(params, parameterInQuery, resolver, sanitizedID)
	model.RequiredQueryParams, model.OptionalQueryParams = splitParams(queryParams)
	if len(model.OptionalQueryParams) > 0 {
		model.QueryStruct = &parameterGroupModel{
			Kind:        parameterInQuery,
			ClassName:   pascalCase(sanitizedID, "QueryParams"),
			VarName:     camelCase(sanitizedID, "queryParams"),
			MapValue:    "Object",
			ImportType:  "java.util.Map<String, Object>",
			Fields:      model.OptionalQueryParams,
			Description: parameterGroupDescription(parameterInQuery),
		}
		model.HasOptionalQuery = true
	}
	model.HasQueryParams = len(model.RequiredQueryParams) > 0 || len(model.OptionalQueryParams) > 0

	headerParams := filterParams(params, parameterInHeader, resolver, sanitizedID)
	model.RequiredHeaderParams, model.OptionalHeaderParams = splitParams(headerParams)
	if len(model.OptionalHeaderParams) > 0 {
		model.HeaderStruct = &parameterGroupModel{
			Kind:        parameterInHeader,
			ClassName:   pascalCase(sanitizedID, "Headers"),
			VarName:     camelCase(sanitizedID, "headers"),
			MapValue:    "String",
			ImportType:  "java.util.Map<String, String>",
			Fields:      model.OptionalHeaderParams,
			Description: parameterGroupDescription(parameterInHeader),
		}
		model.HasOptionalHeaders = true
	}
	model.HasHeaderParams = len(model.RequiredHeaderParams) > 0 || len(model.OptionalHeaderParams) > 0

	if op.RequestBody != nil {
		model.RequestSchema = preferredSchema(op.RequestBody.Content)
		model.RequestBodyType = schemaTypeFromContent(op.RequestBody, resolver, sanitizedID, "Request")
		model.RequestRequired = op.RequestBody.Required != nil && *op.RequestBody.Required
		model.RequestDescription = normalizeText(op.RequestBody.Description)
		model.HasRequestBody = !model.RequestBodyType.IsVoid
	}

	model.ResponseType = responseTypeFromResponses(op.Responses, resolver, sanitizedID, "Response")
	model.HasOptionalArgs = model.HasOptionalQuery || model.HasOptionalHeaders

	return model
}

// firstTag returns the first non-empty tag or a default fallback.
func firstTag(tags []string) string {
	if len(tags) == 0 || tags[0] == "" {
		return "core"
	}
	return tags[0]
}

// sanitizeOperationID ensures every operation has an identifier used to derive
// class and method names.
func sanitizeOperationID(operationID string) string {
	if operationID == "" {
		return "operation"
	}
	return operationID
}

// operationMethodName returns the Java method name for an OpenAPI operation.
// When present, x-codegen.method_name is preferred because generated methods
// are already scoped to their tag client.
func operationMethodName(op *v3.Operation, fallbackID string) string {
	if methodName := codegenMethodName(op); methodName != "" {
		return camelCase(methodName, "operation")
	}
	return camelCase(fallbackID, "operation")
}

func codegenMethodName(op *v3.Operation) string {
	if op == nil || op.Extensions == nil {
		return ""
	}
	node := op.Extensions.GetOrZero("x-codegen")
	if node == nil {
		return ""
	}
	var extension struct {
		MethodName string `yaml:"method_name"`
	}
	if err := node.Decode(&extension); err != nil {
		return ""
	}
	return strings.TrimSpace(extension.MethodName)
}

// collectParameters merges operation and path-level parameters, filtering out
// nil references along the way.
func collectParameters(item *v3.PathItem, op *v3.Operation) []*v3.Parameter {
	var result []*v3.Parameter
	appendParams := func(params []*v3.Parameter) {
		for _, param := range params {
			if param == nil {
				continue
			}
			result = append(result, param)
		}
	}
	if item != nil {
		appendParams(item.Parameters)
	}
	if op != nil {
		appendParams(op.Parameters)
	}
	return result
}

// filterParams keeps parameters that match the requested location and converts
// them into parameterModel values.
func filterParams(params []*v3.Parameter, location string, resolver *typeResolver, operationContext string) []parameterModel {
	var filtered []parameterModel
	seen := map[string]struct{}{}
	for _, param := range params {
		if param == nil || param.In != location {
			continue
		}
		name := strings.TrimSpace(param.Name)
		if name == "" {
			continue
		}
		if location == parameterInHeader && shouldIgnoreHeader(name) {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		schemaRef := parameterSchema(param)
		javaType := resolver.parameterJavaType(schemaRef, operationContext, name)
		required := param.Required != nil && *param.Required
		filtered = append(filtered, parameterModel{
			Name:        name,
			FieldName:   camelCase(name, name),
			Description: normalizeText(param.Description),
			Required:    required,
			Type:        javaType,
			Location:    location,
			Schema:      schemaRef,
			Parameter:   param,
		})
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})
	return filtered
}

// shouldIgnoreHeader hides headers that are already injected by the HTTP layer
// so callers are not forced to duplicate them.
func shouldIgnoreHeader(name string) bool {
	switch strings.ToLower(name) {
	case "authorization", "accept", "content-type":
		return true
	default:
		return false
	}
}

// parameterGroupDescription returns a user-friendly description for optional
// query or header parameter groups.
func parameterGroupDescription(kind string) string {
	switch kind {
	case parameterInQuery:
		return "Optional query parameters for this request."
	case parameterInHeader:
		return "Optional header overrides for this request."
	default:
		return "Optional parameters for this request."
	}
}

// splitParams separates required and optional parameters to simplify template
// rendering logic.
func splitParams(params []parameterModel) (required, optional []parameterModel) {
	for _, param := range params {
		if param.Required {
			required = append(required, param)
		} else {
			optional = append(optional, param)
		}
	}
	return
}

// parameterSchema extracts the schema describing a parameter, checking both the
// top-level schema and any media types it may define.
func parameterSchema(param *v3.Parameter) *base.SchemaProxy {
	if param == nil {
		return nil
	}
	if param.Schema != nil {
		return param.Schema
	}
	if param.Content == nil || param.Content.Len() == 0 {
		return nil
	}
	mediaTypes := make([]string, 0, param.Content.Len())
	for media := range param.Content.KeysFromOldest() {
		mediaTypes = append(mediaTypes, media)
	}
	sort.Strings(mediaTypes)
	for _, media := range mediaTypes {
		if mediaSchema := param.Content.GetOrZero(media); mediaSchema != nil && mediaSchema.Schema != nil {
			return mediaSchema.Schema
		}
	}
	return nil
}

// schemaTypeFromContent resolves the preferred schema for a request body,
// defaulting to a generic map if nothing concrete is available.
func schemaTypeFromContent(body *v3.RequestBody, resolver *typeResolver, context ...string) javaType {
	if body == nil || body.Content == nil || body.Content.Len() == 0 {
		return javaType{IsVoid: true}
	}
	if schemaRef := preferredSchema(body.Content); schemaRef != nil {
		return resolver.javaType(schemaRef, context...)
	}
	return javaType{
		Name:        "java.util.Map<String, Object>",
		Imports:     []string{"java.util.Map"},
		TypeRefExpr: "new TypeReference<java.util.Map<String, Object>>() {}",
	}
}

// responseTypeFromResponses walks every 2xx response and determines the type we
// should deserialize into. Anything else yields void.
func responseTypeFromResponses(responses *v3.Responses, resolver *typeResolver, context ...string) javaType {
	if responses == nil || responses.Codes == nil || responses.Codes.Len() == 0 {
		return javaType{IsVoid: true}
	}
	statuses := make([]string, 0, responses.Codes.Len())
	for status := range responses.Codes.KeysFromOldest() {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	for _, status := range statuses {
		if !strings.HasPrefix(status, "2") {
			continue
		}
		if status == "204" {
			return javaType{IsVoid: true}
		}
		resp := responses.Codes.GetOrZero(status)
		if resp == nil || resp.Content == nil || resp.Content.Len() == 0 {
			continue
		}
		if schemaRef := preferredSchema(resp.Content); schemaRef != nil {
			return resolver.javaType(schemaRef, context...)
		}
	}
	if responses.Default != nil && responses.Default.Content != nil {
		if schemaRef := preferredSchema(responses.Default.Content); schemaRef != nil {
			return resolver.javaType(schemaRef, context...)
		}
	}
	return javaType{IsVoid: true}
}

// splitComment converts raw description strings into trimmed comment lines.
func splitComment(value string) []string {
	if value == "" {
		return nil
	}
	lines := strings.Split(value, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

// normalizeText collapses whitespace so inline descriptions render cleanly in
// generated JavaDoc.
func normalizeText(value string) string {
	if value == "" {
		return ""
	}
	replaced := strings.NewReplacer("\n", " ", "\t", " ").Replace(value)
	return strings.Join(strings.Fields(replaced), " ")
}

// preferredSchema picks the schema we should use when multiple media types are
// described for a request or response body.
func preferredSchema(content *orderedmap.Map[string, *v3.MediaType]) *base.SchemaProxy {
	if content == nil || content.Len() == 0 {
		return nil
	}
	if media := content.GetOrZero("application/problem+json"); media != nil && media.Schema != nil {
		return media.Schema
	}
	if media := content.GetOrZero("application/json"); media != nil && media.Schema != nil {
		return media.Schema
	}
	return nil
}

// buildSchemas converts OpenAPI components into schemaModel instances.
func buildSchemas(doc *v3.Document, params Params, resolver *typeResolver) []schemaModel {
	if doc.Components == nil || doc.Components.Schemas == nil || doc.Components.Schemas.Len() == 0 {
		return nil
	}
	names := make([]string, 0, doc.Components.Schemas.Len())
	for name := range doc.Components.Schemas.KeysFromOldest() {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]schemaModel, 0, len(names))
	for _, name := range names {
		ref := doc.Components.Schemas.GetOrZero(name)
		if ref == nil {
			continue
		}
		description := schemaDescription(ref)
		if enumValues, ok := enumValuesForSchema(schemaFromProxy(ref)); ok {
			imports := sortedImports(map[string]struct{}{
				"com.fasterxml.jackson.annotation.JsonCreator": {},
				"com.fasterxml.jackson.annotation.JsonValue":   {},
				"java.util.Objects":                            {},
			})
			result = append(result, schemaModel{
				Name:             name,
				ClassName:        pascalCase(name, ""),
				Package:          params.modelPackage(),
				DescriptionLines: splitComment(description),
				Imports:          imports,
				IsEnum:           true,
				EnumValues:       enumValues,
			})
			continue
		}
		fields, additionalProps, imports, hasRequired := buildSchemaFields(name, ref, resolver)
		if additionalProps != nil {
			imports = withAdditionalPropertiesImports(imports)
		}
		hasBuilder := shouldGenerateBuilder(fields, additionalProps)
		if hasRequired && hasBuilder {
			imports = uniqueStrings(append(imports, "java.util.Objects"))
		}
		model := schemaModel{
			Name:             name,
			ClassName:        pascalCase(name, ""),
			Package:          params.modelPackage(),
			DescriptionLines: splitComment(description),
			Fields:           fields,
			AdditionalProps:  additionalProps,
			Imports:          imports,
			HasRequired:      hasRequired,
			HasBuilder:       hasBuilder,
		}
		result = append(result, model)
	}
	return result
}

// shouldGenerateBuilder reports whether the model should expose a builder.
// Single-field wrapper records (for example Lon/Lat/Meta-style aliases) don't
// benefit from a builder and should use the canonical record constructor.
func shouldGenerateBuilder(fields []schemaField, additionalProps *additionalPropertiesModel) bool {
	if additionalProps != nil {
		return true
	}
	return !(len(fields) == 1 && fields[0].Name == "value")
}

// buildSchemaFields inspects a schema proxy and returns the fields, required
// imports, and a flag indicating whether any required properties exist.
func buildSchemaFields(name string, ref *base.SchemaProxy, resolver *typeResolver) ([]schemaField, *additionalPropertiesModel, []string, bool) {
	imports := map[string]struct{}{}
	var fields []schemaField
	var additionalProps *additionalPropertiesModel
	hasRequired := false

	if ref == nil {
		return []schemaField{{Name: "value", Type: "Object"}}, nil, nil, false
	}

	schema := ref.Schema()
	if schema == nil {
		return []schemaField{{Name: "value", Type: "Object"}}, nil, nil, false
	}
	if schemaDefinesModelFields(schema) {
		props := collectProperties(schema)
		if len(props) == 0 {
			return []schemaField{{Name: "value", Type: "java.util.Map<String, Object>"}}, nil, []string{"java.util.Map"}, false
		}
		required := collectRequired(schema)
		fields = make([]schemaField, 0, len(props))
		names := make([]string, 0, len(props))
		for prop := range props {
			names = append(names, prop)
		}
		sort.Strings(names)
		for _, propName := range names {
			propRef := props[propName]
			javaType := resolver.javaType(propRef, name, propName)
			for _, imp := range javaType.Imports {
				imports[imp] = struct{}{}
			}
			readOnly := isReadOnlySchema(propRef)
			desc := ""
			if schemaFromProxy(propRef) != nil {
				desc = schemaFromProxy(propRef).Description
			}
			fields = append(fields, schemaField{
				WireName:         propName,
				Name:             camelCase(propName, propName),
				Type:             javaType.Name,
				DescriptionLines: splitComment(desc),
				Required:         required[propName],
				ReadOnly:         readOnly,
				Schema:           propRef,
			})
			if required[propName] && !readOnly {
				hasRequired = true
			}
		}
		additionalProps = resolveAdditionalProperties(schema, resolver, name)
		if additionalProps != nil {
			imports["java.util.Map"] = struct{}{}
			valueTypeRef := additionalPropertiesTypedSchema(schema)
			if valueTypeRef != nil {
				valueJavaType := resolver.javaType(valueTypeRef, name, "AdditionalProperty")
				for _, imp := range valueJavaType.Imports {
					imports[imp] = struct{}{}
				}
			}
			fields = append(fields, schemaField{
				Name:             additionalProps.FieldName,
				Type:             additionalProps.Type,
				DescriptionLines: additionalProps.DescriptionLines,
				Required:         false,
			})
		}
	} else {
		javaType := resolver.javaType(ref, name)
		for _, imp := range javaType.Imports {
			imports[imp] = struct{}{}
		}
		fields = []schemaField{{
			Name: "value",
			Type: javaType.Name,
		}}
	}

	return fields, additionalProps, sortedImports(imports), hasRequired
}

func schemaDefinesModelFields(schema *base.Schema) bool {
	if schema == nil {
		return false
	}
	return schemaHasType(schema, "object") || schemaHasFlattenableProperties(schema)
}

func schemaHasFlattenableProperties(schema *base.Schema) bool {
	if schema == nil {
		return false
	}
	if schema.Properties != nil && schema.Properties.Len() > 0 {
		return true
	}
	for _, item := range schema.AllOf {
		if item == nil {
			continue
		}
		if schemaHasFlattenableProperties(item.Schema()) {
			return true
		}
	}
	return false
}

// resolveAdditionalProperties returns metadata for schemas that allow
// additional fields alongside declared properties.
func resolveAdditionalProperties(schema *base.Schema, resolver *typeResolver, context ...string) *additionalPropertiesModel {
	valueType, ok := additionalPropertiesValueType(schema, resolver, context...)
	if !ok {
		return nil
	}
	return &additionalPropertiesModel{
		FieldName:        "additionalProperties",
		Type:             fmt.Sprintf("java.util.Map<String, %s>", valueType),
		ValueType:        valueType,
		DescriptionLines: splitComment("Additional fields not described by the fixed schema properties."),
	}
}

// additionalPropertiesValueType reports whether additionalProperties is enabled
// and, when enabled, the expected Java value type.
func additionalPropertiesValueType(schema *base.Schema, resolver *typeResolver, context ...string) (string, bool) {
	if schema == nil {
		return "", false
	}
	if schema.AdditionalProperties != nil {
		if schema.AdditionalProperties.IsA() && schema.AdditionalProperties.A != nil {
			valueJavaType := resolver.javaType(schema.AdditionalProperties.A, append(context, "AdditionalProperty")...)
			return valueJavaType.Name, true
		}
		if schema.AdditionalProperties.IsB() && schema.AdditionalProperties.B {
			return "Object", true
		}
	}
	for _, item := range schema.AllOf {
		if item == nil {
			continue
		}
		if valueType, ok := additionalPropertiesValueType(item.Schema(), resolver, context...); ok {
			return valueType, true
		}
	}
	return "", false
}

// additionalPropertiesTypedSchema returns a schema reference when
// additionalProperties defines a concrete value schema.
func additionalPropertiesTypedSchema(schema *base.Schema) *base.SchemaProxy {
	if schema == nil {
		return nil
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsA() && schema.AdditionalProperties.A != nil {
		return schema.AdditionalProperties.A
	}
	for _, item := range schema.AllOf {
		if item == nil {
			continue
		}
		if nested := additionalPropertiesTypedSchema(item.Schema()); nested != nil {
			return nested
		}
	}
	return nil
}

// enumValuesForSchema extracts enum values for string schemas.
func enumValuesForSchema(schema *base.Schema) ([]enumValueModel, bool) {
	if schema == nil || len(schema.Enum) == 0 {
		return nil, false
	}
	if len(schema.Type) > 0 && !schemaHasType(schema, "string") {
		return nil, false
	}
	if len(schema.Type) == 0 && !schemaHasType(schema, "string") {
		for _, node := range schema.Enum {
			if node == nil || node.Tag != "!!str" {
				return nil, false
			}
		}
	}

	values := make([]enumValueModel, 0, len(schema.Enum))
	usedNames := map[string]int{}
	for _, node := range schema.Enum {
		if node == nil {
			continue
		}
		raw := node.Value
		if raw == "" {
			var decoded string
			if err := node.Decode(&decoded); err == nil {
				raw = decoded
			}
		}
		name := enumConstantName(raw)
		if count := usedNames[name]; count > 0 {
			name = fmt.Sprintf("%s_%d", name, count+1)
		}
		usedNames[name] = usedNames[name] + 1
		values = append(values, enumValueModel{
			Name:      name,
			WireValue: raw,
		})
	}
	if len(values) == 0 {
		return nil, false
	}
	return values, true
}

// collectProperties gathers direct and allOf properties into a unified map.
func collectProperties(schema *base.Schema) map[string]*base.SchemaProxy {
	props := map[string]*base.SchemaProxy{}
	if schema == nil {
		return props
	}
	if schema.Properties != nil {
		for name := range schema.Properties.KeysFromOldest() {
			props[name] = schema.Properties.GetOrZero(name)
		}
	}
	for _, item := range schema.AllOf {
		if item == nil {
			continue
		}
		s := item.Schema()
		if s == nil || s.Properties == nil {
			continue
		}
		for name := range s.Properties.KeysFromOldest() {
			props[name] = s.Properties.GetOrZero(name)
		}
	}
	return props
}

// collectRequired returns a lookup map of required properties, accounting for
// composed schemas.
func collectRequired(schema *base.Schema) map[string]bool {
	required := map[string]bool{}
	if schema == nil {
		return required
	}
	for _, name := range schema.Required {
		required[name] = true
	}
	for _, item := range schema.AllOf {
		if item == nil {
			continue
		}
		s := item.Schema()
		if s == nil {
			continue
		}
		for _, name := range s.Required {
			required[name] = true
		}
	}
	return required
}

// schemaFromProxy safely unwraps a SchemaProxy, handling nils.
func schemaFromProxy(proxy *base.SchemaProxy) *base.Schema {
	if proxy == nil {
		return nil
	}
	return proxy.Schema()
}

// schemaDescription returns the description for a schema proxy.
func schemaDescription(proxy *base.SchemaProxy) string {
	if schema := schemaFromProxy(proxy); schema != nil {
		return schema.Description
	}
	return ""
}

// schemaHasType determines whether a schema declares a specific primitive
// type.
func schemaHasType(schema *base.Schema, want string) bool {
	if schema == nil {
		return false
	}
	for _, t := range schema.Type {
		if t == want {
			return true
		}
	}
	return false
}

// isReadOnlySchema reports whether the schema marks this value as readOnly.
func isReadOnlySchema(proxy *base.SchemaProxy) bool {
	schema := schemaFromProxy(proxy)
	return schema != nil && schema.ReadOnly != nil && *schema.ReadOnly
}

// sortedImports deterministically orders import strings so rendered files do
// not churn between runs.
func sortedImports(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	values := make([]string, 0, len(set))
	for v := range set {
		values = append(values, v)
	}
	sort.Strings(values)
	return values
}

// withAdditionalPropertiesImports ensures generated models that capture extra
// JSON fields include the Jackson and collection types used by the template.
func withAdditionalPropertiesImports(imports []string) []string {
	return uniqueStrings(append(imports,
		"com.fasterxml.jackson.annotation.JsonAnyGetter",
		"com.fasterxml.jackson.annotation.JsonAnySetter",
		"com.fasterxml.jackson.databind.annotation.JsonDeserialize",
		"com.fasterxml.jackson.databind.annotation.JsonPOJOBuilder",
		"java.util.LinkedHashMap",
	))
}
