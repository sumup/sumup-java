package generator

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	base "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"go.yaml.in/yaml/v4"
)

const sampleCatalogSchemaVersion = 1

// SampleCatalog is the versioned JSON contract consumed by documentation sites.
type SampleCatalog struct {
	SchemaVersion  int       `json:"schemaVersion"`
	Language       string    `json:"language"`
	SDK            SampleSDK `json:"sdk"`
	OpenAPIVersion string    `json:"openAPIVersion"`
	Samples        []Sample  `json:"samples"`
}

// SampleSDK identifies the package and version used by generated samples.
type SampleSDK struct {
	Module  string `json:"module"`
	Version string `json:"version"`
}

// Sample is a complete Java program for one OpenAPI operation example.
type Sample struct {
	ID          string `json:"id"`
	OperationID string `json:"operationId"`
	Example     string `json:"example,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
	HTTPMethod  string `json:"httpMethod"`
	Path        string `json:"path"`
	Source      string `json:"sample"`
}

type requestExample struct {
	name        string
	summary     string
	description string
	value       any
	provided    bool
}

// BuildSamples creates a deterministic catalog from the same model used by SDK generation.
func BuildSamples(params Params, sdkVersion string) (*SampleCatalog, error) {
	if strings.TrimSpace(sdkVersion) == "" {
		return nil, fmt.Errorf("missing SDK version")
	}
	if err := params.normalize(); err != nil {
		return nil, err
	}
	doc, err := loadDocument(params.SpecPath)
	if err != nil {
		return nil, err
	}
	model, err := buildModel(doc, params)
	if err != nil {
		return nil, err
	}

	registry := make(map[string]schemaModel, len(model.Schemas)*2)
	for _, schema := range model.Schemas {
		registry[schema.Package+"."+schema.ClassName] = schema
		registry[schema.ClassName] = schema
	}

	samples := make([]Sample, 0)
	for _, client := range model.Clients {
		for _, operation := range client.Methods {
			for _, example := range operationRequestExamples(operation.Operation) {
				renderer := javaSampleRenderer{registry: registry}
				source, err := renderer.render(client, operation, example)
				if err != nil {
					return nil, fmt.Errorf("render sample %q: %w", operation.OperationID, err)
				}
				id := operation.OperationID
				if example.name != "" {
					id += "." + example.name
				}
				summary := strings.TrimSpace(operation.Operation.Summary)
				if example.summary != "" {
					summary = strings.TrimSpace(example.summary)
				}
				description := strings.TrimSpace(operation.Operation.Description)
				if example.description != "" {
					description = strings.TrimSpace(example.description)
				}
				samples = append(samples, Sample{
					ID:          id,
					OperationID: operation.OperationID,
					Example:     example.name,
					Summary:     summary,
					Description: description,
					HTTPMethod:  operation.HttpMethod,
					Path:        operation.Path,
					Source:      source,
				})
			}
		}
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].ID < samples[j].ID })

	openAPIVersion := ""
	if doc.Info != nil {
		openAPIVersion = strings.TrimSpace(doc.Info.Version)
	}
	return &SampleCatalog{
		SchemaVersion: sampleCatalogSchemaVersion,
		Language:      "java",
		SDK: SampleSDK{
			Module:  "com.sumup:sumup-sdk",
			Version: strings.TrimSpace(sdkVersion),
		},
		OpenAPIVersion: openAPIVersion,
		Samples:        samples,
	}, nil
}

type javaSampleRenderer struct {
	registry map[string]schemaModel
}

func (r javaSampleRenderer) render(client clientModel, operation operationModel, example requestExample) (string, error) {
	args := make([]string, 0)
	for _, parameter := range operation.PathParams {
		value, provided := parameterSample(parameter.Parameter)
		args = append(args, r.value(parameter.Type.Name, parameter.Schema, value, provided, true, 0))
	}
	for _, parameter := range operation.RequiredQueryParams {
		value, provided := parameterSample(parameter.Parameter)
		args = append(args, r.value(parameter.Type.Name, parameter.Schema, value, provided, true, 0))
	}
	if operation.HasRequestBody {
		args = append(args, r.value(operation.RequestBodyType.Name, operation.RequestSchema, example.value, example.provided, true, 0))
	}

	className := pascalCase(operation.OperationID+" "+example.name, "Sample")
	call := "client." + client.AccessorName + "()." + operation.MethodName + "("
	if len(args) > 0 {
		call += "\n"
		for i, arg := range args {
			call += indentJava(arg, 8)
			if i < len(args)-1 {
				call += ","
			}
			call += "\n"
		}
		call += "    "
	}
	call += ")"

	var source strings.Builder
	source.WriteString("import com.sumup.sdk.SumUpClient;\n\n")
	fmt.Fprintf(&source, "public final class %s {\n", className)
	source.WriteString("  public static void main(String[] args) throws Exception {\n")
	source.WriteString("    var client = new SumUpClient();\n\n")
	if operation.ResponseType.IsVoid {
		source.WriteString("    ")
		source.WriteString(strings.ReplaceAll(call, "\n", "\n    "))
		source.WriteString(";\n")
	} else {
		source.WriteString("    var result = ")
		source.WriteString(strings.ReplaceAll(call, "\n", "\n    "))
		source.WriteString(";\n")
		source.WriteString("    System.out.println(result);\n")
	}
	source.WriteString("  }\n}\n")
	return source.String(), nil
}

func (r javaSampleRenderer) value(typeName string, schema *base.SchemaProxy, raw any, provided bool, allowSchemaExamples bool, depth int) string {
	if depth > 8 {
		return "null"
	}
	if !provided && allowSchemaExamples {
		raw, provided = schemaSample(schema)
	}

	if model, ok := r.registry[typeName]; ok {
		if model.IsEnum {
			value := stringValue(raw, "example")
			if !provided && len(model.EnumValues) > 0 {
				value = model.EnumValues[0].WireValue
			}
			return typeName + ".of(" + strconv.Quote(value) + ")"
		}
		values, _ := raw.(map[string]any)
		if model.HasBuilder {
			var expression strings.Builder
			expression.WriteString(typeName + ".builder()")
			for _, field := range model.Fields {
				if field.ReadOnly || field.Name == "additionalProperties" {
					continue
				}
				value, fieldProvided := values[field.WireName]
				if !fieldProvided && field.Required && !provided && allowSchemaExamples {
					value, fieldProvided = schemaSample(field.Schema)
				}
				if !field.Required && !fieldProvided {
					continue
				}
				expression.WriteString("\n    ." + field.Name + "(")
				expression.WriteString(r.value(field.Type, field.Schema, value, fieldProvided, allowSchemaExamples && !provided, depth+1))
				expression.WriteString(")")
			}
			expression.WriteString("\n    .build()")
			return expression.String()
		}
		if len(model.Fields) == 1 {
			field := model.Fields[0]
			value, fieldProvided := values[field.WireName]
			if !fieldProvided && !provided && allowSchemaExamples {
				value, fieldProvided = schemaSample(field.Schema)
			}
			return "new " + typeName + "(" + r.value(field.Type, field.Schema, value, fieldProvided, allowSchemaExamples && !provided, depth+1) + ")"
		}
	}

	if strings.HasPrefix(typeName, "java.util.List<") {
		inner := strings.TrimSuffix(strings.TrimPrefix(typeName, "java.util.List<"), ">")
		items, _ := raw.([]any)
		parts := make([]string, 0, len(items))
		var itemSchema *base.SchemaProxy
		if schema != nil && schema.Schema() != nil && schema.Schema().Items != nil && schema.Schema().Items.IsA() {
			itemSchema = schema.Schema().Items.A
		}
		for _, item := range items {
			parts = append(parts, r.value(inner, itemSchema, item, true, allowSchemaExamples && !provided, depth+1))
		}
		return "java.util.List.of(" + strings.Join(parts, ", ") + ")"
	}
	if strings.HasPrefix(typeName, "java.util.Map<") {
		values, _ := raw.(map[string]any)
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		entries := make([]string, 0, len(keys))
		for _, key := range keys {
			entries = append(entries, "java.util.Map.entry("+strconv.Quote(key)+", "+r.anyValue(values[key], depth+1)+")")
		}
		return "java.util.Map.ofEntries(" + strings.Join(entries, ", ") + ")"
	}

	switch typeName {
	case "String":
		return strconv.Quote(stringValue(raw, fallbackString(schema)))
	case "Boolean":
		if value, ok := raw.(bool); ok {
			return strconv.FormatBool(value)
		}
		return "true"
	case "Integer":
		return integerLiteral(raw, "")
	case "Long":
		return integerLiteral(raw, "L")
	case "Float":
		return numberLiteral(raw, "f")
	case "Double":
		return numberLiteral(raw, "d")
	case "java.time.OffsetDateTime":
		return "java.time.OffsetDateTime.parse(" + strconv.Quote(stringValue(raw, "2025-01-01T12:00:00Z")) + ")"
	case "java.time.LocalDate":
		return "java.time.LocalDate.parse(" + strconv.Quote(stringValue(raw, "2025-01-01")) + ")"
	case "java.util.UUID":
		return "java.util.UUID.fromString(" + strconv.Quote(stringValue(raw, "00000000-0000-0000-0000-000000000000")) + ")"
	case "Object":
		return r.anyValue(raw, depth+1)
	default:
		if strings.HasPrefix(typeName, "com.sumup.sdk.models.") {
			return "null"
		}
		return r.anyValue(raw, depth+1)
	}
}

func (r javaSampleRenderer) anyValue(raw any, depth int) string {
	switch value := raw.(type) {
	case nil:
		return "null"
	case string:
		return strconv.Quote(value)
	case bool:
		return strconv.FormatBool(value)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10) + "L"
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64) + "d"
	case []any:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			parts = append(parts, r.anyValue(item, depth+1))
		}
		return "java.util.List.of(" + strings.Join(parts, ", ") + ")"
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		entries := make([]string, 0, len(keys))
		for _, key := range keys {
			entries = append(entries, "java.util.Map.entry("+strconv.Quote(key)+", "+r.anyValue(value[key], depth+1)+")")
		}
		return "java.util.Map.ofEntries(" + strings.Join(entries, ", ") + ")"
	default:
		return strconv.Quote(fmt.Sprintf("%v", value))
	}
}

func operationRequestExamples(operation *v3.Operation) []requestExample {
	if operation == nil || operation.RequestBody == nil || operation.RequestBody.Content == nil {
		return []requestExample{{}}
	}
	media := operation.RequestBody.Content.GetOrZero("application/json")
	if media == nil {
		return []requestExample{{}}
	}
	if media.Examples != nil && media.Examples.Len() > 0 {
		names := make([]string, 0, media.Examples.Len())
		for name := range media.Examples.KeysFromOldest() {
			names = append(names, name)
		}
		sort.Strings(names)
		examples := make([]requestExample, 0, len(names))
		for _, name := range names {
			example := media.Examples.GetOrZero(name)
			if example == nil {
				continue
			}
			value, provided := decodeNode(example.Value)
			examples = append(examples, requestExample{name: name, summary: example.Summary, description: example.Description, value: value, provided: provided})
		}
		if len(examples) > 0 {
			return examples
		}
	}
	if value, provided := decodeNode(media.Example); provided {
		return []requestExample{{value: value, provided: true}}
	}
	if value, provided := schemaSample(media.Schema); provided {
		return []requestExample{{value: value, provided: true}}
	}
	return []requestExample{{}}
}

func parameterSample(parameter *v3.Parameter) (any, bool) {
	if parameter == nil {
		return nil, false
	}
	if value, ok := decodeNode(parameter.Example); ok {
		return value, true
	}
	if parameter.Examples != nil {
		for _, example := range parameter.Examples.FromOldest() {
			if example != nil {
				if value, ok := decodeNode(example.Value); ok {
					return value, true
				}
			}
		}
	}
	return schemaSample(parameterSchema(parameter))
}

func schemaSample(proxy *base.SchemaProxy) (any, bool) {
	if proxy == nil || proxy.Schema() == nil {
		return nil, false
	}
	schema := proxy.Schema()
	if value, ok := decodeNode(schema.Example); ok {
		return value, true
	}
	for _, example := range schema.Examples {
		if value, ok := decodeNode(example); ok {
			return value, true
		}
	}
	if value, ok := decodeNode(schema.Default); ok {
		return value, true
	}
	if len(schema.Enum) > 0 {
		return decodeNode(schema.Enum[0])
	}
	return nil, false
}

func decodeNode(node *yaml.Node) (any, bool) {
	if node == nil {
		return nil, false
	}
	var value any
	if err := node.Decode(&value); err != nil {
		return nil, false
	}
	return value, true
}

func fallbackString(schema *base.SchemaProxy) string {
	if schema == nil || schema.Schema() == nil {
		return "example"
	}
	switch schema.Schema().Format {
	case "uuid":
		return "00000000-0000-0000-0000-000000000000"
	case "uri", "url":
		return "https://example.com"
	case "email":
		return "user@example.com"
	case "date":
		return "2025-01-01"
	case "date-time":
		return time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	default:
		return "example"
	}
}

func stringValue(value any, fallback string) string {
	if text, ok := value.(string); ok && text != "" {
		return text
	}
	return fallback
}

func integerLiteral(value any, suffix string) string {
	switch number := value.(type) {
	case int:
		return strconv.Itoa(number) + suffix
	case int64:
		return strconv.FormatInt(number, 10) + suffix
	case float64:
		return strconv.FormatInt(int64(number), 10) + suffix
	default:
		return "1" + suffix
	}
}

func numberLiteral(value any, suffix string) string {
	var literal string
	switch number := value.(type) {
	case int:
		literal = strconv.Itoa(number) + ".0"
	case int64:
		literal = strconv.FormatInt(number, 10) + ".0"
	case float64:
		literal = strconv.FormatFloat(number, 'f', -1, 64)
	default:
		literal = "10.1"
	}
	if !strings.Contains(literal, ".") {
		literal += ".0"
	}
	return literal + suffix
}

func indentJava(value string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	return prefix + strings.ReplaceAll(value, "\n", "\n"+prefix)
}
