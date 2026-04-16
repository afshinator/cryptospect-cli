package metrics

import "testing"

func TestDetectStatus(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		thinData   bool
		want       string
	}{
		// Without thinData
		{name: "high confidence", confidence: 0.9, thinData: false, want: "ok"},
		{name: "exact 0.8 boundary", confidence: 0.8, thinData: false, want: "ok"},
		{name: "medium confidence", confidence: 0.7, thinData: false, want: "degraded"},
		{name: "exact 0.5 boundary", confidence: 0.5, thinData: false, want: "degraded"},
		{name: "low confidence", confidence: 0.3, thinData: false, want: "unavailable"},
		{name: "zero confidence", confidence: 0.0, thinData: false, want: "unavailable"},

		// With thinData (downgrade one level)
		{name: "high confidence with thinData", confidence: 0.9, thinData: true, want: "degraded"},
		{name: "0.8 with thinData", confidence: 0.8, thinData: true, want: "degraded"},
		{name: "medium confidence with thinData", confidence: 0.7, thinData: true, want: "unavailable"},
		{name: "0.5 with thinData", confidence: 0.5, thinData: true, want: "unavailable"},
		{name: "low confidence with thinData", confidence: 0.3, thinData: true, want: "unavailable"},
		{name: "zero confidence with thinData", confidence: 0.0, thinData: true, want: "unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectStatus(tt.confidence, tt.thinData)
			if got != tt.want {
				t.Errorf("detectStatus(%v, %v) = %v, want %v", tt.confidence, tt.thinData, got, tt.want)
			}
		})
	}
}
