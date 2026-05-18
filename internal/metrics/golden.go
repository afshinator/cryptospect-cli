package metrics

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// UpdateGolden is a flag that, when set, writes actual output to golden files
// instead of comparing. Usage: go test -update.
var UpdateGolden = flag.Bool("update", false, "update golden files")

// AssertMatchesGolden compares actual JSON output against a golden file.
// Both sides are normalised (unmarshal→marshal) to ignore key ordering.
// When -update is set, writes actual to goldenPath instead.
func AssertMatchesGolden(t *testing.T, goldenPath string, actual []byte) {
	t.Helper()

	normalized, err := NormaliseJSON(actual)
	if err != nil {
		t.Fatalf("normalising actual JSON: %v", err)
	}

	if *UpdateGolden {
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir golden dir %s: %v", dir, err)
		}
		if err := os.WriteFile(goldenPath, normalized, 0o644); err != nil {
			t.Fatalf("writing golden file %s: %v", goldenPath, err)
		}
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden file %s: %v (use -update to create)", goldenPath, err)
	}

	expectedNorm, err := NormaliseJSON(expected)
	if err != nil {
		t.Fatalf("normalising expected JSON: %v", err)
	}

	if !bytes.Equal(normalized, expectedNorm) {
		t.Errorf("golden mismatch: %s\n--- expected ---\n%s\n--- actual ---\n%s",
			goldenPath, string(expectedNorm), string(normalized))
	}
}

// NormaliseJSON unmarshal→marshal JSON into canonical form, ignoring key ordering.
func NormaliseJSON(b []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return json.MarshalIndent(v, "", "  ")
}
