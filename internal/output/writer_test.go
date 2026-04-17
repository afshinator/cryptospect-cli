package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteSuccess(t *testing.T) {
	// Redirect stdout
	old := Writer()
	defer SetWriter(old)

	var buf bytes.Buffer
	SetWriter(&buf)

	results := []MetricResult{
		{
			Metric: "liquidity-pulse",
			Status: "ok",
			Data:   json.RawMessage(`{"score":0.85}`),
		},
	}

	err := WriteSuccess(results)
	if err != nil {
		t.Fatalf("WriteSuccess error: %v", err)
	}

	// Parse output
	var resp CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("Status = %v, want ok", resp.Status)
	}
	if len(resp.Results) != 1 {
		t.Errorf("Results length = %v, want 1", len(resp.Results))
	}
	if resp.Results[0].Metric != "liquidity-pulse" {
		t.Errorf("Metric = %v, want liquidity-pulse", resp.Results[0].Metric)
	}
}

func TestWriteError(t *testing.T) {
	// Redirect stdout
	old := Writer()
	defer SetWriter(old)

	var buf bytes.Buffer
	SetWriter(&buf)

	err := WriteError(429, "rate_limited", "coingecko", 60)
	if err != nil {
		t.Fatalf("WriteError error: %v", err)
	}

	// Parse output
	var resp CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}

	if resp.Status != "error" {
		t.Errorf("Status = %v, want error", resp.Status)
	}
	if resp.Error == nil {
		t.Fatal("Error field missing")
	}
	if resp.Error.Code != 429 {
		t.Errorf("Error.Code = %v, want 429", resp.Error.Code)
	}
	if resp.Error.Msg != "rate_limited" {
		t.Errorf("Error.Msg = %v, want rate_limited", resp.Error.Msg)
	}
	if resp.Error.Source != "coingecko" {
		t.Errorf("Error.Source = %v, want coingecko", resp.Error.Source)
	}
	if resp.Error.RetryAfterSec != 60 {
		t.Errorf("Error.RetryAfterSec = %v, want 60", resp.Error.RetryAfterSec)
	}
}
