package main

import (
    "encoding/json"
    "testing"
)

func TestVolatilityCommand_E2E(t *testing.T) {
    resp := runCLI(t, "volatility")

    assertSingleResult(t, resp)

    res := resp.Results[0]

    if res.Metric != "volatility" {
        t.Fatalf("Metric = %v, want volatility", res.Metric)
    }

    if res.Status != "ok" && res.Status != "degraded" && res.Status != "unavailable" {
        t.Fatalf("unexpected status: %v", res.Status)
    }

    if res.Status == "ok" || res.Status == "degraded" {
        var data struct {
            Volatility float64 `json:"volatility"`
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
