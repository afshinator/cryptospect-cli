package main

import (
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/api"
)

// ── aggregateFetchMeta ──

func TestAggregateFetchMeta_EmptyMap(t *testing.T) {
	hit, ttl := aggregateFetchMeta(map[string]api.FetchMeta{})
	if hit {
		t.Error("empty map: cache_hit must be false")
	}
	if ttl != 0 {
		t.Errorf("empty map: ttl_remaining must be 0, got %d", ttl)
	}
}

func TestAggregateFetchMeta_SingleCacheHit(t *testing.T) {
	metas := map[string]api.FetchMeta{
		"ep1": {CacheHit: true, TTLRemaining: 900},
	}
	hit, ttl := aggregateFetchMeta(metas)
	if !hit {
		t.Error("single cache hit: cache_hit must be true")
	}
	if ttl != 900 {
		t.Errorf("single cache hit: ttl_remaining = %d, want 900", ttl)
	}
}

func TestAggregateFetchMeta_SingleMiss(t *testing.T) {
	metas := map[string]api.FetchMeta{
		"ep1": {CacheHit: false, TTLRemaining: 3600},
	}
	hit, ttl := aggregateFetchMeta(metas)
	if hit {
		t.Error("single miss: cache_hit must be false")
	}
	if ttl != 0 {
		t.Errorf("single miss: ttl_remaining must be 0, got %d", ttl)
	}
}

func TestAggregateFetchMeta_AllHits_MinTTL(t *testing.T) {
	metas := map[string]api.FetchMeta{
		"ep1": {CacheHit: true, TTLRemaining: 1800},
		"ep2": {CacheHit: true, TTLRemaining: 600},
		"ep3": {CacheHit: true, TTLRemaining: 3200},
	}
	hit, ttl := aggregateFetchMeta(metas)
	if !hit {
		t.Error("all hits: cache_hit must be true")
	}
	if ttl != 600 {
		t.Errorf("all hits: ttl_remaining = %d, want min=600", ttl)
	}
}

func TestAggregateFetchMeta_MixedHitMiss(t *testing.T) {
	metas := map[string]api.FetchMeta{
		"ep1": {CacheHit: true, TTLRemaining: 1200},
		"ep2": {CacheHit: false, TTLRemaining: 3600},
	}
	hit, ttl := aggregateFetchMeta(metas)
	if hit {
		t.Error("mixed: cache_hit must be false when any endpoint is a miss")
	}
	if ttl != 0 {
		t.Errorf("mixed: ttl_remaining must be 0, got %d", ttl)
	}
}

func TestAggregateFetchMeta_NegativeTTL_ClampedToZero(t *testing.T) {
	metas := map[string]api.FetchMeta{
		"ep1": {CacheHit: true, TTLRemaining: -5},
	}
	hit, ttl := aggregateFetchMeta(metas)
	if !hit {
		t.Error("negative TTL: cache_hit should still be true")
	}
	if ttl != 0 {
		t.Errorf("negative TTL: ttl_remaining must be clamped to 0, got %d", ttl)
	}
}

// ── postProcessMeta ──

var testFullOnlyFields = []string{"thresholds", "description", "top_n_stablecoins", "tier_detail"}

func TestPostProcessMeta_NilInput(t *testing.T) {
	result := postProcessMeta(nil, "extended", false, 0, testFullOnlyFields)
	if result != nil {
		t.Error("nil input must return nil")
	}
}

func TestPostProcessMeta_InvalidJSON_PassThrough(t *testing.T) {
	bad := json.RawMessage(`not-json`)
	result := postProcessMeta(bad, "extended", true, 300, testFullOnlyFields)
	if string(result) != string(bad) {
		t.Error("invalid JSON must be returned unchanged")
	}
}

func TestPostProcessMeta_Extended_InjectsCacheFields(t *testing.T) {
	input := json.RawMessage(`{"primary_source":"coingecko","confidence":"high"}`)
	result := postProcessMeta(input, "extended", true, 720, testFullOnlyFields)

	var meta map[string]any
	if err := json.Unmarshal(result, &meta); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if v, ok := meta["cache_hit"].(bool); !ok || !v {
		t.Errorf("cache_hit: got %v, want true", meta["cache_hit"])
	}
	if v, ok := meta["ttl_remaining_sec"].(float64); !ok || v != 720 {
		t.Errorf("ttl_remaining_sec: got %v, want 720", meta["ttl_remaining_sec"])
	}
	// Existing fields preserved
	if meta["primary_source"] != "coingecko" {
		t.Error("existing fields must be preserved")
	}
}

func TestPostProcessMeta_Extended_StripsFullOnlyFields(t *testing.T) {
	input := json.RawMessage(`{
		"confidence":"high",
		"thresholds":{"high":0.15},
		"description":"long text",
		"top_n_stablecoins":[],
		"tier_detail":{}
	}`)
	result := postProcessMeta(input, "extended", false, 0, testFullOnlyFields)

	var meta map[string]any
	if err := json.Unmarshal(result, &meta); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	for _, f := range testFullOnlyFields {
		if _, ok := meta[f]; ok {
			t.Errorf("extended: field %q must be stripped", f)
		}
	}
	if meta["confidence"] != "high" {
		t.Error("non-full-only fields must be preserved")
	}
}

func TestPostProcessMeta_Full_KeepsFullOnlyFields(t *testing.T) {
	input := json.RawMessage(`{
		"confidence":"high",
		"thresholds":{"high":0.15},
		"description":"long text",
		"tier_detail":{}
	}`)
	result := postProcessMeta(input, "full", false, 0, testFullOnlyFields)

	var meta map[string]any
	if err := json.Unmarshal(result, &meta); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := meta["thresholds"]; !ok {
		t.Error("full: thresholds must be present")
	}
	if _, ok := meta["description"]; !ok {
		t.Error("full: description must be present")
	}
	if _, ok := meta["tier_detail"]; !ok {
		t.Error("full: tier_detail must be present")
	}
}

func TestPostProcessMeta_OverwritesExistingCacheFields(t *testing.T) {
	// Verifies MR's internally-computed values get consistently overwritten.
	input := json.RawMessage(`{"cache_hit":true,"ttl_remaining_sec":9999,"confidence":"high"}`)
	result := postProcessMeta(input, "full", false, 0, testFullOnlyFields)

	var meta map[string]any
	if err := json.Unmarshal(result, &meta); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if v, _ := meta["cache_hit"].(bool); v {
		t.Error("cache_hit must be overwritten to false")
	}
	if v, _ := meta["ttl_remaining_sec"].(float64); v != 0 {
		t.Errorf("ttl_remaining_sec must be overwritten to 0, got %v", v)
	}
}

func TestPostProcessMeta_EmptyMeta_InjectsCacheFields(t *testing.T) {
	result := postProcessMeta(json.RawMessage(`{}`), "extended", false, 0, testFullOnlyFields)
	var meta map[string]any
	if err := json.Unmarshal(result, &meta); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := meta["cache_hit"]; !ok {
		t.Error("cache_hit must be present even in empty meta")
	}
	if _, ok := meta["ttl_remaining_sec"]; !ok {
		t.Error("ttl_remaining_sec must be present even in empty meta")
	}
}
