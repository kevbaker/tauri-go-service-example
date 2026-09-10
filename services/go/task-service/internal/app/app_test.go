package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/config"
)

func TestRunEmitsReadinessAndStopsThroughAuthenticatedEndpoint(t *testing.T) {
	configuration := config.Defaults()
	configuration.Database.Path = filepath.Join(t.TempDir(), "tasks.db")
	configuration.Token = "test-token"
	if err := configuration.Validate(); err != nil {
		t.Fatal(err)
	}
	reader, writer := io.Pipe()
	result := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() {
		err := Run(ctx, configuration, writer, io.Discard)
		_ = writer.CloseWithError(err)
		result <- err
	}()

	var ready Readiness
	if err := json.NewDecoder(reader).Decode(&ready); err != nil {
		t.Fatalf("decode readiness: %v", err)
	}

	created := invoke(t, ready.Address, `{"protocolVersion":1,"requestId":"create-1","operation":"tasks.create","payload":{"title":"Persist me","description":"across calls","status":"in_progress"}}`)
	data, ok := created["data"].(map[string]any)
	if !ok || data["title"] != "Persist me" || data["status"] != "in_progress" {
		t.Fatalf("create response = %#v", created)
	}
	id, ok := data["id"].(string)
	if !ok || id == "" {
		t.Fatalf("created id = %#v", data["id"])
	}
	listed := invoke(t, ready.Address, `{"protocolVersion":1,"requestId":"list-1","operation":"tasks.list","payload":{}}`)
	items, ok := listed["data"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("list response = %#v", listed)
	}
	updated := invoke(t, ready.Address, `{"protocolVersion":1,"requestId":"update-1","operation":"tasks.update","payload":{"id":"`+id+`","input":{"description":null,"status":"done"}}}`)
	updatedData, ok := updated["data"].(map[string]any)
	if !ok || updatedData["status"] != "done" || updatedData["description"] != nil {
		t.Fatalf("update response = %#v", updated)
	}
	deleted := invoke(t, ready.Address, `{"protocolVersion":1,"requestId":"delete-1","operation":"tasks.delete","payload":{"id":"`+id+`"}}`)
	if deleted["ok"] != true {
		t.Fatalf("delete response = %#v", deleted)
	}
	missing := invoke(t, ready.Address, `{"protocolVersion":1,"requestId":"get-1","operation":"tasks.get","payload":{"id":"`+id+`"}}`)
	missingError, ok := missing["error"].(map[string]any)
	if !ok || missingError["code"] != "NOT_FOUND" {
		t.Fatalf("missing response = %#v", missing)
	}

	request, err := http.NewRequest(http.MethodPost, "http://"+ready.Address+"/v1/shutdown", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer test-token")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("shutdown request: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("shutdown status = %d", response.StatusCode)
	}
	if err := <-result; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func invoke(t *testing.T, address, body string) map[string]any {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, "http://"+address+"/v1/invoke", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("invoke request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("invoke status = %d", response.StatusCode)
	}
	var decoded map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode invoke response: %v", err)
	}
	return decoded
}
