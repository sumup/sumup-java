package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

// renderClients writes one Java class per OpenAPI tag, effectively generating
// the API-specific client classes.
func renderClients(model sdkModel, params Params) error {
	dir := filepath.Join(params.OutputDir, params.clientPackagePath())
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove clients directory: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create clients directory: %w", err)
	}

	syncTmpl, err := loadTemplate("client.tmpl")
	if err != nil {
		return err
	}
	asyncTmpl, err := loadTemplate("client_async.tmpl")
	if err != nil {
		return err
	}

	for _, client := range model.Clients {
		data := prepareClientTemplateData(client, client.ClassName, false)
		var buf bytes.Buffer
		if err := syncTmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("render client %s: %w", client.ClassName, err)
		}
		target := filepath.Join(dir, client.ClassName+".java")
		if err := os.WriteFile(target, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write client %s: %w", client.ClassName, err)
		}

		asyncData := prepareClientTemplateData(client, asyncClientClassName(client.ClassName), true)
		buf.Reset()
		if err := asyncTmpl.Execute(&buf, asyncData); err != nil {
			return fmt.Errorf("render async client %s: %w", asyncData.ClassName, err)
		}
		asyncTarget := filepath.Join(dir, asyncData.ClassName+".java")
		if err := os.WriteFile(asyncTarget, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write async client %s: %w", asyncData.ClassName, err)
		}
	}
	return nil
}

// renderSumUpClient wires individual clients into a single entry-point usable
// by SDK consumers.
func renderSumUpClient(model sdkModel, params Params) error {
	tmpl, err := loadTemplate("sumup_client.tmpl")
	if err != nil {
		return err
	}

	variants := []struct {
		ClassName string
		Async     bool
	}{
		{ClassName: "SumUpClient", Async: false},
		{ClassName: "SumUpAsyncClient", Async: true},
	}
	for _, variant := range variants {
		data := prepareSumUpClientData(model, variant.ClassName, variant.Async)

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("render %s: %w", variant.ClassName, err)
		}
		target := filepath.Join(params.OutputDir, params.basePackagePath(), variant.ClassName+".java")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create sumup client directory: %w", err)
		}
		if err := os.WriteFile(target, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", variant.ClassName, err)
		}
	}
	return nil
}

// renderModels generates POJO classes that mirror OpenAPI schemas.
func renderModels(model sdkModel, params Params) error {
	if len(model.Schemas) == 0 {
		return nil
	}
	dir := filepath.Join(params.OutputDir, params.modelPackagePath())
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove models directory: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create models directory: %w", err)
	}

	tmpl, err := loadTemplate("model.tmpl")
	if err != nil {
		return err
	}

	for _, schema := range model.Schemas {
		data := prepareModelTemplateData(schema)
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("render model %s: %w", schema.ClassName, err)
		}
		target := filepath.Join(dir, schema.ClassName+".java")
		if err := os.WriteFile(target, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write model %s: %w", schema.ClassName, err)
		}
	}

	return nil
}

// methodParam describes a parameter for a generated Java method.
type methodParam struct {
	Type        string
	Name        string
	Required    bool
	Description string
}

// operationTemplateData captures every bit of context templates need to render
// client methods.
type operationTemplateData struct {
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
	HasRequestBody       bool
	ResponseType         javaType
	ReturnsVoid          bool
	HasOptionalArgs      bool
	RequiredParams       []methodParam
	OptionalParams       []methodParam
	RequiredParamDecls   []string
	OptionalParamDecls   []string
	RequiredArgNames     []string
	OptionalNullArgs     []string
	OptionalArgNames     []string
	CallArgsWithNulls    string
	CallArgsWithOptions  string
}

// clientTemplateData contains the data required to render a single client
// class.
type clientTemplateData struct {
	Package       string
	ClassName     string
	TagName       string
	Imports       []string
	Operations    []operationTemplateData
	QueryStructs  []*parameterGroupModel
	HeaderStructs []*parameterGroupModel
}

