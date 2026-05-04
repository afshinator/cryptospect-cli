package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	_ "github.com/afshinator/cryptospect-cli/internal/metrics/marketbreadth/v1"
	"github.com/afshinator/cryptospect-cli/internal/output"

	"github.com/afshinator/cryptospect-cli/internal/api"
	mbv1 "github.com/afshinator/cryptospect-cli/internal/metrics/marketbreadth/v1"
)

func TestMarketBreadthCommand(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"market-breadth"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("envelope Status = %v, want ok", resp.Status)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("Results length = %v, want 1", len(resp.Results))
	}
	if resp.Results[0].Metric != "market-breadth" {
		t.Errorf("Metric = %v, want market-breadth", resp.Results[0].Metric)
	}

	// Accept any valid status — live API may have transient failures.
	st := resp.Results[0].Status
	if st != "ok" && st != "degraded" && st != "unavailable" {
		t.Errorf("Result status = %v, want ok/degraded/unavailable", st)
	}

	// Only validate data shape when we got real or degraded data.
	if st == "ok" || st == "degraded" {
		var data struct {
			MarketBreadthScore float64 `json:"market_breadth_score"`
			CoinsCounted       int     `json:"coins_counted"`
			DivergenceDetected bool    `json:"divergence_detected"`
			Classification     struct {
				Label string `json:"label"`
			} `json:"classification"`
			Summary string `json:"summary"`
		}
		if err := json.Unmarshal(resp.Results[0].Data, &data); err != nil {
			t.Fatalf("unmarshal market-breadth data: %v", err)
		}
		if data.CoinsCounted < 1 {
			t.Errorf("CoinsCounted = %d, want >= 1", data.CoinsCounted)
		}
		if data.Classification.Label == "" {
			t.Error("classification label must not be empty")
		}
		if data.Summary == "" {
			t.Error("summary must not be empty")
		}
	}
}

func TestMarketBreadthAlias(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"mb"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("alias command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if resp.Results[0].Metric != "market-breadth" {
		t.Errorf("Metric = %v, want market-breadth via alias mb", resp.Results[0].Metric)
	}
}

func TestMarketBreadthDetailExtended(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"market-breadth", "--detail", "extended"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
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
	}
}

func TestMarketBreadthDetailFull(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"market-breadth", "--detail", "full"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
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

func TestMarketBreadthTopFlag(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"market-breadth", "--top", "100"})
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
			CoinsCounted int `json:"coins_counted"`
		}
		if err := json.Unmarshal(resp.Results[0].Data, &data); err != nil {
			t.Fatalf("unmarshal data: %v", err)
		}
		if data.CoinsCounted < 1 {
			t.Errorf("CoinsCounted = %d, want >= 1", data.CoinsCounted)
		}
	}
}

func TestMarketBreadthTopFlagClamped(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"market-breadth", "--detail", "extended", "--top", "10"})
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
			t.Error("top_clamped should be true when --top 10 is passed (min is 50)")
		}
	}
}

func TestMarketBreadth_ProviderDirect(t *testing.T) {
	fixture := "["
	for i := 0; i < 60; i++ {
		if i > 0 {
			fixture += ","
		}
		fixture += `{"id":"coin` + string(rune('0'+i%10)) + `","symbol":"sym","price_change_percentage_1h_in_currency":1.0,"price_change_percentage_24h_in_currency":1.0,"price_change_percentage_7d_in_currency":1.0,"price_change_percentage_30d_in_currency":1.0}`
	}
	fixture += "]"

	klinesFixture := `[[1775088000000,"68114.37000000","68304.58000000","68065.69000000","69000.00000000","0.04166000",1775091599999,"2838.36038640",48,"0.01687000","1149.81361760","0"]]`

	p := &mbv1.Provider{}
	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: json.RawMessage(fixture),
		api.BinanceSpotCVD_BTC_1h:       json.RawMessage(klinesFixture),
	}

	result, err := p.Compute(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "ok" {
		t.Errorf("status: got %s, want ok with 60 coins", result.Status)
	}

	var dataOut map[string]interface{}
	if err := json.Unmarshal(result.Data, &dataOut); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}

	if score, ok := dataOut["market_breadth_score"].(float64); !ok || score <= 0 {
		t.Errorf("market_breadth_score should be > 0, got %v", score)
	}
	if _, ok := dataOut["divergence_detected"].(bool); !ok {
		t.Error("data should include divergence_detected")
	}
	if _, ok := dataOut["classification"]; !ok {
		t.Error("data should include classification")
	}
}

func TestMarketBreadth_UnavailableWhenCGEmpty(t *testing.T) {
	p := &mbv1.Provider{}
	data := map[string]json.RawMessage{
		api.CoinGeckoCoinMarketsBreadth: nil,
	}

	result, err := p.Compute(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "unavailable" {
		t.Errorf("status: got %s, want unavailable when CG data is nil", result.Status)
	}
}
