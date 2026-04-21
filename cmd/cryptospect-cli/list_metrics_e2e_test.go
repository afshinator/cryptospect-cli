package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/output"
)

func TestListMetricsCommand(t *testing.T) {
	// Redirect JSON output to buffer
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	// Run list-metrics command
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"list-metrics"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	// Validate JSON envelope
	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("Status = %v, want ok", resp.Status)
	}
	if len(resp.Results) != 1 {
		t.Errorf("Results length = %v, want 1", len(resp.Results))
	}
	if resp.Results[0].Metric != "list-metrics" {
		t.Errorf("Metric = %v, want list-metrics", resp.Results[0].Metric)
	}
	if resp.Results[0].Status != "ok" {
		t.Errorf("Result status = %v, want ok", resp.Results[0].Status)
	}
	// Verify data is a non‑empty array
	var list []interface{}
	if err := json.Unmarshal(resp.Results[0].Data, &list); err != nil {
		t.Fatalf("unmarshal list data: %v", err)
	}
	if len(list) == 0 {
		t.Error("list data is empty")
	}
	// Expect exactly 6 metrics (the default registered ones)
	if len(list) != 6 {
		t.Errorf("expected 6 metrics, got %d", len(list))
	}
	// Ensure each entry has at least "name" and "aliases"
	for i, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			t.Errorf("list entry %d is not an object", i)
			continue
		}
		if _, ok := m["name"]; !ok {
			t.Errorf("list entry %d missing 'name' field", i)
		}
		if _, ok := m["aliases"]; !ok {
			t.Errorf("list entry %d missing 'aliases' field", i)
		}
	}
}

func TestListMetricsOutputJSON(t *testing.T) {
	// Ensure command works with explicit --output json flag
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"list-metrics", "--output", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("Status = %v, want ok", resp.Status)
	}
	if len(resp.Results) != 1 {
		t.Errorf("Results length = %v, want 1", len(resp.Results))
	}
	if resp.Results[0].Metric != "list-metrics" {
		t.Errorf("Metric = %v, want list-metrics", resp.Results[0].Metric)
	}
}
