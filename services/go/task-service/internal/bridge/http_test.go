package bridge

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/config"
)

func testHTTPHandler() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dispatcher := NewDispatcher(&fakeTaskService{}, config.PublicConfig{}, logger)
	return NewHTTPHandler(dispatcher, "secret", []string{"http://localhost:1420"}, time.Second, nil, logger)
}

func TestHTTPHandlerRequiresAuthentication(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	testHTTPHandler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || strings.Contains(response.Body.String(), "secret") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestHTTPHandlerInvokesBridge(t *testing.T) {
	body := `{"protocolVersion":1,"requestId":"request-1","operation":"system.health","payload":{}}`
	request := httptest.NewRequest(http.MethodPost, "/v1/invoke", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer secret")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testHTTPHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"ok":true`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestHTTPHandlerRejectsOversizedAndMalformedRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"oversized", `{"protocolVersion":1,"requestId":"request-1","operation":"tasks.create","payload":{"title":"` + strings.Repeat("x", int(MaxRequestBytes)) + `"}}`, http.StatusRequestEntityTooLarge},
		{"malformed", `{`, http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/v1/invoke", strings.NewReader(test.body))
			request.Header.Set("Authorization", "Bearer secret")
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			testHTTPHandler().ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestHTTPHandlerEnforcesOriginAllowlist(t *testing.T) {
	for _, test := range []struct {
		origin string
		want   int
	}{
		{"http://localhost:1420", http.StatusNoContent},
		{"http://attacker.invalid", http.StatusForbidden},
	} {
		request := httptest.NewRequest(http.MethodOptions, "/v1/invoke", nil)
		request.Header.Set("Origin", test.origin)
		response := httptest.NewRecorder()
		testHTTPHandler().ServeHTTP(response, request)
		if response.Code != test.want {
			t.Fatalf("origin %s status = %d", test.origin, response.Code)
		}
	}
}
