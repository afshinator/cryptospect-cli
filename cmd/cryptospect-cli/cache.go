package main

import (
	"fmt"

	"github.com/afshinator/cryptospect-cli/internal/cache"
	"github.com/afshinator/cryptospect-cli/internal/output"
	"github.com/spf13/cobra"
)

func newCacheClearCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache-clear",
		Short: "Clear the local API response cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Retrieve config from context (set by root command)
			ctx := cmd.Context()
			cfg, ok := configFromContext(ctx)
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
