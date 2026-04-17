package cache

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpen(t *testing.T) {
	dir := t.TempDir()

	c, err := Open(dir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if c == nil {
		t.Fatal("Open returned nil cache")
	}
	defer func() { _ = c.Close() }()

	// Opening non-existent directory should create it
	nonexistent := filepath.Join(dir, "subdir")
	c2, err := Open(nonexistent)
	if err != nil {
		t.Fatalf("Open nonexistent failed: %v", err)
	}
	_ = c2.Close()
	if _, err := os.Stat(nonexistent); err != nil {
		t.Errorf("subdirectory not created: %v", err)
	}

	// Opening file (not directory) should fail
	filePath := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Open(filePath)
	if err == nil {
		t.Error("Open with file path should fail")
	}
}

func TestGetSet(t *testing.T) {
	dir := t.TempDir()
	c, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	endpoint := "coingecko_global_market"
	data := []byte(`{"total_volume":5000000000}`)

	// Get non-existent entry
	entry, err := c.Get(endpoint)
	if err != nil {
		t.Fatalf("Get non-existent failed: %v", err)
	}
	if entry.Found {
		t.Error("Get should return Found=false for non-existent")
	}

	// Set entry
	ttl := 300
	if err := c.Set(endpoint, data, ttl); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get entry (should be fresh)
	entry, err = c.Get(endpoint)
	if err != nil {
		t.Fatalf("Get after Set failed: %v", err)
	}
	if !entry.Found {
		t.Error("Get should return Found=true")
	}
	if !bytes.Equal(entry.Data, data) {
		t.Errorf("Data = %q, want %q", entry.Data, data)
	}
	if entry.TTLSeconds != ttl {
		t.Errorf("TTLSeconds = %v, want %v", entry.TTLSeconds, ttl)
	}
	if entry.Stale {
		t.Error("Entry should not be stale immediately")
	}
	if entry.FetchedAt.IsZero() {
		t.Error("FetchedAt should be set")
	}
}

func TestStaleEntry(t *testing.T) {
	dir := t.TempDir()
	c, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	endpoint := "coingecko_coin_markets"
	data := []byte(`[{"id":"bitcoin"}]`)

	// Set with very short TTL (1 second)
	if err := c.Set(endpoint, data, 1); err != nil {
		t.Fatal(err)
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	entry, err := c.Get(endpoint)
	if err != nil {
		t.Fatalf("Get after expiration failed: %v", err)
	}
	if !entry.Found {
		t.Error("Expired entry should still be Found=true")
	}
	if !entry.Stale {
		t.Error("Expired entry should be Stale=true")
	}
}

func TestClear(t *testing.T) {
	dir := t.TempDir()
	c, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	// Set a few entries
	if err := c.Set("endpoint1", []byte("data1"), 300); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("endpoint2", []byte("data2"), 300); err != nil {
		t.Fatal(err)
	}

	// Verify they exist
	entry1, _ := c.Get("endpoint1")
	if !entry1.Found {
		t.Error("entry1 should exist before Clear")
	}

	// Clear cache
	if err := c.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify cleared
	entry1, _ = c.Get("endpoint1")
	if entry1.Found {
		t.Error("entry1 should not exist after Clear")
	}
	entry2, _ := c.Get("endpoint2")
	if entry2.Found {
		t.Error("entry2 should not exist after Clear")
	}

	// .gitkeep should remain (if present)
	gitkeepPath := filepath.Join(dir, ".gitkeep")
	if err := os.WriteFile(gitkeepPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := c.Clear(); err != nil {
		t.Fatalf("Clear with .gitkeep failed: %v", err)
	}
	if _, err := os.Stat(gitkeepPath); err != nil {
		t.Error(".gitkeep should not be removed by Clear")
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	c, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	endpoint := "test_endpoint"
	data1 := []byte("first")

	// Simulate concurrent write by creating temp file manually
	path := c.FilePath(endpoint)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Temp file exists, Set should still succeed (overwrites temp)
	if err := c.Set(endpoint, data1, 300); err != nil {
		t.Fatalf("Set with existing temp file failed: %v", err)
	}
	// Verify data1 written
	entry, _ := c.Get(endpoint)
	if !bytes.Equal(entry.Data, []byte("first")) {
		t.Errorf("Data = %q, want 'first'", entry.Data)
	}
}

func TestEndpointWithDots(t *testing.T) {
	dir := t.TempDir()
	c, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	// Endpoint keys with dots should be safe as filenames
	endpoint := "coingecko.global_market"
	data := []byte(`{"key":"value"}`)
	if err := c.Set(endpoint, data, 300); err != nil {
		t.Fatalf("Set with dots failed: %v", err)
	}
	entry, err := c.Get(endpoint)
	if err != nil {
		t.Fatalf("Get with dots failed: %v", err)
	}
	if !entry.Found {
		t.Error("Entry with dots not found")
	}
	if !bytes.Equal(entry.Data, data) {
		t.Errorf("Data mismatch for dotted endpoint")
	}
}
