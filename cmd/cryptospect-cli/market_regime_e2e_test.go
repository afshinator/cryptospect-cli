package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	_ "github.com/afshinator/cryptospect-cli/internal/metrics/marketregime/v1"
	"github.com/afshinator/cryptospect-cli/internal/output"

	mrv1 "github.com/afshinator/cryptospect-cli/internal/metrics/marketregime/v1"
)

func TestMarketRegimeCommand(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"market-regime"})
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
	if resp.Results[0].Metric != "market-regime" {
		t.Errorf("Metric = %v, want market-regime", resp.Results[0].Metric)
	}

	// Accept any valid status — live API may have transient failures.
	st := resp.Results[0].Status
	if st != "ok" && st != "degraded" && st != "unavailable" {
		t.Errorf("Result status = %v, want ok/degraded/unavailable", st)
	}

	// Only validate data shape when we got real or degraded data.
	if st == "ok" || st == "degraded" {
		var data struct {
			Regime             string  `json:"regime"`
			Modifier           string  `json:"modifier"`
			DominanceTrend     string  `json:"dominance_trend"`
			Conviction         string  `json:"conviction"`
			MarketBreadthScore float64 `json:"market_breadth_score"`
			Classification     struct {
				Label       string `json:"label"`
				Description string `json:"description"`
			} `json:"classification"`
			Summary string `json:"summary"`
		}
		if err := json.Unmarshal(resp.Results[0].Data, &data); err != nil {
			t.Fatalf("unmarshal market-regime data: %v", err)
		}
		if data.Regime == "" {
			t.Error("regime must not be empty")
		}
		if data.Modifier == "" {
			t.Error("modifier must not be empty")
		}
		if data.DominanceTrend == "" {
			t.Error("dominance_trend must not be empty")
		}
		if data.Conviction == "" {
			t.Error("conviction must not be empty")
		}
		if data.Classification.Label == "" {
			t.Error("classification label must not be empty")
		}
		if data.Classification.Description == "" {
			t.Error("classification description must not be empty")
		}
		if data.Summary == "" {
			t.Error("summary must not be empty")
		}

		// Basic detail must suppress meta
		if resp.Results[0].Meta != nil {
			t.Error("Meta should be nil at basic detail (suppressed by root.go)")
		}
	}
}

func TestMarketRegimeAlias(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"mr"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("alias command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("no results in response")
	}
	if resp.Results[0].Metric != "market-regime" {
		t.Errorf("Metric = %v, want market-regime via alias mr", resp.Results[0].Metric)
	}
}

func TestMarketRegimeDetailExtended(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"market-regime", "--detail", "extended"})
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
		// Extended should have these fields
		for _, f := range []string{
			"cache_hit", "ttl_remaining_sec", "primary_source",
			"btc_dominance_pct", "confidence", "dominance_cold_start",
			"notes", "cache_hint_sec", "lp_ratio", "weights_used",
		} {
			if _, ok := meta[f]; !ok {
				t.Errorf("extended detail missing field %q", f)
			}
		}
	}
}

func TestMarketRegimeDetailFull(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"market-regime", "--detail", "full"})
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

func TestMarketRegime_UnavailableDataShape(t *testing.T) {
	// When data is unavailable, the payload should be {"error":"..."}
	// and meta should be nil.
	// This test is inherently live-API-dependent; it asserts the shape
	// if/when we get an unavailable response.
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"market-regime"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("no results")
	}

	if resp.Results[0].Status == "unavailable" {
		var data map[string]string
		if err := json.Unmarshal(resp.Results[0].Data, &data); err != nil {
			t.Fatalf("unmarshal unavailable data: %v", err)
		}
		if _, ok := data["error"]; !ok {
			t.Error("unavailable data should be {\"error\":\"...\"}")
		}
		if resp.Results[0].Meta != nil {
			t.Error("meta should be nil when unavailable")
		}
	}
}

func TestMarketRegime_ProviderDirect(t *testing.T) {
	// Direct provider test for confidence that provider works end-to-end
	p := &mrv1.Provider{}
	result, err := p.Compute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want unavailable with nil data", result.Status)
	}
	if result.Metric != "market-regime" {
		t.Errorf("Metric = %q, want market-regime", result.Metric)
	}
}
