package metrics

import "testing"

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