// prepareClientTemplateData converts an internal client model into a richer
// structure that is easier for templates to consume.
func prepareClientTemplateData(client clientModel, className string, async bool) clientTemplateData {
	importSet := map[string]struct{}{
		"com.sumup.sdk.core.ApiClient":      {},
		"com.sumup.sdk.core.ApiException":   {},
		"com.sumup.sdk.core.HttpMethod":     {},
		"com.sumup.sdk.core.RequestOptions": {},
		"java.util.Objects":                 {},
	}
	importSet["java.util.Map"] = struct{}{}
	if async {
		importSet["java.util.concurrent.CompletableFuture"] = struct{}{}
	}

	operations := make([]operationTemplateData, 0, len(client.Methods))
	var queryStructs []*parameterGroupModel
	var headerStructs []*parameterGroupModel
	needsLinkedMap := false
	needsTypeReference := false

	for _, op := range client.Methods {
		opData := operationTemplateData{
			OperationID:          op.OperationID,
			MethodName:           op.MethodName,
			SummaryLines:         op.SummaryLines,
			DescriptionLines:     op.DescriptionLines,
			HttpMethod:           op.HttpMethod,
			Path:                 op.Path,
			PathParams:           op.PathParams,
			RequiredQueryParams:  op.RequiredQueryParams,
			OptionalQueryParams:  op.OptionalQueryParams,
			RequiredHeaderParams: op.RequiredHeaderParams,
			OptionalHeaderParams: op.OptionalHeaderParams,
			QueryStruct:          op.QueryStruct,
			HeaderStruct:         op.HeaderStruct,
			RequestBodyType:      op.RequestBodyType,
			RequestRequired:      op.RequestRequired,
			HasRequestBody:       op.HasRequestBody,
			ResponseType:         op.ResponseType,
			ReturnsVoid:          op.ResponseType.IsVoid,
			HasOptionalArgs:      op.HasOptionalArgs,
		}

		if op.ResponseType.TypeRefExpr != "" && !op.ResponseType.IsVoid {
			needsTypeReference = true
		}

		requiredParams := make([]methodParam, 0)
		for _, param := range op.PathParams {
			requiredParams = append(requiredParams, methodParam{
				Type:        param.Type.Name,
				Name:        param.FieldName,
				Required:    true,
				Description: describeParameter(param),
			})
		}
		for _, param := range op.RequiredQueryParams {
			requiredParams = append(requiredParams, methodParam{
				Type:        param.Type.Name,
				Name:        param.FieldName,
				Required:    true,
				Description: describeParameter(param),
			})
		}
		if op.HasRequestBody {
			requiredParams = append(requiredParams, methodParam{
				Type:        op.RequestBodyType.Name,
				Name:        "request",
				Required:    op.RequestRequired,
				Description: requestParamDescription(op),
			})
		}
		requiredDecls := make([]string, len(requiredParams))
		requiredNames := make([]string, len(requiredParams))
		for i, param := range requiredParams {
			requiredDecls[i] = fmt.Sprintf("%s %s", param.Type, param.Name)
			requiredNames[i] = param.Name
		}
		opData.RequiredParams = requiredParams
		opData.RequiredParamDecls = requiredDecls
		opData.RequiredArgNames = requiredNames

		var optionalParams []methodParam
		if op.QueryStruct != nil {
			optionalParams = append(optionalParams, methodParam{
				Type:        op.QueryStruct.ClassName,
				Name:        op.QueryStruct.VarName,
				Description: optionalGroupParamDescription(op.QueryStruct),
			})
			queryStructs = append(queryStructs, op.QueryStruct)
			needsLinkedMap = true
		}
		if op.HeaderStruct != nil {
			optionalParams = append(optionalParams, methodParam{
				Type:        op.HeaderStruct.ClassName,
				Name:        op.HeaderStruct.VarName,
				Description: optionalGroupParamDescription(op.HeaderStruct),
			})
			headerStructs = append(headerStructs, op.HeaderStruct)
			needsLinkedMap = true
		}
		optionalDecls := make([]string, len(optionalParams))
		optionalNulls := make([]string, len(optionalParams))
		for i, param := range optionalParams {
			optionalDecls[i] = fmt.Sprintf("%s %s", param.Type, param.Name)
			optionalNulls[i] = "null"
		}
		opData.OptionalParams = optionalParams
		opData.OptionalParamDecls = optionalDecls
		opData.OptionalNullArgs = optionalNulls
		optionalNames := make([]string, len(optionalParams))
		for i, param := range optionalParams {
			optionalNames[i] = param.Name
		}
		opData.OptionalArgNames = optionalNames

		callWithNulls := append([]string{}, requiredNames...)
		callWithNulls = append(callWithNulls, optionalNulls...)
		opData.CallArgsWithNulls = strings.Join(callWithNulls, ", ")

		callWithOptions := append([]string{}, requiredNames...)
		callWithOptions = append(callWithOptions, optionalNames...)
		opData.CallArgsWithOptions = strings.Join(callWithOptions, ", ")

		if op.HasQueryParams || op.HasHeaderParams {
			needsLinkedMap = true
		}

		operations = append(operations, opData)
	}

	if needsTypeReference {
		importSet["com.fasterxml.jackson.core.type.TypeReference"] = struct{}{}
	}
	if needsLinkedMap {
		importSet["java.util.LinkedHashMap"] = struct{}{}
	}

	imports := make([]string, 0, len(importSet))
	for imp := range importSet {
		imports = append(imports, imp)
	}
	sort.Strings(imports)

	return clientTemplateData{
		Package:       client.Package,
		ClassName:     className,
		TagName:       client.TagName,
		Imports:       imports,
		Operations:    operations,
		QueryStructs:  queryStructs,
		HeaderStructs: headerStructs,
	}
}

