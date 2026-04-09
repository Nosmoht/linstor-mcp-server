package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

const (
	DefaultProfile = "homelab"
	DefaultName    = "linstor-mcp-server"
	Version        = "0.1.0"
)

type FlagValues struct {
	ConfigPath     string
	Profile        string
	HTTPAddr       string
	EnableHTTPBeta bool
	LogFormat      string
}

type Config struct {
	ServerName          string `toml:"server_name"`
	ServerVersion       string `toml:"server_version"`
	Profile             string `toml:"profile"`
	KubeContext         string `toml:"kube_context"`
	LinstorCluster      string `toml:"linstor_cluster"`
	ControllerMode      string `toml:"controller_mode"`
	ControllerURL       string `toml:"controller_url"`
	ControllerService   string `toml:"controller_service"`
	ControllerNamespace string `toml:"controller_namespace"`
	TLSCASecret         string `toml:"tls_ca_secret"`
	TLSClientSecret     string `toml:"tls_client_secret"`
	StateDir            string `toml:"state_dir"`
	HTTPAddr            string `toml:"http_addr"`
	EnableHTTPBeta      bool   `toml:"enable_http_beta"`
	LogFormat           string `toml:"log_format"`
}

func Load(flags FlagValues) (Config, error) {
	cfg := defaultConfig()

	path := flags.ConfigPath
	if path == "" {
		path = os.Getenv("LINSTOR_MCP_CONFIG")
	}
	if path != "" {
		if err := mergeConfigFile(path, &cfg); err != nil {
			return Config{}, err
		}
	}

	applyEnv(&cfg)
	applyFlags(&cfg, flags)

	if cfg.Profile == "" {
		cfg.Profile = DefaultProfile
	}
	applyProfileDefaults(&cfg)

	if cfg.StateDir == "" {
		stateHome, err := os.UserHomeDir()
		if err != nil {
			return Config{}, err
		}
		cfg.StateDir = filepath.Join(stateHome, ".local", "state", "linstor-mcp-server")
	}

	if cfg.ServerName == "" {
		cfg.ServerName = DefaultName
	}
	if cfg.ServerVersion == "" {
		cfg.ServerVersion = Version
	}
	if cfg.LogFormat == "" {
		cfg.LogFormat = "text"
	}
	if cfg.HTTPAddr != "" && !cfg.EnableHTTPBeta {
		return Config{}, errors.New("http transport is beta; pass --enable-http-beta or set enable_http_beta=true")
	}
	if cfg.ControllerMode == "" {
		cfg.ControllerMode = "port-forward"
	}
	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		ServerName:          DefaultName,
		ServerVersion:       Version,
		Profile:             DefaultProfile,
		KubeContext:         "admin@homelab",
		LinstorCluster:      "homelab",
		ControllerMode:      "port-forward",
		ControllerService:   "linstor-controller",
		ControllerNamespace: "piraeus-datastore",
		TLSCASecret:         "linstor-api-tls",
		TLSClientSecret:     "linstor-client-tls",
		LogFormat:           "text",
	}
}

func mergeConfigFile(path string, cfg *Config) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %q: %w", path, err)
	}
	if err := toml.Unmarshal(b, cfg); err != nil {
		return fmt.Errorf("parse config file %q: %w", path, err)
	}
	return nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("LINSTOR_MCP_PROFILE"); v != "" {
		cfg.Profile = v
	}
	if v := os.Getenv("LINSTOR_MCP_KUBE_CONTEXT"); v != "" {
		cfg.KubeContext = v
	}
	if v := os.Getenv("LINSTOR_MCP_LINSTOR_CLUSTER"); v != "" {
		cfg.LinstorCluster = v
	}
	if v := os.Getenv("LINSTOR_MCP_CONTROLLER_MODE"); v != "" {
		cfg.ControllerMode = v
	}
	if v := os.Getenv("LINSTOR_MCP_CONTROLLER_URL"); v != "" {
		cfg.ControllerURL = v
	}
	if v := os.Getenv("LINSTOR_MCP_CONTROLLER_SERVICE"); v != "" {
		cfg.ControllerService = v
	}
	if v := os.Getenv("LINSTOR_MCP_CONTROLLER_NAMESPACE"); v != "" {
		cfg.ControllerNamespace = v
	}
	if v := os.Getenv("LINSTOR_MCP_TLS_CA_SECRET"); v != "" {
		cfg.TLSCASecret = v
	}
	if v := os.Getenv("LINSTOR_MCP_TLS_CLIENT_SECRET"); v != "" {
		cfg.TLSClientSecret = v
	}
	if v := os.Getenv("LINSTOR_MCP_STATE_DIR"); v != "" {
		cfg.StateDir = v
	}
	if v := os.Getenv("LINSTOR_MCP_HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if v := os.Getenv("LINSTOR_MCP_ENABLE_HTTP_BETA"); strings.EqualFold(v, "true") || v == "1" {
		cfg.EnableHTTPBeta = true
	}
	if v := os.Getenv("LINSTOR_MCP_LOG_FORMAT"); v != "" {
		cfg.LogFormat = v
	}
}

func applyFlags(cfg *Config, flags FlagValues) {
	if flags.Profile != "" {
		cfg.Profile = flags.Profile
	}
	if flags.HTTPAddr != "" {
		cfg.HTTPAddr = flags.HTTPAddr
	}
	if flags.EnableHTTPBeta {
		cfg.EnableHTTPBeta = true
	}
	if flags.LogFormat != "" {
		cfg.LogFormat = flags.LogFormat
	}
}

func applyProfileDefaults(cfg *Config) {
	switch cfg.Profile {
	case "", DefaultProfile:
		if cfg.KubeContext == "" {
			cfg.KubeContext = "admin@homelab"
		}
		if cfg.LinstorCluster == "" {
			cfg.LinstorCluster = "homelab"
		}
		if cfg.ControllerMode == "" {
			cfg.ControllerMode = "port-forward"
		}
		if cfg.ControllerService == "" {
			cfg.ControllerService = "linstor-controller"
		}
		if cfg.ControllerNamespace == "" {
			cfg.ControllerNamespace = "piraeus-datastore"
		}
		if cfg.TLSCASecret == "" {
			cfg.TLSCASecret = "linstor-api-tls"
		}
		if cfg.TLSClientSecret == "" {
			cfg.TLSClientSecret = "linstor-client-tls"
		}
	}
}

func NewLogger(format string) *slog.Logger {
	var handler slog.Handler
	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	default:
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	return slog.New(handler)
}
