package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/output"
)

// runCLI executes the CLI with args and returns parsed CLIResponse.
// This is intentionally minimal and mirrors existing E2E patterns.
func runCLI(t *testing.T, args ...string) output.CLIResponse {
	t.Helper()

	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)

	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}

	return resp
}

func assertSingleResult(t *testing.T, resp output.CLIResponse) {
	t.Helper()
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
}

// assertCacheFields verifies common cache metadata fields exist.
func assertCacheFields(t *testing.T, meta json.RawMessage) {
	t.Helper()
	if len(meta) == 0 {
		t.Fatalf("meta is empty")
	}
	var m map[string]any
	if err := json.Unmarshal(meta, &m); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	if _, ok := m["cache_hit"]; !ok {
		t.Fatalf("meta missing cache_hit")
	}
	if _, ok := m["ttl_remaining_sec"]; !ok {
		t.Fatalf("meta missing ttl_remaining_sec")
	}
}
