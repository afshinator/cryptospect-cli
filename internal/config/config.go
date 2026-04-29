package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type contextKey int

const (
	configContextKey contextKey = iota
	detailContextKey
	topNContextKey
)

// StoreInContext returns a new context carrying cfg.
func StoreInContext(ctx context.Context, cfg Config) context.Context { //nolint:gocritic // value copy is intentional; Config is an immutable snapshot
	return context.WithValue(ctx, configContextKey, cfg)
}

// FromContext retrieves the Config stored by StoreInContext.
func FromContext(ctx context.Context) (Config, bool) {
	cfg, ok := ctx.Value(configContextKey).(Config)
	return cfg, ok
}

// StoreDetailInContext returns a new context carrying the --detail flag value.
func StoreDetailInContext(ctx context.Context, detail string) context.Context {
	return context.WithValue(ctx, detailContextKey, detail)
}

// DetailFromContext retrieves the detail level stored by StoreDetailInContext.
func DetailFromContext(ctx context.Context) (string, bool) {
	detail, ok := ctx.Value(detailContextKey).(string)
	return detail, ok
}

// StoreTopNInContext returns a new context carrying the --top flag value for
// metrics that accept a per-invocation count (e.g. stablecoin-power --top N).
func StoreTopNInContext(ctx context.Context, n int) context.Context {
	return context.WithValue(ctx, topNContextKey, n)
}

// TopNFromContext retrieves the top-N value stored by StoreTopNInContext.
// Returns (0, false) when no value has been stored.
func TopNFromContext(ctx context.Context) (int, bool) {
	n, ok := ctx.Value(topNContextKey).(int)
	return n, ok
}

// resolveConfigPath returns the first existing file with .yaml or .yml extension,
// preferring the given path if it exists. If neither exists, returns the given path.
func resolveConfigPath(path string) string {
	if _, err := os.Stat(path); err == nil {
		return path
	}
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml":
		alt := strings.TrimSuffix(path, ".yaml") + ".yml"
		if _, err := os.Stat(alt); err == nil {
			return alt
		}
	case ".yml":
		alt := strings.TrimSuffix(path, ".yml") + ".yaml"
		if _, err := os.Stat(alt); err == nil {
			return alt
		}
	}
	return path
}

// Config holds the application configuration.
type Config struct {
	APIs            APIsConfig        `mapstructure:"apis"`
	Cache           CacheConfig       `mapstructure:"cache"`
	Output          OutputConfig      `mapstructure:"output"`
	Metrics         MetricsConfig     `mapstructure:"metrics"`
	SourceOverrides map[string]string `mapstructure:"source_overrides"`
}

// MetricsConfig holds per-metric configuration overrides.
type MetricsConfig struct {
	StablecoinPower StablecoinPowerConfig `mapstructure:"stablecoin_power"`
}

// StablecoinPowerConfig holds configuration specific to the stablecoin-power metric.
type StablecoinPowerConfig struct {
	// StablecoinIDs is the ordered list of CoinGecko coin IDs included in the
	// stablecoin supply numerator. When empty, the provider falls back to the
	// built-in SPPStableIDs list in the coingecko package.
	StablecoinIDs []string `mapstructure:"stablecoin_ids"`
}

// APIsConfig holds API key configuration for each provider.
type APIsConfig struct {
	CoinGecko APIKeyConfig `mapstructure:"coingecko"`
	Binance   APIKeyConfig `mapstructure:"binance"`
}

// APIKeyConfig holds a single API key.
type APIKeyConfig struct {
	APIKey string `mapstructure:"api_key"`
}

// CacheConfig holds cache configuration.
type CacheConfig struct {
	Enabled bool           `mapstructure:"enabled"`
	Dir     string         `mapstructure:"dir"`
	TTL     map[string]int `mapstructure:"ttl"`
}

// OutputConfig holds output formatting configuration.
type OutputConfig struct {
	Format string `mapstructure:"format"`
	Pretty bool   `mapstructure:"pretty"`
}

func defaults() Config {
	return Config{
		Cache: CacheConfig{
			Enabled: true,
		},
		Output: OutputConfig{
			Format: "json",
			Pretty: false,
		},
		SourceOverrides: make(map[string]string),
	}
}

