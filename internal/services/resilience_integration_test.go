package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"audnexus-provider/internal/api"
	"audnexus-provider/internal/config"
	"audnexus-provider/internal/models"
)

func TestSearchBooks_SingleflightCoalescing(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		time.Sleep(30 * time.Millisecond) // Simulate slow network

		resp := models.AudibleSearchResponse{
			Products: []models.AudibleProduct{
				{
					ASIN:  "B08G9PRS1K",
					Title: "Project Hail Mary",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{Region: "us"}
	audibleClient := api.NewAudibleClientWithBaseURL(5, server.URL)
	searchService := NewResilientSearchService(nil, audibleClient, nil, nil, cfg)

	// Launch 10 concurrent requests for the exact same query
	const concurrentRequests = 10
	errChan := make(chan error, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		go func() {
			results, err := searchService.SearchBooks(context.Background(), "Project Hail Mary", "Andy Weir")
			if err != nil {
				errChan <- err
				return
			}
			if len(results) == 0 {
				errChan <- errors.New("expected results")
				return
			}
			errChan <- nil
		}()
	}

	for i := 0; i < concurrentRequests; i++ {
		if err := <-errChan; err != nil {
			t.Fatalf("request failed: %v", err)
		}
	}

	// Singleflight should ensure only 1 outbound HTTP call was made!
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected singleflight to coalesce into 1 HTTP call, but got %d", requestCount)
	}
}

func TestSearchBooks_CircuitBreaker_FastFailToFallback(t *testing.T) {
	// Audible always fails
	var audibleCalls int32
	audibleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&audibleCalls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer audibleServer.Close()

	// iTunes succeeds
	itunesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.ITunesSearchResponse{
			ResultCount: 1,
			Results: []models.ITunesResult{
				{
					TrackID:    123,
					TrackName:  "Dune",
					ArtistName: "Frank Herbert",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer itunesServer.Close()

	cfg := &config.Config{Region: "us"}
	audibleClient := api.NewAudibleClientWithBaseURL(5, audibleServer.URL)
	itunesClient := api.NewITunesClientWithBaseURL(5, itunesServer.URL)
	searchService := NewResilientSearchService(nil, audibleClient, itunesClient, nil, cfg)

	// Trip Audible circuit breaker by failing 3 times
	for i := 0; i < 3; i++ {
		_, _ = searchService.SearchBooks(context.Background(), "Dune", "Frank Herbert")
	}

	audibleCallsBefore := atomic.LoadInt32(&audibleCalls)

	// Circuit should now be open! Next call should fast-fail Audible (0 additional calls) and go straight to iTunes
	start := time.Now()
	results, err := searchService.SearchBooks(context.Background(), "Dune", "Frank Herbert")
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 || results[0].Result.Title != "Dune" {
		t.Fatalf("expected iTunes result 'Dune', got: %+v", results)
	}

	audibleCallsAfter := atomic.LoadInt32(&audibleCalls)
	if audibleCallsAfter != audibleCallsBefore {
		t.Errorf("expected 0 calls to Audible when circuit is open, but calls increased from %d to %d", audibleCallsBefore, audibleCallsAfter)
	}
	if duration > 100*time.Millisecond {
		t.Errorf("expected fast-fail under 100ms, took %v", duration)
	}
}

func TestSearchBooks_AllUpstreamsFail_SyntheticFallback(t *testing.T) {
	// All servers fail
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failingServer.Close()

	cfg := &config.Config{
		Region:            "us",
		StoreAuthorAsMood: true,
	}
	audibleClient := api.NewAudibleClientWithBaseURL(5, failingServer.URL)
	itunesClient := api.NewITunesClientWithBaseURL(5, failingServer.URL)
	openLibClient := api.NewOpenLibraryClientWithBaseURL(5, failingServer.URL)

	searchService := NewResilientSearchService(nil, audibleClient, itunesClient, openLibClient, cfg)

	results, err := searchService.SearchBooks(context.Background(), "Rare Private Audiobook", "Unknown Narrator")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 synthetic result, got %d", len(results))
	}
	if results[0].Result.Title != "Rare Private Audiobook" {
		t.Errorf("expected synthetic title, got %s", results[0].Result.Title)
	}
	if results[0].Score < 45 {
		t.Errorf("expected synthetic score >= 45, got %d", results[0].Score)
	}
}
