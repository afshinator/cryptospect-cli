package metrics

import "testing"

func TestParseSemVer(t *testing.T) {
	tests := []struct {
		input   string
		want    [3]int
		wantErr bool
	}{
		{"v1.0.0", [3]int{1, 0, 0}, false},
		{"v2.3.14", [3]int{2, 3, 14}, false},
		{"v0.0.0", [3]int{0, 0, 0}, false},
		{"v10.20.30", [3]int{10, 20, 30}, false},
		{"1.0.0", [3]int{}, true},       // missing v prefix
		{"v1.0", [3]int{}, true},        // only 2 parts
		{"v1.0.0.0", [3]int{}, true},    // 4 parts
		{"vx.y.z", [3]int{}, true},      // non-numeric
		{"", [3]int{}, true},            // empty
		{"v-1.0.0", [3]int{}, true},     // negative
		{"v1.0.0-beta", [3]int{}, true}, // pre-release suffix not supported
	}
	for _, tc := range tests {
		got, err := ParseSemVer(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseSemVer(%q) = %v, want error", tc.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSemVer(%q) error = %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseSemVer(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestCompareSemVer(t *testing.T) {
	tests := []struct {
		a, b [3]int
		want int
	}{
		{[3]int{1, 0, 0}, [3]int{1, 0, 0}, 0},
		{[3]int{1, 0, 0}, [3]int{2, 0, 0}, -1},
		{[3]int{2, 0, 0}, [3]int{1, 0, 0}, 1},
		{[3]int{1, 2, 0}, [3]int{1, 3, 0}, -1},
		{[3]int{1, 3, 0}, [3]int{1, 2, 0}, 1},
		{[3]int{1, 0, 1}, [3]int{1, 0, 0}, 1},
		{[3]int{1, 0, 0}, [3]int{1, 0, 1}, -1},
		{[3]int{0, 0, 0}, [3]int{0, 0, 0}, 0},
		{[3]int{1, 9, 9}, [3]int{2, 0, 0}, -1},
		{[3]int{2, 0, 0}, [3]int{1, 9, 9}, 1},
	}
	for _, tc := range tests {
		got := CompareSemVer(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("CompareSemVer(%v, %v) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
