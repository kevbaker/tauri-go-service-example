package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/config"
	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/mcpserver"
	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/sqlite"
	"github.com/kbaker/tauri-go-service-example/services/go/task-service/internal/task"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RunMCP(ctx context.Context, configuration config.MCPConfig, stderr io.Writer) error {
	logger := newLogger(stderr, configuration.App.LogLevel, false)
	logger.Info("MCP server starting", "component", "mcp-server", "event", "mcp.server.starting", "transport", "stdio")
	repository, err := sqlite.Open(ctx, configuration.Database.Path)
	if err != nil {
		return fmt.Errorf("initialize persistence: %w", err)
	}
	server := mcpserver.New(task.NewService(repository), logger)
	runErr := server.Run(ctx, &mcp.StdioTransport{})
	if errors.Is(runErr, context.Canceled) && ctx.Err() != nil {
		runErr = nil
	}
	closeErr := repository.Close()
	logger.Info("MCP server stopped", "component", "mcp-server", "event", "mcp.server.stopped")
	return errors.Join(runErr, closeErr)
}
