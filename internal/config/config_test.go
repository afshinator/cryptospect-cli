package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func TestDefaults(t *testing.T) {
	cfg, err := Load("/nonexistent")
	if err != nil {
		t.Fatalf("Load with nonexistent file: %v", err)
	}
	if !cfg.Cache.Enabled {
		t.Error("Cache.Enabled default should be true")
	}
	if cfg.Cache.Dir != "" {
		t.Errorf("Cache.Dir default should be empty, got %q", cfg.Cache.Dir)
	}
	if cfg.Output.Pretty {
		t.Error("Output.Pretty default should be false")
	}
}

func TestLoadValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
apis:
  coingecko:
    api_key: "test-key-123"
cache:
  enabled: false
  dir: "/custom/cache"
  ttl:
    coingecko_global_market: 600
output:
  pretty: true
source_overrides:
  global_market: "coingecko.custom_endpoint"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.APIs.CoinGecko.APIKey != "test-key-123" {
		t.Errorf("CoinGecko API key = %q, want test-key-123", cfg.APIs.CoinGecko.APIKey)
	}
	if cfg.Cache.Enabled {
		t.Error("Cache.Enabled should be false")
	}
	if cfg.Cache.Dir != "/custom/cache" {
		t.Errorf("Cache.Dir = %q, want /custom/cache", cfg.Cache.Dir)
	}
	if cfg.Cache.TTL["coingecko_global_market"] != 600 {
		t.Errorf("TTL for coingecko_global_market = %v, want 600", cfg.Cache.TTL["coingecko_global_market"])
	}
	if !cfg.Output.Pretty {
		t.Error("Output.Pretty should be true")
	}
	if cfg.SourceOverrides["global_market"] != "coingecko.custom_endpoint" {
		t.Errorf("SourceOverrides[global_market] = %q, want coingecko.custom_endpoint", cfg.SourceOverrides["global_market"])
	}
}

