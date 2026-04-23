package metrics

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseSemVer parses a SemVer string with a required "v" prefix (e.g. "v1.2.3").
// Returns [major, minor, patch] or an error for any other format.
func ParseSemVer(s string) ([3]int, error) {
	if !strings.HasPrefix(s, "v") {
		return [3]int{}, fmt.Errorf("semver %q must start with 'v'", s)
	}
	parts := strings.Split(s[1:], ".")
	if len(parts) != 3 {
		return [3]int{}, fmt.Errorf("semver %q must have exactly 3 parts (major.minor.patch)", s)
	}
	var out [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return [3]int{}, fmt.Errorf("semver %q part %d is not a non-negative integer: %q", s, i, p)
		}
		out[i] = n
	}
	return out, nil
}

// CompareSemVer compares two parsed SemVer triples.
// Returns -1 if a < b, 0 if equal, 1 if a > b.
func CompareSemVer(a, b [3]int) int {
	for i := range 3 {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
