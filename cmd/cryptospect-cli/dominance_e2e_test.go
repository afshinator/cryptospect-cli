package main

import (
    "encoding/json"
    "testing"
)

func TestDominanceCommand_E2E(t *testing.T) {
    resp := runCLI(t, "dominance")

    assertSingleResult(t, resp)

    res := resp.Results[0]

    if res.Metric != "dominance" {
        t.Fatalf("Metric = %v, want dominance", res.Metric)
    }

    if res.Status != "ok" && res.Status != "degraded" && res.Status != "unavailable" {
        t.Fatalf("unexpected status: %v", res.Status)
    }

    if res.Status == "ok" || res.Status == "degraded" {
        var data struct {
            BTC float64 `json:"btc_dominance"`
            ETH float64 `json:"eth_dominance"`
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
