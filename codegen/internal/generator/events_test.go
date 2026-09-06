package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderEvents(t *testing.T) {
	t.Parallel()
	const spec = `{"openapi":"3.1.0","info":{"title":"test","version":"1"},"paths":{},"components":{"schemas":{"Widget":{"type":"object","properties":{"id":{"type":"string"}}}}},"webhooks":{"widgets.updated":{"post":{"operationId":"WidgetUpdatedWebhook","description":"Widget changed.","x-object":{"$ref":"#/components/schemas/Widget"},"responses":{"200":{"description":"ok"}}}}}}`
	path := filepath.Join(t.TempDir(), "openapi.json")
	if err := os.WriteFile(path, []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := loadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	params := Params{OutputDir: t.TempDir(), BasePackage: "com.test.sdk"}
	if err := renderEvents(doc, params); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(params.OutputDir, "com/test/sdk/events")
	for file, expected := range map[string]string{
		"WidgetUpdatedEvent.java": "extends FetchableEvent<Widget>",
		"EventsHandler.java":      "onWidgetUpdated(EventCallback<WidgetUpdatedEvent>",
		"AsyncEventsHandler.java": "onWidgetUpdated(AsyncEventCallback<WidgetUpdatedEvent>",
		"EventNotification.java":  `case "widgets.updated" -> WidgetUpdatedEvent.class`,
	} {
		content, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), expected) {
			t.Errorf("%s missing %q", file, expected)
		}
		if err := renderEvents(doc, params); err != nil {
			t.Fatal(err)
		}
		again, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != string(again) {
			t.Errorf("%s is not deterministic", file)
		}
	}
	runtime := filepath.Join(dir, "EventSignature.java")
	if err := os.WriteFile(runtime, []byte("// Handwritten runtime"), 0o644); err != nil {
		t.Fatal(err)
	}

	doc.Webhooks.GetOrZero("widgets.updated").Post.OperationId = ""
	if err := renderEvents(doc, params); err == nil {
		t.Fatal("expected error for missing operation ID")
	}
	doc.Webhooks = nil
	if err := renderEvents(doc, params); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "WidgetUpdatedEvent.java")); !os.IsNotExist(err) {
		t.Fatalf("obsolete event still exists: %v", err)
	}
	if _, err := os.Stat(runtime); err != nil {
		t.Fatalf("handwritten runtime was removed: %v", err)
	}

}
