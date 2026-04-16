package config

import (
	"os"
	"path/filepath"
	"testing"
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
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
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
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
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
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
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
	if err := os.WriteFile(path, []byte(content), 0644); err != nil { // world-readable
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("Load with world-readable config should fail")
	}
}
