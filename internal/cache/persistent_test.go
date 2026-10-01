package cache

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestPersistentCache_SetAndGet(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audnexus-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	c, err := NewPersistentCache(tempDir, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to create persistent cache: %v", err)
	}

	key := "album_test1"
	val := map[string]string{"title": "Test Title"}

	c.Set(key, val)

	var retrieved map[string]string
	found, stale := c.Get(key, &retrieved)
	if !found {
		t.Fatal("expected item to be found in cache")
	}
	if stale {
		t.Error("expected fresh item, but got stale")
	}
	if retrieved["title"] != "Test Title" {
		t.Errorf("expected Test Title, got %v", retrieved["title"])
	}
}

func TestPersistentCache_SurvivesRestart(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audnexus-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	c1, err := NewPersistentCache(tempDir, 10*time.Second)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	c1.Set("key123", "persisted_value")

	// Verify file was written to disk
	entries, _ := os.ReadDir(tempDir)
	if len(entries) == 0 {
		t.Fatal("expected cache file on disk")
	}

	// Create a new independent cache instance pointing to the same directory
	c2, err := NewPersistentCache(tempDir, 10*time.Second)
	if err != nil {
		t.Fatalf("failed to create second cache: %v", err)
	}

	var loadedVal string
	found, stale := c2.Get("key123", &loadedVal)
	if !found {
		t.Fatal("expected item to be recovered from disk")
	}
	if stale {
		t.Error("expected fresh item from disk")
	}
	if loadedVal != "persisted_value" {
		t.Errorf("expected persisted_value, got %s", loadedVal)
	}
}

func TestPersistentCache_StaleWhileRevalidate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audnexus-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ttl := 20 * time.Millisecond
	c, err := NewPersistentCache(tempDir, ttl)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	key := "test_swr"
	c.Set(key, "old_value")

	// Wait for TTL to expire
	time.Sleep(ttl + 10*time.Millisecond)

	// Case 1: Stale item, fetcher fails -> should return stale value gracefully!
	var result string
	fetchErr := errors.New("upstream offline")
	err = c.GetOrFetch(key, &result, func() (interface{}, error) {
		return nil, fetchErr
	})
	if err != nil {
		t.Fatalf("expected nil error on stale fallback, got: %v", err)
	}
	if result != "old_value" {
		t.Errorf("expected stale value 'old_value', got: %s", result)
	}

	// Case 2: Stale item, fetcher succeeds -> should return updated value
	err = c.GetOrFetch(key, &result, func() (interface{}, error) {
		return "new_value", nil
	})
	if err != nil {
		t.Fatalf("unexpected fetcher error: %v", err)
	}
	if result != "new_value" {
		t.Errorf("expected updated value 'new_value', got: %s", result)
	}
}