// Load loads configuration from the given path, using a new viper instance.
// It respects the standard precedence: config file → environment variables → defaults.
func Load(path string) (Config, error) {
	return LoadWithViper(nil, path)
}

// LoadWithViper loads configuration using the provided viper instance (or creates one if nil).
// The viper instance should already have any CLI flags bound via viper.BindPFlag.
// Environment variables are bound with prefix CRYPTOSPECT.
// Returns the parsed configuration or an error.
func LoadWithViper(v *viper.Viper, path string) (Config, error) {
	cfg := defaults()

	if v == nil {
		v = viper.New()
	}
	resolvedPath := resolveConfigPath(path)
	v.SetConfigFile(resolvedPath)

	// Set defaults
	v.SetDefault("cache.enabled", true)
	v.SetDefault("output.format", "json")
	v.SetDefault("output.pretty", false)

	// Read config file if it exists
	if info, err := os.Stat(resolvedPath); err == nil {
		if info.Mode().Perm()&0o077 != 0 {
			return Config{}, fmt.Errorf("config file %s has permissions %04o; must be 0600 or stricter", resolvedPath, info.Mode().Perm())
		}
		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("reading config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("stat config: %w", err)
	}

	// Bind environment variables (if not already bound)
	v.SetEnvPrefix("CRYPTOSPECT")
	if err := v.BindEnv("apis.coingecko.api_key", "CRYPTOSPECT_COINGECKO_KEY"); err != nil {
		return Config{}, fmt.Errorf("binding coingecko env: %w", err)
	}
	if err := v.BindEnv("apis.binance.api_key", "CRYPTOSPECT_BINANCE_KEY"); err != nil {
		return Config{}, fmt.Errorf("binding binance env: %w", err)
	}

	// Unmarshal into struct
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}

	// Validate output format
	if cfg.Output.Format != "" && cfg.Output.Format != "json" {
		return Config{}, fmt.Errorf("invalid output format %q; only 'json' is supported", cfg.Output.Format)
	}

	return cfg, nil
}

// SourceFor returns the endpoint key to use for a given datapoint.
func (c *Config) SourceFor(datapoint, defaultEndpoint string) string {
	if endpoint, ok := c.SourceOverrides[datapoint]; ok {
		return endpoint
	}
	return defaultEndpoint
}

// CacheDir returns the cache directory path, defaulting to ~/.cryptospect-cli/cache if empty.
func (c *Config) CacheDir() string {
	if c.Cache.Dir != "" {
		return c.Cache.Dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cryptospect-cli", "cache")
}

// Write writes a default configuration template to the given path.
func Write(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists at %s", path)
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating config directory: %w", err)
		}
	}

	template := `# cryptospect-cli configuration

# API keys
apis:
  coingecko:
    api_key: "" # override with CRYPTOSPECT_COINGECKO_KEY
  binance:
    api_key: "" # override with CRYPTOSPECT_BINANCE_KEY

# Cache settings
cache:
  enabled: true
  dir: "" # default: ~/.cryptospect-cli/cache
  ttl:
    # per-endpoint TTL overrides in seconds
    # use underscores instead of dots in keys (coingecko_global_market)
    # coingecko_global_market: 300     # default: 5 min
    # coingecko_coin_markets: 300      # default: 5 min
    # coingecko_derivatives: 300       # default: 5 min
    # binance_spot_cvd: 300            # default: 5 min

# Output settings
output:
  format: "json" # only "json" supported
  pretty: false  # pretty-print JSON

# Per-metric configuration
metrics:
  stablecoin_power:
    # stablecoin_ids: list of CoinGecko coin IDs included in the stable supply numerator.
    # Default (when omitted): built-in list of top-18 stablecoins by market cap.
    # Override to track a custom set or add newly launched stablecoins.
    # stablecoin_ids:
    #   - tether
    #   - usd-coin
    #   - usds
    #   - ethena-usde
    #   - dai
    #   - paypal-usd
    #   - usd1-wlfi
    #   - falcon-finance

# Source mapping overrides
source_overrides:
  # override registry's default datapoint → endpoint mapping
  # example: global_market: "coingecko.custom_endpoint"
`

	return os.WriteFile(path, []byte(template), 0o600)
}
