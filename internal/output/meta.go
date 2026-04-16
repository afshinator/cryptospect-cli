package output

// MetaBasic contains basic metadata about the metric computation.
type MetaBasic struct {
	// CacheHit indicates whether the result was served from cache.
	CacheHit bool `json:"cache_hit"`
	// TTLRemaining is seconds until cached data expires (0 if not cached).
	TTLRemaining int `json:"ttl_remaining"`
}

// SourceMeta describes a single data source used in the computation.
type SourceMeta struct {
	// Endpoint is the endpoint key (e.g., "coingecko.global").
	Endpoint string `json:"endpoint"`
	// Timestamp is Unix seconds when the source data was fetched.
	Timestamp int64 `json:"timestamp"`
	// CacheHit indicates whether this specific source came from cache.
	CacheHit bool `json:"cache_hit"`
}

// MetaExtended extends MetaBasic with source-level metadata.
type MetaExtended struct {
	MetaBasic
	// Sources maps datapoint names to their source metadata.
	Sources map[string]SourceMeta `json:"sources"`
}

// MetaFull extends MetaExtended with thresholds and description.
type MetaFull struct {
	MetaExtended
	// Thresholds contains metric-specific threshold values.
	Thresholds map[string]float64 `json:"thresholds,omitempty"`
	// Description is a human-readable description of the metric.
	Description string `json:"description,omitempty"`
}
