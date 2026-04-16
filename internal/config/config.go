package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	APIs            APIsConfig        `mapstructure:"apis"`
	Cache           CacheConfig       `mapstructure:"cache"`
	Output          OutputConfig      `mapstructure:"output"`
	SourceOverrides map[string]string `mapstructure:"source_overrides"`
}

type APIsConfig struct {
	CoinGecko APIKeyConfig `mapstructure:"coingecko"`
	Binance   APIKeyConfig `mapstructure:"binance"`
}

type APIKeyConfig struct {
	APIKey string `mapstructure:"api_key"`
}

type CacheConfig struct {
	Enabled bool           `mapstructure:"enabled"`
	Dir     string         `mapstructure:"dir"`
	TTL     map[string]int `mapstructure:"ttl"`
}

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

func Load(path string) (Config, error) {
	cfg := defaults()

	v := viper.New()
	v.SetConfigFile(path)

	// Set defaults
	v.SetDefault("cache.enabled", true)
	v.SetDefault("output.format", "json")
	v.SetDefault("output.pretty", false)

	// Read config file if it exists
	if _, err := os.Stat(path); err == nil {
		// Check permissions
		info, err := os.Stat(path)
		if err != nil {
			return Config{}, fmt.Errorf("stat config: %w", err)
		}
		if info.Mode().Perm()&0077 != 0 {
			return Config{}, fmt.Errorf("config file %s has permissions %04o; must be 0600 or stricter", path, info.Mode().Perm())
		}

		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("reading config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("stat config: %w", err)
	}

	// Bind environment variables
	v.SetEnvPrefix("CRYPTOSPECT")
	v.BindEnv("apis.coingecko.api_key", "CRYPTOSPECT_COINGECKO_KEY")
	v.BindEnv("apis.binance.api_key", "CRYPTOSPECT_BINANCE_KEY")

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

func (c *Config) SourceFor(datapoint, defaultEndpoint string) string {
	if endpoint, ok := c.SourceOverrides[datapoint]; ok {
		return endpoint
	}
	return defaultEndpoint
}

func Write(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists at %s", path)
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
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

# Source mapping overrides
source_overrides:
  # override registry's default datapoint → endpoint mapping
  # example: global_market: "coingecko.custom_endpoint"
`

	return os.WriteFile(path, []byte(template), 0600)
}
