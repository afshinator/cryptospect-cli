package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry represents a cached entry returned by Get.
type Entry struct {
	Data       []byte    `json:"data"`
	Found      bool      `json:"found"`
	Stale      bool      `json:"stale"`
	FetchedAt  time.Time `json:"fetched_at"`
	TTLSeconds int       `json:"ttl_seconds"`
}

type record struct {
	Data       []byte    `json:"data"`
	FetchedAt  time.Time `json:"fetched_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	TTLSeconds int       `json:"ttl_seconds"`
}

// Cache provides file‑based caching with TTL and stale‑while‑revalidate semantics.
type Cache struct {
	dir string
}

// Exists returns true if the cache directory exists and is a directory.
func Exists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking cache dir: %w", err)
	}
	return info.IsDir(), nil
}

// Open opens (or creates) a cache directory at the given path.
func Open(path string) (*Cache, error) {
	info, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("checking cache dir: %w", err)
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			return nil, fmt.Errorf("creating cache dir: %w", err)
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("cache path exists but is not a directory: %s", path)
	}
	return &Cache{dir: path}, nil
}

// Close closes the cache (no‑op for file‑based cache).
func (c *Cache) Close() error {
	c.dir = ""
	return nil
}

func (c *Cache) filePath(endpoint string) string {
	// endpoint may contain dots; filepath.Join will treat them as part of the filename
	safeName := filepath.Join(c.dir, endpoint+".json")
	return safeName
}

// Get retrieves a cached entry for the given endpoint.
func (c *Cache) Get(endpoint string) (Entry, error) {
	var entry Entry
	path := c.filePath(endpoint)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Entry{Found: false}, nil
		}
		return Entry{}, fmt.Errorf("reading cache file: %w", err)
	}

	var rec record
	if err := json.Unmarshal(data, &rec); err != nil {
		return Entry{}, fmt.Errorf("unmarshalling cache record: %w", err)
	}

	entry.Found = true
	entry.Data = rec.Data
	entry.FetchedAt = rec.FetchedAt
	entry.TTLSeconds = rec.TTLSeconds
	entry.Stale = time.Now().After(rec.ExpiresAt)

	return entry, nil
}

// Set writes data for the given endpoint with the specified TTL (seconds).
func (c *Cache) Set(endpoint string, data []byte, ttl int) error {
	now := time.Now()
	rec := record{
		Data:       data,
		FetchedAt:  now,
		ExpiresAt:  now.Add(time.Duration(ttl) * time.Second),
		TTLSeconds: ttl,
	}

	encoded, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshalling cache record: %w", err)
	}

	path := c.filePath(endpoint)
	tmpPath := path + ".tmp"

	// Write to temp file first
	if err := os.WriteFile(tmpPath, encoded, 0o600); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("atomic rename: %w", err)
	}

	return nil
}

// Clear removes all cached entries from the cache directory.
func (c *Cache) Clear() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return fmt.Errorf("reading cache dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == ".gitkeep" {
			continue
		}
		path := filepath.Join(c.dir, name)
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing %s: %w", name, err)
		}
	}

	return nil
}

// FilePath returns the full path for an endpoint (used by tests).
func (c *Cache) FilePath(endpoint string) string {
	return c.filePath(endpoint)
}
