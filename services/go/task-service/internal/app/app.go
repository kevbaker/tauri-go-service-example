package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/bridge"
	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/config"
	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/sqlite"
	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/task"
)

type Readiness struct {
	Address         string `json:"address"`
	ProtocolVersion int    `json:"protocolVersion"`
}

func Run(ctx context.Context, configuration config.Config, stdout, stderr io.Writer) error {
	if err := configuration.Validate(); err != nil {
		return err
	}
	logger := newLogger(stderr, configuration.App.LogLevel, configuration.App.Environment == "development" && configuration.Server.Mode == "remote")
	logger.Info("service starting", "component", "go-service", "event", "service.starting", "mode", configuration.Server.Mode)

	repository, err := sqlite.Open(ctx, configuration.Database.Path)
	if err != nil {
		return fmt.Errorf("initialize persistence: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = repository.Close()
		}
	}()

	listener, err := net.Listen("tcp", net.JoinHostPort(configuration.Server.Host, fmt.Sprintf("%d", configuration.Server.Port)))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	stop := make(chan struct{})
	var stopOnce sync.Once
	requestStop := func() { stopOnce.Do(func() { close(stop) }) }
	dispatcher := bridge.NewDispatcher(task.NewService(repository), configuration.Public(), logger)
	handler := bridge.NewHTTPHandler(dispatcher, configuration.Token, configuration.Server.AllowedOrigins, configuration.Server.RequestTimeout, requestStop, logger)
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       configuration.Server.RequestTimeout + time.Second,
		WriteTimeout:      configuration.Server.RequestTimeout + time.Second,
		IdleTimeout:       30 * time.Second,
	}

	if err := json.NewEncoder(stdout).Encode(Readiness{Address: listener.Addr().String(), ProtocolVersion: bridge.ProtocolVersion}); err != nil {
		return fmt.Errorf("write readiness: %w", err)
	}
	logger.Info("service ready", "component", "go-service", "event", "service.ready", "address", listener.Addr().String())

	serveErrors := make(chan error, 1)
	go func() { serveErrors <- server.Serve(listener) }()
	select {
	case <-ctx.Done():
		logger.Info("service stopping", "component", "go-service", "event", "service.stopping", "reason", "context")
	case <-stop:
		logger.Info("service stopping", "component", "go-service", "event", "service.stopping", "reason", "request")
	case err := <-serveErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), configuration.Server.ShutdownTimeout)
	defer cancel()
	shutdownErr := server.Shutdown(shutdownContext)
	if shutdownErr != nil {
		_ = server.Close()
	}
	closeErr := repository.Close()
	closed = true
	logger.Info("service stopped", "component", "go-service", "event", "service.stopped")
	return errors.Join(shutdownErr, closeErr)
}

func newLogger(writer io.Writer, levelName string, readable bool) *slog.Logger {
	level := new(slog.LevelVar)
	switch levelName {
	case "debug":
		level.Set(slog.LevelDebug)
	case "warn":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}
	options := &slog.HandlerOptions{Level: level}
	if readable {
		return slog.New(slog.NewTextHandler(writer, options))
	}
	return slog.New(slog.NewJSONHandler(writer, options))
}
