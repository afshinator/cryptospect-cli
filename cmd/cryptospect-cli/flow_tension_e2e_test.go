package main

import (
	"bytes"
	"encoding/json"
	"testing"

	_ "github.com/afshinator/cryptospect-cli/internal/metrics/flowtension/v1"
	"github.com/afshinator/cryptospect-cli/internal/output"
)

func TestFlowTensionCommand(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"flow-tension"})
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
	if resp.Results[0].Metric != "flow-tension" {
		t.Errorf("Metric = %v, want flow-tension", resp.Results[0].Metric)
	}

	// Accept any valid status — live API may have transient failures in CI.
	st := resp.Results[0].Status
	if st != "ok" && st != "degraded" && st != "unavailable" {
		t.Errorf("Result status = %v, want ok/degraded/unavailable", st)
	}

	// Only validate data shape when we got real or degraded data.
	if st == "ok" || st == "degraded" {
		var data struct {
			Signals struct {
				CVD struct {
					Ratio float64 `json:"ratio"`
					Hook  string  `json:"hook"`
				} `json:"cvd"`
				OpenInterest struct {
					CurrentUSD    float64  `json:"current_usd"`
					ChangePct24h  *float64 `json:"change_pct_24h,omitempty"`
					ExchangeCount int      `json:"exchange_count"`
					Hook          string   `json:"hook"`
				} `json:"open_interest"`
				FundingRate struct {
					Rate float64 `json:"rate"`
					Hook string  `json:"hook"`
				} `json:"funding_rate"`
			} `json:"signals"`
			Summary string `json:"summary"`
		}
		if err := json.Unmarshal(resp.Results[0].Data, &data); err != nil {
			t.Fatalf("unmarshal flow-tension data: %v", err)
		}
		if data.Signals.CVD.Hook == "" {
			t.Error("CVD hook must not be empty")
		}
		if data.Summary == "" {
			t.Error("summary must not be empty")
		}
	}
}

func TestFlowTensionAlias(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"ft"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if resp.Results[0].Metric != "flow-tension" {
		t.Errorf("Metric = %v, want flow-tension via alias ft", resp.Results[0].Metric)
	}
}

func TestFlowTensionDetailExtended(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"flow-tension", "--detail", "extended"})
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

func TestFlowTensionDetailFull(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"flow-tension", "--detail", "full"})
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
