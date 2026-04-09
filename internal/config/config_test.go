package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(FlagValues{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Profile != "homelab" {
		t.Fatalf("Profile = %q, want homelab", cfg.Profile)
	}
	if cfg.ControllerMode != "port-forward" {
		t.Fatalf("ControllerMode = %q, want port-forward", cfg.ControllerMode)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	t.Setenv("LINSTOR_MCP_PROFILE", "custom")
	t.Setenv("LINSTOR_MCP_KUBE_CONTEXT", "other")

	cfg, err := Load(FlagValues{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Profile != "custom" {
		t.Fatalf("Profile = %q, want custom", cfg.Profile)
	}
	if cfg.KubeContext != "other" {
		t.Fatalf("KubeContext = %q, want other", cfg.KubeContext)
	}
}

func TestLoadHTTPRequiresBeta(t *testing.T) {
	cfgPath := t.TempDir() + "/config.toml"
	if err := os.WriteFile(cfgPath, []byte("http_addr = \":8080\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := Load(FlagValues{ConfigPath: cfgPath}); err == nil {
		t.Fatal("Load() error = nil, want beta transport error")
	}
}
