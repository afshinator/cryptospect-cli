package main

import (
	"bytes"
	"encoding/json"
	"testing"

	_ "github.com/afshinator/cryptospect-cli/internal/metrics/liquiditypulse/v1"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

func TestLiquidityPulseCommand(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"liquidity-pulse"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if resp.Status != "ok" && resp.Status != "error" {
		t.Errorf("envelope Status = %v, want ok or error", resp.Status)
	}
	if len(resp.Results) != 1 {
		t.Errorf("Results length = %v, want 1", len(resp.Results))
	}
	if resp.Results[0].Metric != "liquidity-pulse" {
		t.Errorf("Metric = %v, want liquidity-pulse", resp.Results[0].Metric)
	}

	// When live API is unavailable, returns "unavailable" - accept both
	if resp.Results[0].Status != "ok" && resp.Results[0].Status != "degraded" && resp.Results[0].Status != "unavailable" {
		t.Errorf("Result status = %v, want ok/degraded/unavailable", resp.Results[0].Status)
	}

	// Only validate data structure if not unavailable
	if resp.Results[0].Status == "ok" || resp.Results[0].Status == "degraded" {
		var data struct {
			VolumeToMcapRatio float64 `json:"volume_to_mcap_ratio"`
			VolumeUSD         float64 `json:"volume_usd"`
			MarketCapUSD      float64 `json:"market_cap_usd"`
			Classification    struct {
				Label string `json:"label"`
			} `json:"classification"`
			Summary string `json:"summary"`
		}
		if err := json.Unmarshal(resp.Results[0].Data, &data); err != nil {
			t.Fatalf("unmarshal liquidity data: %v", err)
		}
		if data.VolumeToMcapRatio < 0 {
			t.Errorf("VolumeToMcapRatio = %v, want >= 0", data.VolumeToMcapRatio)
		}
		if data.Classification.Label == "" {
			t.Error("classification label is empty")
		}
	}
}

func TestLiquidityPulseAlias(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"lp"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if resp.Results[0].Metric != "liquidity-pulse" {
		t.Errorf("Metric = %v, want liquidity-pulse via alias", resp.Results[0].Metric)
	}
}

func TestLiquidityPulseOutputJSON(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"liquidity-pulse", "--output", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if resp.Status != "ok" && resp.Status != "error" {
		t.Errorf("envelope Status = %v, want ok or error", resp.Status)
	}
}

func TestLiquidityPulseDetailExtended(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"liquidity-pulse", "--detail", "extended"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}

	if len(resp.Results) == 0 {
		t.Fatal("no results in response")
	}
	st := resp.Results[0].Status
	if (st == "ok" || st == "degraded") && resp.Results[0].Meta == nil {
		t.Error("Meta should be present for extended detail when data is available")
	}
}

func TestLiquidityPulseDetailFull(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"liquidity-pulse", "--detail", "full"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}

	if len(resp.Results) == 0 {
		t.Fatal("no results in response")
	}
	st := resp.Results[0].Status
	if (st == "ok" || st == "degraded") && resp.Results[0].Meta == nil {
		t.Error("Meta should be present for full detail when data is available")
	}
}
