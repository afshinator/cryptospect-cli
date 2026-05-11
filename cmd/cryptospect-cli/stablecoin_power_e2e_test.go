package main

import (
	"bytes"
	"encoding/json"
	"testing"

	_ "github.com/afshinator/cryptospect-cli/internal/metrics/stablecoinpower/v1"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

func TestStablecoinPowerCommand(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"stablecoin-power"})
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
		t.Fatalf("Results length = %v, want 1", len(resp.Results))
	}
	if resp.Results[0].Metric != "stablecoin-power" {
		t.Errorf("Metric = %v, want stablecoin-power", resp.Results[0].Metric)
	}

	// Accept any valid status — live API may be unavailable in CI.
	st := resp.Results[0].Status
	if st != "ok" && st != "degraded" && st != "unavailable" {
		t.Errorf("Result status = %v, want ok/degraded/unavailable", st)
	}

	// Only validate data shape when we got real data.
	if st == "ok" || st == "degraded" {
		var data struct {
			StablePowerRatio   float64 `json:"stable_power_ratio"`
			StableMcapUSD      float64 `json:"stable_mcap_usd"`
			VolatileMcapUSD    float64 `json:"volatile_mcap_usd"`
			SupplyTrend7d      string  `json:"supply_trend_7d"`
			StablecoinsCounted int     `json:"stablecoins_counted"`
			Classification     struct {
				Label string `json:"label"`
			} `json:"classification"`
			Summary string `json:"summary"`
		}
		if err := json.Unmarshal(resp.Results[0].Data, &data); err != nil {
			t.Fatalf("unmarshal stablecoin-power data: %v", err)
		}
		if data.StablePowerRatio < 0 {
			t.Errorf("StablePowerRatio = %v, want >= 0", data.StablePowerRatio)
		}
		if data.StablecoinsCounted < 1 {
			t.Errorf("StablecoinsCounted = %d, want >= 1", data.StablecoinsCounted)
		}
		if data.SupplyTrend7d == "" {
			t.Error("SupplyTrend7d must not be empty")
		}
		if data.Classification.Label == "" {
			t.Error("classification label is empty")
		}
		if data.Summary == "" {
			t.Error("summary is empty")
		}
	}
}

func TestStablecoinPowerAlias(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"sp"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if resp.Results[0].Metric != "stablecoin-power" {
		t.Errorf("Metric = %v, want stablecoin-power via alias sp", resp.Results[0].Metric)
	}
}

func TestStablecoinPowerDetailExtended(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"stablecoin-power", "--detail", "extended"})
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

	// Extended must not include full-detail-only fields.
	if resp.Results[0].Meta != nil {
		var meta map[string]interface{}
		if err := json.Unmarshal(resp.Results[0].Meta, &meta); err != nil {
			t.Fatalf("unmarshal meta: %v", err)
		}
		if _, has := meta["thresholds"]; has {
			t.Error("extended detail must not include thresholds")
		}
		if _, has := meta["description"]; has {
			t.Error("extended detail must not include description")
		}
		if _, has := meta["top_n_stablecoins"]; has {
			t.Error("extended detail must not include top_n_stablecoins")
		}
	}
}

func TestStablecoinPowerDetailFull(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"stablecoin-power", "--detail", "full"})
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

	if resp.Results[0].Meta != nil {
		var meta map[string]interface{}
		if err := json.Unmarshal(resp.Results[0].Meta, &meta); err != nil {
			t.Fatalf("unmarshal meta: %v", err)
		}
		if _, has := meta["thresholds"]; !has {
			t.Error("full detail should include thresholds")
		}
		if _, has := meta["description"]; !has {
			t.Error("full detail should include description")
		}
	}
}

func TestStablecoinPowerTopFlag(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"stablecoin-power", "--top", "10"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}

	st := resp.Results[0].Status
	if st == "ok" || st == "degraded" {
		var data struct {
			StablecoinsCounted int `json:"stablecoins_counted"`
		}
		if err := json.Unmarshal(resp.Results[0].Data, &data); err != nil {
			t.Fatalf("unmarshal data: %v", err)
		}
		if data.StablecoinsCounted < 1 {
			t.Errorf("StablecoinsCounted = %d, want >= 1", data.StablecoinsCounted)
		}
	}
}

func TestStablecoinPowerTopFlagClamped(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	// --top 3 is below the minimum of 8, should be clamped.
	cmd.SetArgs([]string{"stablecoin-power", "--detail", "extended", "--top", "3"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}

	st := resp.Results[0].Status
	if (st == "ok" || st == "degraded") && resp.Results[0].Meta != nil {
		var meta map[string]interface{}
		if err := json.Unmarshal(resp.Results[0].Meta, &meta); err != nil {
			t.Fatalf("unmarshal meta: %v", err)
		}
		if clamped, ok := meta["top_clamped"].(bool); ok && !clamped {
			t.Error("top_clamped should be true when --top 3 is passed")
		}
	}
}
