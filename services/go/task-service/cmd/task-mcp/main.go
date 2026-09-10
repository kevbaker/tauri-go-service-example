package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/app"
	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/config"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(arguments []string) int {
	flags := flag.NewFlagSet("task-mcp", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("config", "", "path to the shared YAML configuration file")
	databasePath := flags.String("database-path", "", "override the SQLite database path")
	logLevel := flags.String("log-level", "", "override the log level (debug, info, warn, or error)")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "task-mcp does not accept positional arguments")
		return 2
	}

	set := map[string]bool{}
	flags.Visit(func(value *flag.Flag) { set[value.Name] = true })
	overrides := config.MCPOverrides{}
	if set["database-path"] {
		overrides.DatabasePath = databasePath
	}
	if set["log-level"] {
		overrides.LogLevel = logLevel
	}
	configuration, err := config.LoadMCP(*configPath, os.LookupEnv, overrides)
	if err != nil {
		writeStartupError(err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.RunMCP(ctx, configuration, os.Stderr); err != nil {
		writeStartupError(err)
		return 1
	}
	return 0
}

func writeStartupError(err error) {
	_ = json.NewEncoder(os.Stderr).Encode(map[string]string{
		"level":     "error",
		"component": "mcp-server",
		"event":     "mcp.server.failed",
		"message":   err.Error(),
		"exitCode":  strconv.Itoa(1),
	})
}
