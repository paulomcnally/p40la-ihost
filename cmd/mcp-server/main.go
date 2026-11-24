// Comando p40la-mcp: servidor MCP de p40la-ihost. Por defecto usa stdio;
// con P40LA_MCP_TRANSPORT=http expone Streamable HTTP con OAuth (SPEC-147).
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/crypto/bcrypt"

	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/registry"
	mcpserver "github.com/paulomcnally/p40la-ihost/internal/mcp/server"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/all"
)

// version se sobreescribe con -ldflags="-X main.version=..." en el build.
var version = "dev"

func main() {
	hashPassword := flag.Bool("hash-password", false, "lee una contraseña por stdin y escribe su hash bcrypt")
	showVersion := flag.Bool("version", false, "imprime la versión y sale")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}
	if *hashPassword {
		if err := hashPasswordCLI(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := run(); err != nil {
		slog.Error("p40la-mcp", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// stdout queda reservado para el protocolo MCP en modo stdio: todo log va a stderr.
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

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:        "p40la-mcp",
		Title:       "p40la-ihost MCP",
		Description: "Tools para operar facturas, autos, mantenimientos, inspecciones y seguros de p40la-ihost",
		Version:     version,
	}, nil)

	reg := &registry.Registry{}
	all.Register(reg, client, cfg)
	reg.Mount(mcpServer)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.IsHTTP() {
		httpServer, err := mcpserver.New(cfg, mcpServer, logger, version)
		if err != nil {
			return err
		}
		return httpServer.Run(ctx)
	}

	logger.Info("p40la-mcp iniciado",
		"version", version,
		"transport", cfg.Transport,
		"base_url", cfg.BaseURL,
		"writes", cfg.AllowWrites,
		"timezone", cfg.Timezone,
	)
	if err := mcpServer.Run(ctx, &mcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func hashPasswordCLI() error {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("leer contraseña: %w", err)
	}
	password := strings.TrimRight(line, "\r\n")
	if password == "" {
		return fmt.Errorf("la contraseña no puede estar vacía")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return fmt.Errorf("generar hash: %w", err)
	}
	fmt.Println(string(hash))
	return nil
}
