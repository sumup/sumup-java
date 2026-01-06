package generator

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Run executes the custom generator using the provided parameters.
func Run(ctx context.Context, params Params) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := params.normalize(); err != nil {
		return err
	}
	if err := params.validate(); err != nil {
		return err
	}

	doc, err := loadDocument(params.SpecPath)
	if err != nil {
		return err
	}

	model, err := buildModel(doc, params)
	if err != nil {
		return err
	}

	slog.Info("Generating SDK", "spec", params.SpecPath)
	if err := renderClients(model, params); err != nil {
		return err
	}
	if err := renderModels(model, params); err != nil {
		return err
	}
	if err := renderSumUpClient(model, params); err != nil {
		return err
	}

	return nil
}

// loadDocument reads and parses the OpenAPI specification into the pbo33f
// high-level model.
func loadDocument(specPath string) (*v3.Document, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}

	doc, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}

	model, err := doc.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("build spec model: %w", err)
	}

	return &model.Model, nil
}
