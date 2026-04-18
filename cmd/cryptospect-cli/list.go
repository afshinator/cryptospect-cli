package main

import (
	"encoding/json"

	"github.com/afshinator/cryptospect-cli/internal/metrics"
	"github.com/afshinator/cryptospect-cli/internal/output"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-metrics",
		Short: "List all available metrics and their aliases",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := metrics.GlobalRegistry()
			list := reg.List()

			// Convert to JSON data
			data, err := json.Marshal(list)
			if err != nil {
				return err
			}

			// Create a single MetricResult with metric name "list-metrics"
			result := output.MetricResult{
				Metric: "list-metrics",
				Status: "ok",
				Data:   json.RawMessage(data),
			}

			return output.WriteSuccess([]output.MetricResult{result})
		},
	}

	return cmd
}
