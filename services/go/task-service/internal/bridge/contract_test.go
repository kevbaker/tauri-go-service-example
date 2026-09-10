package bridge

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/config"
	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/task"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func loadTaskContract(t *testing.T) *jsonschema.Schema {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate contract test source")
	}
	schemaPath := filepath.Join(filepath.Dir(sourceFile), "..", "..", "..", "..", "..", "contracts", "task-api.schema.json")
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile task contract: %v", err)
	}
	return schema
}

func validateContractValue(t *testing.T, schema *jsonschema.Schema, value any) {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal Go wire value: %v", err)
	}
	var document any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("decode Go wire value as JSON: %v", err)
	}
	if err := schema.Validate(document); err != nil {
		t.Fatalf("Go wire value does not satisfy task contract:\n%s\n%v", encoded, err)
	}
}

func TestGoWireValuesSatisfyTaskContract(t *testing.T) {
	schema := loadTaskContract(t)
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	description := "Canonical task contract"
	value := task.Task{
		ID:          "task-1",
		Title:       "Share task types",
		Description: &description,
		Status:      task.StatusInProgress,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	configuration := config.Defaults()
	configuration.Token = "contract-test"
	if err := configuration.Validate(); err != nil {
		t.Fatalf("validate default configuration: %v", err)
	}

	values := []any{
		Request{ProtocolVersion: 1, RequestID: "request-config", Operation: OperationConfigGetPublic, Payload: json.RawMessage(`{}`)},
		Request{ProtocolVersion: 1, RequestID: "request-list", Operation: OperationTasksList, Payload: json.RawMessage(`{"status":"in_progress","limit":25,"offset":0}`)},
		Request{ProtocolVersion: 1, RequestID: "request-get", Operation: OperationTasksGet, Payload: json.RawMessage(`{"id":"task-1"}`)},
		Request{ProtocolVersion: 1, RequestID: "request-create", Operation: OperationTasksCreate, Payload: json.RawMessage(`{"title":"Share task types","description":null}`)},
		Request{ProtocolVersion: 1, RequestID: "request-update", Operation: OperationTasksUpdate, Payload: json.RawMessage(`{"id":"task-1","input":{"description":null}}`)},
		Request{ProtocolVersion: 1, RequestID: "request-delete", Operation: OperationTasksDelete, Payload: json.RawMessage(`{"id":"task-1"}`)},
		Response{ProtocolVersion: 1, RequestID: "request-list", OK: true, Data: []task.Task{value}},
		Response{ProtocolVersion: 1, RequestID: "request-config", OK: true, Data: configuration.Public()},
		Response{ProtocolVersion: 1, RequestID: "request-get", OK: true, Data: value},
		Response{ProtocolVersion: 1, RequestID: "request-delete", OK: true, Data: struct{}{}},
		Response{ProtocolVersion: 1, RequestID: "request-error", Error: &ApplicationError{Code: "VALIDATION", Message: "The request is invalid", FieldErrors: map[string]string{"title": "Title is required"}}},
	}

	for _, candidate := range values {
		validateContractValue(t, schema, candidate)
	}
}

func TestTaskContractRejectsAnUnsupportedGoRequest(t *testing.T) {
	schema := loadTaskContract(t)
	request := Request{ProtocolVersion: 1, RequestID: "request-1", Operation: "tasks.publish", Payload: json.RawMessage(`{}`)}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	var document any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if err := schema.Validate(document); err == nil {
		t.Fatal("unsupported task operation satisfied the canonical contract")
	}
}
