package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"audnexus-provider/internal/api"
	"audnexus-provider/internal/config"
	"audnexus-provider/internal/models"
)

func TestSearchBooks_Audible(t *testing.T) {
	audibleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.AudibleSearchResponse{
			Products: []models.AudibleProduct{
				{
					ASIN:        "B08G9PRS1K",
					Title:       "Project Hail Mary",
					Authors:     []models.Contributor{{Name: "Andy Weir"}},
					Narrators:   []models.Contributor{{Name: "Ray Porter"}},
					ReleaseDate: "2021-05-04",
					Publisher:   "Audible Studios",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer audibleServer.Close()

	cfg := &config.Config{
		Region:            "us",
		StoreAuthorAsMood: true,
	}

	audibleClient := api.NewAudibleClientWithBaseURL(5, audibleServer.URL)
	searchService := NewSearchService(nil, audibleClient, nil, cfg)

	results, err := searchService.SearchBooks(context.Background(), "Project Hail Mary", "Andy Weir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	res := results[0]
	if res.Result.RatingKey != "album_B08G9PRS1K" {
		t.Errorf("expected RatingKey album_B08G9PRS1K, got %s", res.Result.RatingKey)
	}
	if res.Result.Title != "Project Hail Mary" {
		t.Errorf("expected Title Project Hail Mary, got %s", res.Result.Title)
	}
	if res.Score < 90 {
		t.Errorf("expected high score for exact match, got %d", res.Score)
	}
}

func TestSearchBooks_ITunesFallback(t *testing.T) {
	// Audible server returns empty products
	audibleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.AudibleSearchResponse{Products: []models.AudibleProduct{}})
	}))
	defer audibleServer.Close()

	itunesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.ITunesSearchResponse{
			ResultCount: 1,
			Results: []models.ITunesResult{
				{
					TrackID:        999999,
					ArtistName:     "Andy Weir",
					TrackName:      "Project Hail Mary",
					ReleaseDate:    "2021-05-04T00:00:00Z",
					ArtworkUrl600:  "https://example.com/art.jpg",
					Description:    "<p>Andy Weir's masterpiece</p>",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer itunesServer.Close()

	cfg := &config.Config{
		Region:            "us",
		StoreAuthorAsMood: true,
	}

	audibleClient := api.NewAudibleClientWithBaseURL(5, audibleServer.URL)
	itunesClient := api.NewITunesClientWithBaseURL(5, itunesServer.URL)
	searchService := NewSearchService(nil, audibleClient, itunesClient, cfg)

	results, err := searchService.SearchBooks(context.Background(), "Project Hail Mary", "Andy Weir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 fallback result, got %d", len(results))
	}

	res := results[0]
	if res.Result.RatingKey != "itunes_999999" {
		t.Errorf("expected itunes_999999, got %s", res.Result.RatingKey)
	}
}

func TestSearchBooks_GoogleBooksFallback(t *testing.T) {
	// Audible and iTunes return empty
	audibleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.AudibleSearchResponse{Products: []models.AudibleProduct{}})
	}))
	defer audibleServer.Close()

	itunesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.ITunesSearchResponse{Results: []models.ITunesResult{}})
	}))
	defer itunesServer.Close()

	googleBooksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.GoogleBooksResponse{
			TotalItems: 1,
			Items: []models.GoogleBooksItem{
				{
					ID: "gb_item_1",
					VolumeInfo: models.GoogleVolumeInfo{
						Title:         "Dune",
						Authors:       []string{"Frank Herbert"},
						Publisher:     "Chilton Books",
						PublishedDate: "1965",
						Description:   "Classic sci-fi novel",
						Categories:    []string{"Science Fiction"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer googleBooksServer.Close()

	cfg := &config.Config{
		Region:            "us",
		StoreAuthorAsMood: true,
	}

	audibleClient := api.NewAudibleClientWithBaseURL(5, audibleServer.URL)
	itunesClient := api.NewITunesClientWithBaseURL(5, itunesServer.URL)
	gbClient := api.NewGoogleBooksClientWithBaseURL(5, googleBooksServer.URL)

	searchService := NewAudiobookshelfPipelineSearchService(nil, audibleClient, itunesClient, gbClient, nil, cfg)

	results, err := searchService.SearchBooks(context.Background(), "Dune", "Frank Herbert")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("expected Google Books result")
	}

	if results[0].Result.RatingKey != "googlebooks_gb_item_1" {
		t.Errorf("expected googlebooks_gb_item_1, got %s", results[0].Result.RatingKey)
	}
	if results[0].Result.Title != "Dune" {
		t.Errorf("expected Title Dune, got %s", results[0].Result.Title)
	}
}
