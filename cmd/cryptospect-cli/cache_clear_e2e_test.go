package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/afshinator/cryptospect-cli/internal/output"
)

func TestCacheClearCommand(t *testing.T) {
	// Create temporary cache directory
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a dummy cache file
	dummyFile := filepath.Join(cacheDir, "test.cache")
	if err := os.WriteFile(dummyFile, []byte("test data"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create temporary config file with cache dir
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `cache:
  enabled: true
  dir: "` + cacheDir + `"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0o600); err != nil {
		t.Fatal(err)
	}

	// Redirect JSON output to buffer
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	// Run cache-clear command with --config flag
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--config", configFile, "cache-clear"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	// Verify JSON envelope
	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("Status = %v, want ok", resp.Status)
	}
	if len(resp.Results) != 1 {
		t.Errorf("Results length = %v, want 1", len(resp.Results))
	}
	if resp.Results[0].Metric != "cache-clear" {
		t.Errorf("Metric = %v, want cache-clear", resp.Results[0].Metric)
	}
	if resp.Results[0].Status != "ok" {
		t.Errorf("Result status = %v, want ok", resp.Results[0].Status)
	}

	// Verify cache directory is empty (except .gitkeep maybe)
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != ".gitkeep" {
			t.Errorf("unexpected file left in cache: %s", e.Name())
		}
	}
}

func TestCacheClearOutputJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `cache:
  enabled: true
  dir: "` + cacheDir + `"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0o600); err != nil {
		t.Fatal(err)
	}

	// Redirect JSON output to buffer
	oldWriter := output.Writer()
	defer output.SetWriter(oldWriter)
	var buf bytes.Buffer
	output.SetWriter(&buf)

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--config", configFile, "cache-clear", "--output", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	// Validate JSON envelope
	var resp output.CLIResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("Status = %v, want ok", resp.Status)
	}
	if len(resp.Results) != 1 {
		t.Errorf("Results length = %v, want 1", len(resp.Results))
	}
	if resp.Results[0].Metric != "cache-clear" {
		t.Errorf("Metric = %v, want cache-clear", resp.Results[0].Metric)
	}
	if resp.Results[0].Status != "ok" {
		t.Errorf("Result status = %v, want ok", resp.Results[0].Status)
	}
}

func TestCacheClearWithDefaultCacheDir(t *testing.T) {
	// Test without explicit cache dir in config; should default to ~/.cryptospect-cli/cache
	// but we don't want to touch home directory. Instead, we can test that the command
	// fails with a clear error when cache directory cannot be determined.
	// We'll skip this for now.
	t.Skip("default cache directory test requires mocking home dir")
}
