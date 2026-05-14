package main

import (
	"encoding/json"

	"github.com/afshinator/cryptospect-cli/internal/api"
)

// fullDetailOnlyFields lists meta keys that are stripped at --detail extended.
// Any new full-detail-only field added to a metric's Meta struct must be added here.
var fullDetailOnlyFields = []string{
	"thresholds",
	"description",
	"top_n_stablecoins", // stablecoin-power
	"tier_detail",       // momentum-divergence
}

// aggregateFetchMeta computes aggregate cache metadata from per-endpoint fetch results.
// cache_hit is true only when every endpoint was served from a fresh cache entry.
// ttlRemaining is the minimum TTL (seconds) across all endpoints; 0 when cache_hit is false.
func aggregateFetchMeta(metas map[string]api.FetchMeta) (cacheHit bool, ttlRemaining int) {
	if len(metas) == 0 {
		return false, 0
	}
	for _, m := range metas {
		if !m.CacheHit {
			return false, 0
		}
	}
	first := true
	minTTL := 0
	for _, m := range metas {
		if first || m.TTLRemaining < minTTL {
			minTTL = m.TTLRemaining
			first = false
		}
	}
	if minTTL < 0 {
		minTTL = 0
	}
	return true, minTTL
}

// postProcessMeta injects cache_hit and ttl_remaining_sec into the meta JSON blob
// and strips full-detail-only fields when detail is "extended".
// Returns nil if metaJSON is nil. Returns metaJSON unchanged on parse/marshal error.
func postProcessMeta(metaJSON json.RawMessage, detail string, cacheHit bool, ttlRemaining int, fullOnlyFields []string) json.RawMessage {
	if metaJSON == nil {
		return nil
	}
	var meta map[string]any
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		return metaJSON
	}
	meta["cache_hit"] = cacheHit
	meta["ttl_remaining_sec"] = ttlRemaining
	if detail == "extended" {
		for _, f := range fullOnlyFields {
			delete(meta, f)
		}
	}
	out, err := json.Marshal(meta)
	if err != nil {
		return metaJSON
	}
	return out
}
