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
			defs := reg.List()

			data, err := json.Marshal(defs)
			if err != nil {
				return err
			}

			result := output.MetricResult{
				Metric:  "list-metrics",
				Version: "v1.0.0",
				Status:  "ok",
				Data:    json.RawMessage(data),
			}

			return output.WriteSuccess([]output.MetricResult{result})
		},
	}

	return cmd
}
