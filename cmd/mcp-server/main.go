// Comando p40la-mcp: servidor MCP por stdio para operar p40la-ihost.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/registry"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/all"
)

// version se sobreescribe con -ldflags="-X main.version=..." en el build.
var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("p40la-mcp", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// stdout queda reservado para el protocolo MCP: todo log va a stderr.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	client, err := api.New(cfg)
	if err != nil {
		return err
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:        "p40la-mcp",
		Title:       "p40la-ihost MCP",
		Description: "Tools para operar facturas de p40la-ihost",
		Version:     version,
	}, nil)

	reg := &registry.Registry{}
	all.Register(reg, client, cfg)
	reg.Mount(server)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("p40la-mcp iniciado", "version", version, "base_url", cfg.BaseURL, "writes", cfg.AllowWrites, "timezone", cfg.Timezone)
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
