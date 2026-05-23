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
            M2 float64 `json:"m2"`
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
