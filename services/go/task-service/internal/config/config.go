package config

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App      AppConfig      `yaml:"app" json:"app"`
	Database DatabaseConfig `yaml:"database" json:"database"`
	Server   ServerConfig   `yaml:"server" json:"server"`
	UI       UIConfig       `yaml:"ui" json:"ui"`
	Token    string         `yaml:"-" json:"-"`
}

type AppConfig struct {
	Environment string `yaml:"environment" json:"environment"`
	LogLevel    string `yaml:"logLevel" json:"logLevel"`
}

type DatabaseConfig struct {
	Path string `yaml:"path" json:"path"`
}

type ServerConfig struct {
	Mode             string        `yaml:"mode" json:"mode"`
	Host             string        `yaml:"host" json:"host"`
	Port             int           `yaml:"port" json:"port"`
	AllowedOrigins   []string      `yaml:"allowedOrigins" json:"allowedOrigins"`
	RequestTimeout   time.Duration `yaml:"-" json:"-"`
	ShutdownTimeout  time.Duration `yaml:"-" json:"-"`
	RequestTimeoutS  string        `yaml:"requestTimeout" json:"requestTimeout"`
	ShutdownTimeoutS string        `yaml:"shutdownTimeout" json:"shutdownTimeout"`
}

type UIConfig struct {
	AppName  string `yaml:"appName" json:"appName"`
	PageSize int    `yaml:"pageSize" json:"pageSize"`
}

type PublicConfig struct {
	Environment string `json:"environment"`
	UI          struct {
		AppName  string `json:"appName"`
		PageSize int    `json:"pageSize"`
	} `json:"ui"`
}

type Overrides struct {
	DatabasePath *string
	Mode         *string
	Host         *string
	Port         *int
	Token        *string
}

func Defaults() Config {
	return Config{
		App:      AppConfig{Environment: "production", LogLevel: "info"},
		Database: DatabaseConfig{Path: "./data/tasks.db"},
		Server: ServerConfig{
			Mode:             "desktop",
			Host:             "127.0.0.1",
			Port:             0,
			AllowedOrigins:   []string{},
			RequestTimeoutS:  "10s",
			ShutdownTimeoutS: "5s",
		},
		UI: UIConfig{AppName: "Tauri Go Tasks", PageSize: 25},
	}
}

func Load(path string, lookupEnv func(string) (string, bool), overrides Overrides) (Config, error) {
	configuration := Defaults()
	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return Config{}, fmt.Errorf("open config file: %w", err)
		}
		defer file.Close()
		decoder := yaml.NewDecoder(io.LimitReader(file, 1<<20))
		decoder.KnownFields(true)
		if err := decoder.Decode(&configuration); err != nil {
			return Config{}, fmt.Errorf("decode config file: %w", err)
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			return Config{}, errors.New("decode config file: expected one YAML document")
		}
	}
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	if err := applyEnvironment(&configuration, lookupEnv); err != nil {
		return Config{}, err
	}
	applyOverrides(&configuration, overrides)
	if overrides.Token == nil {
		if configuration.Server.Mode == "remote" {
			configuration.Token, _ = lookupEnv("TGS_REMOTE_ACCESS_TOKEN")
		} else {
			configuration.Token, _ = lookupEnv("TGS_SERVICE_TOKEN")
		}
	}
	if err := configuration.Validate(); err != nil {
		return Config{}, err
	}
	return configuration, nil
}

