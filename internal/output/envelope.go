package output

import "encoding/json"

// CLIResponse is the top-level envelope returned by all commands.
type CLIResponse struct {
	// Status is "ok" for successful responses, "error" for unrecoverable failures.
	Status string `json:"status"`
	// TS is Unix seconds when the response was created.
	TS int64 `json:"ts"`
	// Results contains zero or more metric results. For single-metric commands,
	// this slice has one element.
	Results []MetricResult `json:"results,omitempty"`
	// Error is present when Status is "error". Omitted on success.
	Error *CLIError `json:"error,omitempty"`
}

// MetricResult holds the output of a single metric computation.
type MetricResult struct {
	// Metric is the canonical metric name, e.g., "liquidity-pulse".
	Metric string `json:"metric"`
	// Version is the SemVer of the provider, e.g., "v1.0.0".
	Version string `json:"version"`
	// Namespace identifies the provider namespace, e.g., "cryptospect".
	Namespace string `json:"namespace,omitempty"`
	// Status is "ok", "degraded", or "unavailable" for this specific metric.
	Status string `json:"status"`
	// Data contains the metric-specific payload as raw JSON.
	Data json.RawMessage `json:"data"`
	// Meta contains metadata (cache hit, sources, thresholds, description).
	// Omitted when --detail basic.
	Meta json.RawMessage `json:"meta,omitempty"`
}

// CLIError describes an unrecoverable failure.
type CLIError struct {
	// Code is an HTTP status code or custom error code.
	Code int `json:"code"`
	// Msg is a short machine-readable label, e.g., "rate_limited".
	Msg string `json:"msg"`
	// RetryAfterSec hints how long to wait before retrying (rate limits).
	RetryAfterSec int `json:"retry_after_sec,omitempty"`
	// Source indicates which API failed, e.g., "coingecko".
	Source string `json:"source,omitempty"`
}
