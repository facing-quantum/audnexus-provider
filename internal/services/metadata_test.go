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

func TestGetBookMetadata(t *testing.T) {
	audnexusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		book := models.Book{
			ASIN:          "B08G9PRS1K",
			Title:         "Project Hail Mary",
			Summary:       "<p><b>Winner</b> of the Audie Award.</p>",
			Rating:        "4.9",
			ReleaseDate:   "2021-05-04",
			PublisherName: "Audible Studios",
			Authors:       []models.Contributor{{Name: "Andy Weir"}},
			Narrators:     []models.Contributor{{Name: "Ray Porter"}},
			SeriesPrimary: &models.Series{Name: "Space Adventures", Position: "1"},
			Genres:        []models.Genre{{Name: "Sci-Fi"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(book)
	}))
	defer audnexusServer.Close()

	cfg := &config.Config{
		Region:            "us",
		StoreAuthorAsMood: true,
	}

	audnexusClient := api.NewClientWithBaseURL(5, audnexusServer.URL)
	metadataService := NewMetadataService(audnexusClient, nil, nil, cfg)

	meta, err := metadataService.GetBookMetadata(context.Background(), "B08G9PRS1K")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.Title != "Project Hail Mary" {
		t.Errorf("expected Title Project Hail Mary, got %s", meta.Title)
	}
	if meta.TitleSort != "Space Adventures, Book 1 - Project Hail Mary" {
		t.Errorf("expected TitleSort with series, got %s", meta.TitleSort)
	}
	if meta.Rating != 9.8 {
		t.Errorf("expected scaled rating 9.8, got %v", meta.Rating)
	}
	if meta.Summary != "Winner of the Audie Award." {
		t.Errorf("expected cleaned HTML summary, got %q", meta.Summary)
	}
	if len(meta.Styles) != 1 || meta.Styles[0] != "Ray Porter" {
		t.Errorf("expected narrator Ray Porter in Styles, got %+v", meta.Styles)
	}
}

func TestGetAuthorMetadata(t *testing.T) {
	audnexusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		author := models.Author{
			ASIN:        "B00G0WYW92",
			Name:        "Andy Weir",
			Description: "<p>American novelist.</p>",
			Genres:      []models.Genre{{Name: "Sci-Fi"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(author)
	}))
	defer audnexusServer.Close()

	cfg := &config.Config{
		Region:               "us",
		SortAuthorByLastName: true,
	}

	audnexusClient := api.NewClientWithBaseURL(5, audnexusServer.URL)
	metadataService := NewMetadataService(audnexusClient, nil, nil, cfg)

	meta, err := metadataService.GetAuthorMetadata(context.Background(), "B00G0WYW92")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.Title != "Andy Weir" {
		t.Errorf("expected Title Andy Weir, got %s", meta.Title)
	}
	if meta.TitleSort != "Weir, Andy" {
		t.Errorf("expected TitleSort Weir, Andy, got %s", meta.TitleSort)
	}
	if meta.Summary != "American novelist." {
		t.Errorf("expected cleaned summary, got %q", meta.Summary)
	}
}

func TestGetBookMetadata_CoverEnrichmentAndSeriesCleaning(t *testing.T) {
	// Book missing image and with dirty series sequence "Book 2, Dramatized"
	audnexusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		book := models.Book{
			ASIN:          "B001",
			Title:         "Way of Kings",
			Image:         "", // Missing cover!
			SeriesPrimary: &models.Series{Name: "Stormlight", Position: "Book 2, Dramatized"},
			Genres: []models.Genre{
				{Name: "Epic Fantasy", Type: "genre"},
				{Name: "Magic", Type: "tag"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(book)
	}))
	defer audnexusServer.Close()

	// AudiobookCovers returns high-res cover
	coversServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := []models.AudiobookCoverItem{
			{
				Versions: models.AudiobookCoverVersions{
					PNG: models.AudiobookCoverSizes{
						Original: "https://api.audiobookcovers.com/cover/hi-res.png",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer coversServer.Close()

	cfg := &config.Config{Region: "us"}
	audnexusClient := api.NewClientWithBaseURL(5, audnexusServer.URL)
	coversClient := api.NewAudiobookCoversClientWithBaseURL(5, coversServer.URL)

	metadataService := NewEnrichedMetadataService(audnexusClient, nil, nil, coversClient, nil, cfg)

	meta, err := metadataService.GetBookMetadata(context.Background(), "B001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1. Cover should be enriched from AudiobookCovers
	if meta.Thumb != "https://api.audiobookcovers.com/cover/hi-res.png" {
		t.Errorf("expected enriched cover, got %s", meta.Thumb)
	}

	// 2. Series sequence should be cleaned to "2"
	if meta.TitleSort != "Stormlight, Book 2 - Way of Kings" {
		t.Errorf("expected clean series sequence 'Stormlight, Book 2 - Way of Kings', got %s", meta.TitleSort)
	}

	// 3. Tags vs Genres separation
	if len(meta.Genres) != 1 || meta.Genres[0] != "Epic Fantasy" {
		t.Errorf("expected genre 'Epic Fantasy', got %+v", meta.Genres)
	}
}