func TestLoadWithYmlExtension(t *testing.T) {
	dir := t.TempDir()
	// Create .yml file (no .yaml file)
	ymlPath := filepath.Join(dir, "config.yml")
	content := `
apis:
  coingecko:
    api_key: "yml-key-123"
cache:
  enabled: false
  dir: "/yml/cache"
`
	if err := os.WriteFile(ymlPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	// Load with .yaml path (should resolve to .yml)
	yamlPath := filepath.Join(dir, "config.yaml")
	cfg, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("Load with .yaml path (but .yml exists) failed: %v", err)
	}
	if cfg.APIs.CoinGecko.APIKey != "yml-key-123" {
		t.Errorf("CoinGecko API key = %q, want yml-key-123", cfg.APIs.CoinGecko.APIKey)
	}
	if cfg.Cache.Dir != "/yml/cache" {
		t.Errorf("Cache.Dir = %q, want /yml/cache", cfg.Cache.Dir)
	}
	if cfg.Cache.Enabled {
		t.Error("Cache.Enabled should be false")
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	t.Setenv("CRYPTOSPECT_COINGECKO_KEY", "env-key-456")
	t.Setenv("CRYPTOSPECT_BINANCE_KEY", "binance-env-key")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
apis:
  coingecko:
    api_key: "file-key-123"
  binance:
    api_key: "file-binance-key"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Environment should override file
	if cfg.APIs.CoinGecko.APIKey != "env-key-456" {
		t.Errorf("CoinGecko API key = %q, want env-key-456", cfg.APIs.CoinGecko.APIKey)
	}
	if cfg.APIs.Binance.APIKey != "binance-env-key" {
		t.Errorf("Binance API key = %q, want binance-env-key", cfg.APIs.Binance.APIKey)
	}
}

func TestSourceFor(t *testing.T) {
	cfg := Config{
		SourceOverrides: map[string]string{
			"global_market": "coingecko.custom",
		},
	}

	// Override present
	if got := cfg.SourceFor("global_market", "coingecko.default"); got != "coingecko.custom" {
		t.Errorf("SourceFor with override = %q, want coingecko.custom", got)
	}

	// No override, use default
	if got := cfg.SourceFor("other", "coingecko.other"); got != "coingecko.other" {
		t.Errorf("SourceFor without override = %q, want coingecko.other", got)
	}
}

func TestInvalidOutputFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `output: {format: "xml"}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("Load with invalid output format should fail")
	}
}

func TestFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `output: {format: "json"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { // world-readable
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("Load with world-readable config should fail")
	}
}

func TestResolveConfigPath(t *testing.T) {
	tests := []struct {
		name  string
		setup func(dir string) (path string, want string)
	}{
		{
			name: "given path exists (yaml)",
			setup: func(dir string) (string, string) {
				path := filepath.Join(dir, "config.yaml")
				if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
					t.Fatal(err)
				}
				return path, path
			},
		},
		{
			name: "given path does not exist, alternative extension exists (yml)",
			setup: func(dir string) (string, string) {
				ymlPath := filepath.Join(dir, "config.yml")
				if err := os.WriteFile(ymlPath, []byte(""), 0o600); err != nil {
					t.Fatal(err)
				}
				// Request .yaml, but only .yml exists
				requestPath := filepath.Join(dir, "config.yaml")
				return requestPath, ymlPath
			},
		},
		{
			name: "both exist, prefer given extension (yaml)",
			setup: func(dir string) (string, string) {
				yamlPath := filepath.Join(dir, "config.yaml")
				if err := os.WriteFile(yamlPath, []byte(""), 0o600); err != nil {
					t.Fatal(err)
				}
				ymlPath := filepath.Join(dir, "config.yml")
				if err := os.WriteFile(ymlPath, []byte(""), 0o600); err != nil {
					t.Fatal(err)
				}
				return yamlPath, yamlPath
			},
		},
		{
			name: "both exist, prefer given extension (yml)",
			setup: func(dir string) (string, string) {
				yamlPath := filepath.Join(dir, "config.yaml")
				if err := os.WriteFile(yamlPath, []byte(""), 0o600); err != nil {
					t.Fatal(err)
				}
				ymlPath := filepath.Join(dir, "config.yml")
				if err := os.WriteFile(ymlPath, []byte(""), 0o600); err != nil {
					t.Fatal(err)
				}
				return ymlPath, ymlPath
			},
		},
		{
			name: "neither exists, return original path",
			setup: func(dir string) (string, string) {
				path := filepath.Join(dir, "nonexistent.cfg")
				return path, path
			},
		},
		{
			name: "no extension, no alternative",
			setup: func(dir string) (string, string) {
				path := filepath.Join(dir, "config")
				return path, path
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path, want := tt.setup(dir)
			got := resolveConfigPath(path)
			if got != want {
				t.Errorf("resolveConfigPath(%q) = %q, want %q", path, got, want)
			}
		})
	}
}

func TestLoadWithViperFlagPrecedence(t *testing.T) {
	// Create a viper instance
	v := viper.New()
	// Create a flagset and bind to viper
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("api-key", "", "API key")
	if err := v.BindPFlag("apis.coingecko.api_key", fs.Lookup("api-key")); err != nil {
		t.Fatalf("binding flag: %v", err)
	}
	// Set the flag value (simulating CLI flag)
	if err := fs.Set("api-key", "flag-value-789"); err != nil {
		t.Fatalf("setting flag: %v", err)
	}

	// Call LoadWithViper with a non-existent config file (no env vars)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg, err := LoadWithViper(v, path)
	if err != nil {
		t.Fatalf("LoadWithViper failed: %v", err)
	}
	// Flag value should be present in config
	if cfg.APIs.CoinGecko.APIKey != "flag-value-789" {
		t.Errorf("CoinGecko API key = %q, want flag-value-789", cfg.APIs.CoinGecko.APIKey)
	}
}
