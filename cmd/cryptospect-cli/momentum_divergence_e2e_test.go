package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/output"
)

func TestMomentumDivergenceCommand(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"momentum-divergence"})
	_ = cmd.Execute()

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "ok" && resp.Status != "error" {
		t.Errorf("status: got %q, want ok or error", resp.Status)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results: got %d, want 1", len(resp.Results))
	}
	r := resp.Results[0]
	if r.Metric != "momentum-divergence" {
		t.Errorf("metric: got %q, want momentum-divergence", r.Metric)
	}
	if r.Status != "ok" && r.Status != "degraded" && r.Status != "unavailable" {
		t.Errorf("result status: got %q, want ok/degraded/unavailable", r.Status)
	}
	if r.Data == nil {
		t.Error("data should not be nil")
	}
}

func TestMomentumDivergenceAlias(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"md"})
	_ = cmd.Execute()

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results: got %d, want 1", len(resp.Results))
	}
	if resp.Results[0].Metric != "momentum-divergence" {
		t.Errorf("metric: got %q, want momentum-divergence", resp.Results[0].Metric)
	}
}

func TestMomentumDivergenceDetailExtended(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"momentum-divergence", "--detail", "extended"})
	_ = cmd.Execute()

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results: got %d, want 1", len(resp.Results))
	}
	r := resp.Results[0]
	if r.Meta == nil {
		t.Fatal("meta should not be nil at extended detail")
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(r.Meta, &meta); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	if _, ok := meta["thresholds"]; ok {
		t.Error("thresholds should not be present at extended detail")
	}
	if _, ok := meta["description"]; ok {
		t.Error("description should not be present at extended detail")
	}
}

func TestMomentumDivergenceDetailFull(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"momentum-divergence", "--detail", "full"})
	_ = cmd.Execute()

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results: got %d, want 1", len(resp.Results))
	}
	r := resp.Results[0]
	if r.Meta == nil {
		t.Fatal("meta should not be nil at full detail")
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(r.Meta, &meta); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	if _, ok := meta["thresholds"]; !ok {
		t.Error("thresholds should be present at full detail")
	}
}

func TestMomentumDivergenceSegmentsFlag(t *testing.T) {
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"momentum-divergence", "--segments", "5,20,100", "--detail", "extended"})
	_ = cmd.Execute()

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results: got %d, want 1", len(resp.Results))
	}
	r := resp.Results[0]
	if r.Meta == nil {
		t.Fatal("meta should not be nil")
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(r.Meta, &meta); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	segUsed, ok := meta["segments_used"].(map[string]interface{})
	if !ok {
		t.Fatal("segments_used should be present in meta")
	}
	if lc, ok := segUsed["large_ceiling"].(float64); !ok || int(lc) != 5 {
		t.Errorf("large_ceiling: got %v, want 5", lc)
	}
	if mc, ok := segUsed["mid_ceiling"].(float64); !ok || int(mc) != 20 {
		t.Errorf("mid_ceiling: got %v, want 20", mc)
	}
	if sc, ok := segUsed["small_ceiling"].(float64); !ok || int(sc) != 100 {
		t.Errorf("small_ceiling: got %v, want 100", sc)
	}
}
