package main

import (
	"encoding/json"
	"testing"
)

// assertCacheFields verifies that cache_hit and ttl_remaining_sec are present
// and correctly typed in a meta JSON blob. Both fields must be present at
// --detail extended and --detail full for all metrics.
func assertCacheFields(t *testing.T, metaJSON json.RawMessage) {
	t.Helper()
	var meta map[string]any
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		t.Fatalf("assertCacheFields: unmarshal meta: %v", err)
	}
	if _, ok := meta["cache_hit"]; !ok {
		t.Error("meta missing cache_hit")
	}
	if _, ok := meta["ttl_remaining_sec"]; !ok {
		t.Error("meta missing ttl_remaining_sec")
	}
	if _, ok := meta["cache_hit"].(bool); !ok {
		t.Errorf("cache_hit must be bool, got %T", meta["cache_hit"])
	}
	if _, ok := meta["ttl_remaining_sec"].(float64); !ok {
		t.Errorf("ttl_remaining_sec must be a number, got %T", meta["ttl_remaining_sec"])
	}
}
