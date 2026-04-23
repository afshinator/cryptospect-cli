package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/afshinator/cryptospect-cli/internal/config"
	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cryptospect.yaml")
}

func NewRootCommand() *cobra.Command {
	var verbose bool
	var detail string
	var outputFmt string
	var apiKey string
	var configFile string

	cmd := &cobra.Command{
		Use:     "cryptospect-cli",
		Version: version,
		Short:   "Compute crypto market regime metrics for agentic consumption",
		Long: `A portable, zero-dependency CLI tool that fetches live and historical cryptocurrency data,
computes high-signal market regime metrics, and outputs them in a format optimized for AI agents and LLM tool-calling.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Validate detail flag
			allowedDetails := []string{"basic", "extended", "full"}
			valid := false
			for _, d := range allowedDetails {
				if detail == d {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid detail value %q, must be one of: basic, extended, full", detail)
			}

			// Configure logging
			level := slog.LevelWarn
			if verbose {
				level = slog.LevelDebug
			}
			handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
			slog.SetDefault(slog.New(handler))

			// Determine config file path
			path := configFile
			if path == "" {
				path = defaultConfigPath()
			}

			// Create viper instance and bind CLI flags
			v := viper.New()
			if err := v.BindPFlag("apis.coingecko.api_key", cmd.Flags().Lookup("api-key")); err != nil {
				return fmt.Errorf("binding api-key flag: %w", err)
			}
			if err := v.BindPFlag("output.format", cmd.Flags().Lookup("output")); err != nil {
				return fmt.Errorf("binding output flag: %w", err)
			}

			// Load configuration
			cfg, err := config.LoadWithViper(v, path)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Store config and detail in context for subcommands and providers
			ctx := config.StoreInContext(cmd.Context(), cfg)
			ctx = config.StoreDetailInContext(ctx, detail)
			cmd.SetContext(ctx)

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file (default $HOME/.cryptospect.yaml)")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging on stderr")
	cmd.PersistentFlags().StringVar(&detail, "detail", "basic", "Detail level: basic, extended, full")
	cmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "json", "Output format (only json supported)")
	cmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for authenticated endpoints (maps to CoinGecko)")

	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newCacheClearCommand())

	reg := metrics.GlobalRegistry()
	for _, p := range reg.BestProviders() {
		p := p
		def := p.Def()
		metricCmd := &cobra.Command{
			Use:     def.Name,
			Aliases: def.Aliases,
			Short:   def.Description,
			Long:    fmt.Sprintf("%s\n\nVersion: %s | Namespace: %s", def.Description, def.Version, def.Namespace),
			RunE:    buildMetricRunE(p),
		}
		cmd.AddCommand(metricCmd)
	}

	return cmd
}

func buildMetricRunE(p metrics.MetricProvider) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		result, err := p.Compute(cmd.Context(), nil)
		if err != nil {
			return err
		}
		return output.WriteSuccess([]output.MetricResult{result})
	}
}