func applyEnvironment(configuration *Config, lookup func(string) (string, bool)) error {
	stringValues := []struct {
		key    string
		target *string
	}{
		{"TGS_APP_ENVIRONMENT", &configuration.App.Environment},
		{"TGS_APP_LOG_LEVEL", &configuration.App.LogLevel},
		{"TGS_DATABASE_PATH", &configuration.Database.Path},
		{"TGS_SERVER_MODE", &configuration.Server.Mode},
		{"TGS_SERVER_HOST", &configuration.Server.Host},
		{"TGS_SERVER_REQUEST_TIMEOUT", &configuration.Server.RequestTimeoutS},
		{"TGS_SERVER_SHUTDOWN_TIMEOUT", &configuration.Server.ShutdownTimeoutS},
		{"TGS_UI_APP_NAME", &configuration.UI.AppName},
	}
	for _, value := range stringValues {
		if raw, ok := lookup(value.key); ok {
			*value.target = raw
		}
	}
	if raw, ok := lookup("TGS_SERVER_PORT"); ok {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("TGS_SERVER_PORT must be an integer: %w", err)
		}
		configuration.Server.Port = value
	}
	if raw, ok := lookup("TGS_UI_PAGE_SIZE"); ok {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("TGS_UI_PAGE_SIZE must be an integer: %w", err)
		}
		configuration.UI.PageSize = value
	}
	if raw, ok := lookup("TGS_SERVER_ALLOWED_ORIGINS"); ok {
		configuration.Server.AllowedOrigins = splitNonEmpty(raw)
	}
	return nil
}

func applyOverrides(configuration *Config, overrides Overrides) {
	if overrides.DatabasePath != nil {
		configuration.Database.Path = *overrides.DatabasePath
	}
	if overrides.Mode != nil {
		configuration.Server.Mode = *overrides.Mode
	}
	if overrides.Host != nil {
		configuration.Server.Host = *overrides.Host
	}
	if overrides.Port != nil {
		configuration.Server.Port = *overrides.Port
	}
	if overrides.Token != nil {
		configuration.Token = *overrides.Token
	}
}

func (c *Config) Validate() error {
	fields := make([]string, 0)
	if strings.TrimSpace(c.App.Environment) == "" {
		fields = append(fields, "app.environment is required")
	}
	switch c.App.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		fields = append(fields, "app.logLevel must be debug, info, warn, or error")
	}
	if strings.TrimSpace(c.Database.Path) == "" {
		fields = append(fields, "database.path is required")
	}
	if c.Server.Mode != "desktop" && c.Server.Mode != "remote" {
		fields = append(fields, "server.mode must be desktop or remote")
	}
	ip := net.ParseIP(c.Server.Host)
	if ip == nil {
		fields = append(fields, "server.host must be an IP address")
	} else if c.Server.Mode == "desktop" && !ip.IsLoopback() {
		fields = append(fields, "desktop mode must bind to loopback")
	}
	if c.Server.Port < 0 || c.Server.Port > 65535 {
		fields = append(fields, "server.port must be between 0 and 65535")
	}
	if c.Server.Mode == "remote" && ip != nil && !ip.IsLoopback() && len(c.Server.AllowedOrigins) == 0 {
		fields = append(fields, "remote non-loopback mode requires allowed origins")
	}
	for _, origin := range c.Server.AllowedOrigins {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Path != "" {
			fields = append(fields, "server.allowedOrigins entries must be origins")
			break
		}
	}
	var err error
	c.Server.RequestTimeout, err = time.ParseDuration(c.Server.RequestTimeoutS)
	if err != nil || c.Server.RequestTimeout <= 0 {
		fields = append(fields, "server.requestTimeout must be a positive duration")
	}
	c.Server.ShutdownTimeout, err = time.ParseDuration(c.Server.ShutdownTimeoutS)
	if err != nil || c.Server.ShutdownTimeout <= 0 {
		fields = append(fields, "server.shutdownTimeout must be a positive duration")
	}
	if c.Server.Mode == "remote" && c.Server.Port == 0 {
		fields = append(fields, "remote mode requires a fixed non-zero port")
	}
	if strings.TrimSpace(c.Token) == "" {
		fields = append(fields, "the mode-specific access token is required")
	}
	if strings.TrimSpace(c.UI.AppName) == "" {
		fields = append(fields, "ui.appName is required")
	}
	if c.UI.PageSize < 1 || c.UI.PageSize > 100 {
		fields = append(fields, "ui.pageSize must be between 1 and 100")
	}
	if len(fields) > 0 {
		return fmt.Errorf("invalid configuration: %s", strings.Join(fields, "; "))
	}
	return nil
}

func (c Config) Public() PublicConfig {
	public := PublicConfig{Environment: c.App.Environment}
	public.UI.AppName = c.UI.AppName
	public.UI.PageSize = c.UI.PageSize
	return public
}

func splitNonEmpty(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
