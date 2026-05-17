package main

import (
	"fmt"

	"github.com/afshinator/cryptospect-cli/internal/cache"
	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/output"
	"github.com/spf13/cobra"
)

func newCacheClearCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache-clear",
		Short: "Clear the local API response cache",
		Long: `Clear all cached API responses stored on disk.

Cache location: ~/.cryptospect-cli/cache/ (or the directory set by your config file).

Each entry is a raw API response keyed by endpoint. Responses are reused until their
TTL expires — default TTLs range from 60 s (Binance klines) to 300 s (CoinGecko global).
The cache is shared across all metrics; clearing it affects every subsequent call.

When to use:
  • Force fresh data after a market event without waiting for TTL expiry
  • Troubleshoot stale or unexpected metric output
  • Reset state after changing API keys or endpoints in your config

The cache is repopulated automatically on the next metric call.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Retrieve config from context (set by root command)
			ctx := cmd.Context()
			cfg, ok := config.FromContext(ctx)
			if !ok {
				return fmt.Errorf("config not found in context")
			}

			cacheDir := cfg.CacheDir()
			if cacheDir == "" {
				return fmt.Errorf("unable to determine cache directory")
			}

			// Check if cache directory exists
			exists, err := cache.Exists(cacheDir)
			if err != nil {
				return fmt.Errorf("checking cache directory: %w", err)
			}
			if !exists {
				// Directory doesn't exist, nothing to clear
				result := output.MetricResult{
					Metric: "cache-clear",
					Status: "ok",
					Data:   nil,
				}
				return output.WriteSuccess([]output.MetricResult{result})
			}

			c, err := cache.Open(cacheDir)
			if err != nil {
				return fmt.Errorf("opening cache: %w", err)
			}
			defer func() { _ = c.Close() }()

			if err := c.Clear(); err != nil {
				return fmt.Errorf("clearing cache: %w", err)
			}

			// Return success envelope
			result := output.MetricResult{
				Metric: "cache-clear",
				Status: "ok",
				Data:   nil, // no data needed
			}
			return output.WriteSuccess([]output.MetricResult{result})
		},
	}

	return cmd
}