// prepareSumUpClientData emits the data the SumUpClient template expects.
func prepareSumUpClientData(model sdkModel, className string, async bool) map[string]any {
	imports := []string{
		"com.sumup.sdk.core.ApiClient",
		"java.net.http.HttpClient",
		"java.time.Duration",
		"java.util.Objects",
	}
	clientInfos := make([]map[string]string, 0, len(model.Clients))
	for _, client := range model.Clients {
		clientClass := client.ClassName
		if async {
			clientClass = asyncClientClassName(clientClass)
		}
		imports = append(imports, fmt.Sprintf("%s.%s", model.ClientPackage, clientClass))
		clientInfos = append(clientInfos, map[string]string{
			"ClassName":    clientClass,
			"FieldName":    client.FieldName,
			"AccessorName": client.AccessorName,
			"TagName":      client.TagName,
		})
	}
	sort.Strings(imports)
	return map[string]any{
		"Package":   model.BasePackage,
		"Imports":   unique(imports),
		"Clients":   clientInfos,
		"ClassName": className,
	}
}

// prepareModelTemplateData translates schemaModel into a template-friendly map.
func prepareModelTemplateData(schema schemaModel) map[string]any {
	return map[string]any{
		"Package":          schema.Package,
		"ClassName":        schema.ClassName,
		"Imports":          schema.Imports,
		"DescriptionLines": schema.DescriptionLines,
		"Fields":           schema.Fields,
		"HasRequired":      schema.HasRequired,
	}
}

// unique collapses consecutive duplicates in a sorted slice.
func unique(values []string) []string {
	if len(values) == 0 {
		return values
	}
	out := values[:1]
	for _, v := range values[1:] {
		if v != out[len(out)-1] {
			out = append(out, v)
		}
	}
	return out
}

// loadTemplate parses the named template from the embedded filesystem.
func loadTemplate(name string) (*template.Template, error) {
	content, err := templatesFS.ReadFile("templates/" + name)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", name, err)
	}
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"quote": func(s string) string { return strconvQuote(s) },
		"join":  func(values []string, sep string) string { return strings.Join(values, sep) },
	}).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}
	return tmpl, nil
}

// strconvQuote exposes strconv.Quote to templates without importing strconv in
// each template file.
func strconvQuote(value string) string {
	return strconv.Quote(value)
}

func describeParameter(param parameterModel) string {
	if param.Description != "" {
		return param.Description
	}
	switch param.Location {
	case parameterInPath:
		return "Path parameter."
	case parameterInQuery:
		return "Query parameter."
	case parameterInHeader:
		return "Header parameter."
	default:
		return "Request parameter."
	}
}

func requestParamDescription(op operationModel) string {
	if op.RequestDescription != "" {
		return op.RequestDescription
	}
	return "Request body payload."
}

func optionalGroupParamDescription(group *parameterGroupModel) string {
	if group == nil {
		return ""
	}
	if group.Description != "" {
		return group.Description
	}
	switch group.Kind {
	case parameterInQuery:
		return "Optional query parameters."
	case parameterInHeader:
		return "Optional header overrides."
	default:
		return "Optional parameters."
	}
}
