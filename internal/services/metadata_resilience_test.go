package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"audnexus-provider/internal/api"
	"audnexus-provider/internal/cache"
	"audnexus-provider/internal/config"
	"audnexus-provider/internal/models"
)

func TestGetBookMetadata_StaleWhileRevalidate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "metadata-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ttl := 30 * time.Millisecond
	persistentCache, err := cache.NewPersistentCache(tempDir, ttl)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	shouldFail := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldFail {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		book := models.Book{
			ASIN:        "B08G9PRS1K",
			Title:       "Project Hail Mary",
			Summary:     "Great book",
			ReleaseDate: "2021-05-04",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(book)
	}))
	defer server.Close()

	cfg := &config.Config{Region: "us"}
	audnexusClient := api.NewClientWithBaseURL(5, server.URL)
	metadataService := NewResilientMetadataService(audnexusClient, nil, nil, persistentCache, cfg)

	// 1. Initial fetch succeeds and populates cache
	meta1, err := metadataService.GetBookMetadata(context.Background(), "B08G9PRS1K")
	if err != nil {
		t.Fatalf("unexpected error on initial fetch: %v", err)
	}
	if meta1.Title != "Project Hail Mary" {
		t.Errorf("expected Title Project Hail Mary, got %s", meta1.Title)
	}

	// 2. Upstream server goes down, and cache expires
	shouldFail = true
	time.Sleep(ttl + 10*time.Millisecond)

	// 3. Stale-while-revalidate should return cached metadata rather than failing!
	meta2, err := metadataService.GetBookMetadata(context.Background(), "B08G9PRS1K")
	if err != nil {
		t.Fatalf("expected graceful fallback to stale cache, but got error: %v", err)
	}
	if meta2.Title != "Project Hail Mary" {
		t.Errorf("expected stale cached title 'Project Hail Mary', got %s", meta2.Title)
	}
}

func TestGetBookMetadata_SyntheticLocalItem(t *testing.T) {
	cfg := &config.Config{Region: "us"}
	metadataService := NewResilientMetadataService(nil, nil, nil, nil, cfg)

	// A local synthetic ID should resolve without network calls
	meta, err := metadataService.GetBookMetadata(context.Background(), "local_abc12345")
	if err != nil {
		t.Fatalf("unexpected error for synthetic item: %v", err)
	}
	if meta.RatingKey != "local_abc12345" {
		t.Errorf("expected rating key local_abc12345, got %s", meta.RatingKey)
	}
}
