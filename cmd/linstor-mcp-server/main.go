package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ntbc/linstor-mcp-server/internal/app"
	"github.com/ntbc/linstor-mcp-server/internal/config"
)

var (
	version = config.Version
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	var flags config.FlagValues
	var showVersion bool
	flag.StringVar(&flags.ConfigPath, "config", "", "Path to config.toml")
	flag.StringVar(&flags.Profile, "profile", "", "Configuration profile name")
	flag.StringVar(&flags.HTTPAddr, "http-addr", "", "Streamable HTTP listen address")
	flag.BoolVar(&flags.EnableHTTPBeta, "enable-http-beta", false, "Enable beta Streamable HTTP transport")
	flag.StringVar(&flags.LogFormat, "log-format", "", "Log format: text or json")
	flag.BoolVar(&showVersion, "version", false, "Print version information and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("linstor-mcp-server version=%s commit=%s date=%s\n", version, commit, date)
		return 0
	}

	cfg, err := config.Load(flags)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		return 1
	}
	cfg.ServerVersion = version

	logger := config.NewLogger(cfg.LogFormat)
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	server, err := app.New(ctx, cfg)
	if err != nil {
		logger.Error("failed to initialize server", "error", err)
		return 1
	}
	defer server.Close()

	if cfg.HTTPAddr != "" {
		if err := server.RunHTTP(ctx); err != nil {
			logger.Error("http server failed", "error", err)
			return 1
		}
		return 0
	}

	if err := server.RunStdio(ctx); err != nil {
		logger.Error("stdio server failed", "error", err)
		return 1
	}
	return 0
}
