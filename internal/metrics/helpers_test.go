package metrics

import (
	"encoding/json"
	"testing"
)

func TestConfidenceToFloat(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"high", 0.9},
		{"medium", 0.6},
		{"low", 0.3},
		{"unknown", 0.0},
		{"", 0.0},
	}
	for _, tc := range tests {
		got := ConfidenceToFloat(tc.input)
		if got != tc.want {
			t.Errorf("ConfidenceToFloat(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestFloatToConfidence(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{1.0, "high"},
		{0.8, "high"},
		{0.79, "medium"},
		{0.5, "medium"},
		{0.49, "low"},
		{0.0, "low"},
		{-0.1, "low"},
	}
	for _, tc := range tests {
		got := FloatToConfidence(tc.input)
		if got != tc.want {
			t.Errorf("FloatToConfidence(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestUnavailableResult(t *testing.T) {
	result, err := UnavailableResult("my-metric", "v1.0.0", "cryptospect", "test failure")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Metric != "my-metric" {
		t.Errorf("Metric = %q, want %q", result.Metric, "my-metric")
	}
	if result.Version != "v1.0.0" {
		t.Errorf("Version = %q, want %q", result.Version, "v1.0.0")
	}
	if result.Namespace != "cryptospect" {
		t.Errorf("Namespace = %q, want %q", result.Namespace, "cryptospect")
	}
	if result.Status != "unavailable" {
		t.Errorf("Status = %q, want %q", result.Status, "unavailable")
	}
	if result.Meta != nil {
		t.Errorf("Meta should be nil, got %s", string(result.Meta))
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}
	if data["error"] != "test failure" {
		t.Errorf("error message = %q, want %q", data["error"], "test failure")
	}
}

func TestUnavailableResult_AllProviders_Equivalent(t *testing.T) {
	// Prove the helper produces identical output to each provider's old inline version.
	// Each provider's old unavailable() returned: Metric, Version, Status="unavailable", Data={"error":msg}, Meta=nil.
	// The helper adds Namespace which old providers didn't set.
	providers := []struct {
		name, version, namespace string
	}{
		{"liquidity-pulse", "v1.0.0", "cryptospect"},
		{"stablecoin-power", "v1.0.0", "cryptospect"},
		{"flow-tension", "v1.0.0", "cryptospect"},
		{"market-breadth", "v1.0.0", "cryptospect"},
		{"momentum-divergence", "v1.1.0", "cryptospect"},
		{"market-regime", "v1.0.0", "cryptospect"},
	}
	for _, p := range providers {
		result, err := UnavailableResult(p.name, p.version, p.namespace, "test")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", p.name, err)
		}
		if result.Metric != p.name {
			t.Errorf("%s: Metric = %q", p.name, result.Metric)
		}
		if result.Version != p.version {
			t.Errorf("%s: Version = %q", p.name, result.Version)
		}
		if result.Namespace != p.namespace {
			t.Errorf("%s: Namespace = %q", p.name, result.Namespace)
		}
		if result.Status != "unavailable" {
			t.Errorf("%s: Status = %q", p.name, result.Status)
		}
	}
}

func TestDetectStatus_PreservesFloatToConfidence(t *testing.T) {
	// Round-trip: a confidence string → float → DetectStatus should give
	// the expected status mapping from Design-Decisions.md.
	tests := []struct {
		conf     string
		thinData bool
		want     string
	}{
		{"high", false, "ok"},
		{"high", true, "degraded"}, // downgrade
		{"medium", false, "degraded"},
		{"medium", true, "unavailable"}, // downgrade
		{"low", false, "unavailable"},
		{"low", true, "unavailable"},
	}
	for _, tc := range tests {
		f := ConfidenceToFloat(tc.conf)
		got := DetectStatus(f, tc.thinData)
		if got != tc.want {
			t.Errorf("DetectStatus(ConfidenceToFloat(%q)=%v, thinData=%v) = %q, want %q",
				tc.conf, f, tc.thinData, got, tc.want)
		}
	}
}
