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
	flags := flag.NewFlagSet("task-service", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("config", "", "path to a YAML configuration file")
	databasePath := flags.String("database-path", "", "override the SQLite database path")
	mode := flags.String("mode", "", "override server mode (desktop or remote)")
	host := flags.String("host", "", "override server bind IP")
	port := flags.Int("port", 0, "override server port")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "task-service does not accept positional arguments")
		return 2
	}

	set := map[string]bool{}
	flags.Visit(func(value *flag.Flag) { set[value.Name] = true })
	overrides := config.Overrides{}
	if set["database-path"] {
		overrides.DatabasePath = databasePath
	}
	if set["mode"] {
		overrides.Mode = mode
	}
	if set["host"] {
		overrides.Host = host
	}
	if set["port"] {
		overrides.Port = port
	}

	configuration, err := config.Load(*configPath, os.LookupEnv, overrides)
	if err != nil {
		writeStartupError(err)
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx, configuration, os.Stdout, os.Stderr); err != nil {
		writeStartupError(err)
		return 1
	}
	return 0
}

func writeStartupError(err error) {
	_ = json.NewEncoder(os.Stderr).Encode(map[string]string{
		"level":     "error",
		"component": "go-service",
		"event":     "service.failed",
		"message":   err.Error(),
		"exitCode":  strconv.Itoa(1),
	})
}
