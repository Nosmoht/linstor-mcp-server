package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ntbc/linstor-mcp-server/internal/app"
	"github.com/ntbc/linstor-mcp-server/internal/config"
)

func main() {
	var flags config.FlagValues
	flag.StringVar(&flags.ConfigPath, "config", "", "Path to config.toml")
	flag.StringVar(&flags.Profile, "profile", "", "Configuration profile name")
	flag.StringVar(&flags.HTTPAddr, "http-addr", "", "Streamable HTTP listen address")
	flag.BoolVar(&flags.EnableHTTPBeta, "enable-http-beta", false, "Enable beta Streamable HTTP transport")
	flag.StringVar(&flags.LogFormat, "log-format", "", "Log format: text or json")
	flag.Parse()

	cfg, err := config.Load(flags)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := config.NewLogger(cfg.LogFormat)
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	server, err := app.New(ctx, cfg)
	if err != nil {
		logger.Error("failed to initialize server", "error", err)
		os.Exit(1)
	}
	defer server.Close()

	if cfg.HTTPAddr != "" {
		if err := server.RunHTTP(ctx); err != nil {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
		return
	}

	if err := server.RunStdio(ctx); err != nil {
		logger.Error("stdio server failed", "error", err)
		os.Exit(1)
	}
}
