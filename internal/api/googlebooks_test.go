package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"audnexus-provider/internal/models"
)

func TestGoogleBooksSearch(t *testing.T) {
	expectedResponse := models.GoogleBooksResponse{
		TotalItems: 1,
		Items: []models.GoogleBooksItem{
			{
				ID: "vol123",
				VolumeInfo: models.GoogleVolumeInfo{
					Title:         "Project Hail Mary",
					Authors:       []string{"Andy Weir"},
					Publisher:     "Ballantine Books",
					PublishedDate: "2021-05-04",
					Description:   "A lone astronaut on a mission.",
					Categories:    []string{"Science Fiction"},
					IndustryIdentifiers: []models.IndustryIdentifier{
						{Type: "ISBN_13", Identifier: "9781603935470"},
					},
					ImageLinks: map[string]string{
						"thumbnail": "https://books.google.com/cover.jpg",
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/books/v1/volumes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query().Get("q")
		if q == "" {
			t.Errorf("expected q query param")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewGoogleBooksClientWithBaseURL(10, server.URL)
	items, err := client.Search(context.Background(), "Project Hail Mary", "Andy Weir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].VolumeInfo.Title != "Project Hail Mary" {
		t.Errorf("expected title Project Hail Mary, got %s", items[0].VolumeInfo.Title)
	}
}
