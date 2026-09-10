package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPrecedenceAndPublicAllowlist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	contents := []byte(`
app:
  environment: development
  logLevel: info
database:
  path: from-file.db
server:
  mode: desktop
  host: 127.0.0.1
  port: 0
  allowedOrigins: []
  requestTimeout: 2s
  shutdownTimeout: 3s
ui:
  appName: File Name
  pageSize: 10
`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	environment := map[string]string{
		"TGS_SERVICE_TOKEN":      "secret",
		"TGS_UI_PAGE_SIZE":       "20",
		"TGS_UI_REFRESH_INTERVAL": "45s",
	}
	overridePath := "override.db"
	configuration, err := Load(path, func(key string) (string, bool) { value, ok := environment[key]; return value, ok }, Overrides{DatabasePath: &overridePath})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if configuration.Database.Path != "override.db" || configuration.UI.PageSize != 20 {
		t.Fatalf("Load() = %#v", configuration)
	}
	public := configuration.Public()
	if public.UI.AppName != "File Name" || public.UI.RefreshIntervalMS != 45_000 || strings.Contains(strings.TrimSpace(public.Environment), "secret") {
		t.Fatalf("Public() = %#v", public)
	}
	encoded, err := json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	for _, privateValue := range []string{"secret", "override.db", "127.0.0.1"} {
		if strings.Contains(string(encoded), privateValue) {
			t.Fatalf("Public() exposed private value %q: %s", privateValue, encoded)
		}
	}
}

func TestValidateRemoteBindingSafety(t *testing.T) {
	tests := []struct {
		name          string
		host          string
		port          int
		origins       []string
		expectedError string
	}{
		{name: "fixed port", host: "127.0.0.1", expectedError: "fixed non-zero port"},
		{name: "origins", host: "0.0.0.0", port: 8787, expectedError: "requires allowed origins"},
		{name: "valid external", host: "0.0.0.0", port: 8787, origins: []string{"https://example.test"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration := Defaults()
			configuration.Server.Mode = "remote"
			configuration.Server.Host = test.host
			configuration.Server.Port = test.port
			configuration.Server.AllowedOrigins = test.origins
			configuration.Token = "secret"
			err := configuration.Validate()
			if test.expectedError == "" && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if test.expectedError != "" && (err == nil || !strings.Contains(err.Error(), test.expectedError)) {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestLoadRejectsInvalidEnvironmentNumbers(t *testing.T) {
	_, err := Load("", func(key string) (string, bool) {
		values := map[string]string{"TGS_SERVER_PORT": "not-a-number", "TGS_SERVICE_TOKEN": "secret"}
		value, ok := values[key]
		return value, ok
	}, Overrides{})
	if err == nil || !strings.Contains(err.Error(), "TGS_SERVER_PORT") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestValidateRejectsTooFrequentUIRefresh(t *testing.T) {
	configuration := Defaults()
	configuration.Token = "secret"
	configuration.UI.RefreshIntervalS = "500ms"
	if err := configuration.Validate(); err == nil || !strings.Contains(err.Error(), "ui.refreshInterval") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("unknown: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path, func(string) (string, bool) { return "", false }, Overrides{})
	if err == nil || !strings.Contains(err.Error(), "field unknown") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestValidateRefusesExternalDesktopBind(t *testing.T) {
	configuration := Defaults()
	configuration.Token = "secret"
	configuration.Server.Host = "0.0.0.0"
	if err := configuration.Validate(); err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestModeOverrideSelectsRemoteToken(t *testing.T) {
	remote := "remote"
	environment := map[string]string{
		"TGS_SERVICE_TOKEN":       "desktop-secret",
		"TGS_REMOTE_ACCESS_TOKEN": "remote-secret",
		"TGS_SERVER_PORT":         "8787",
	}
	configuration, err := Load("", func(key string) (string, bool) { value, ok := environment[key]; return value, ok }, Overrides{Mode: &remote})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if configuration.Token != "remote-secret" {
		t.Fatalf("Token = %q", configuration.Token)
	}
}

func TestCheckedInConfigurationsLoad(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..", "config")
	environment := map[string]string{
		"TGS_SERVICE_TOKEN":       "desktop-secret",
		"TGS_REMOTE_ACCESS_TOKEN": "remote-secret",
	}
	lookup := func(key string) (string, bool) { value, ok := environment[key]; return value, ok }
	for _, name := range []string{"default.yaml", "development.yaml"} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(filepath.Join(root, name), lookup, Overrides{}); err != nil {
				t.Fatalf("Load(%s) error = %v", name, err)
			}
		})
	}
}

func TestLoadMCPUsesSharedSourcesWithoutRequiringNetworkConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	contents := []byte(`
app:
  environment: development
  logLevel: info
database:
  path: from-file.db
server:
  mode: remote
  host: invalid-for-http
  port: 0
  allowedOrigins: []
  requestTimeout: invalid
  shutdownTimeout: invalid
ui:
  appName: Tasks
  pageSize: 25
`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	environment := map[string]string{"TGS_DATABASE_PATH": "from-environment.db"}
	overrideLevel := "debug"
	configuration, err := LoadMCP(path, func(key string) (string, bool) {
		value, ok := environment[key]
		return value, ok
	}, MCPOverrides{LogLevel: &overrideLevel})
	if err != nil {
		t.Fatalf("LoadMCP() error = %v", err)
	}
	if configuration.Database.Path != "from-environment.db" || configuration.App.LogLevel != "debug" {
		t.Fatalf("LoadMCP() = %#v", configuration)
	}
}
