package main

import (
	"encoding/json"
	"testing"
)

func TestChinaM2Command_E2E(t *testing.T) {
	resp := runCLI(t, "china-m2")

	assertSingleResult(t, resp)

	res := resp.Results[0]

	if res.Metric != "china-m2" {
		t.Fatalf("Metric = %v, want china-m2", res.Metric)
	}

	if res.Status != "ok" && res.Status != "degraded" && res.Status != "unavailable" {
		t.Fatalf("unexpected status: %v", res.Status)
	}

	if res.Status == "ok" || res.Status == "degraded" {
		var data struct {
			M2             float64 `json:"m2"`
			Classification struct {
				Label string `json:"label"`
			} `json:"classification"`
		}
		if err := json.Unmarshal(res.Data, &data); err != nil {
			t.Fatalf("unmarshal data: %v", err)
		}
		if data.Classification.Label == "" {
			t.Fatal("classification label empty")
		}
	}
}

func TestChinaM2Alias_E2E(t *testing.T) {
	resp := runCLI(t, "cnm2")
	assertSingleResult(t, resp)
	if resp.Results[0].Metric != "china-m2" {
		t.Fatalf("alias failed, got %v", resp.Results[0].Metric)
	}
}

func TestChinaM2DetailFull_E2E(t *testing.T) {
	resp := runCLI(t, "cnm2", "--detail", "full")
	assertSingleResult(t, resp)
	res := resp.Results[0]
	if (res.Status == "ok" || res.Status == "degraded") && res.Meta == nil {
		t.Fatal("expected meta for full detail")
	}
	if res.Meta != nil {
		assertCacheFields(t, res.Meta)
	}
}
