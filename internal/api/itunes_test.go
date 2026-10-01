package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"audnexus-provider/internal/models"
)

func TestITunesSearchAudiobooks(t *testing.T) {
	expectedResponse := models.ITunesSearchResponse{
		ResultCount: 1,
		Results: []models.ITunesResult{
			{
				TrackID:          12345678,
				ArtistName:       "Andy Weir",
				CollectionName:   "Project Hail Mary",
				TrackName:        "Project Hail Mary (Unabridged)",
				ArtworkUrl100:    "https://example.com/art100.jpg",
				ArtworkUrl600:    "https://example.com/art600.jpg",
				ReleaseDate:      "2021-05-04T00:00:00Z",
				PrimaryGenreName: "Sci-Fi & Fantasy",
				Description:      "<p>A great audiobook</p>",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("media") != "audiobook" {
			t.Errorf("expected media=audiobook, got %s", r.URL.Query().Get("media"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewITunesClientWithBaseURL(10, server.URL)
	items, err := client.SearchAudiobooks(context.Background(), "Project Hail Mary", "us")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ArtistName != "Andy Weir" {
		t.Errorf("expected ArtistName Andy Weir, got %s", items[0].ArtistName)
	}
	if items[0].TrackID != 12345678 {
		t.Errorf("expected TrackID 12345678, got %d", items[0].TrackID)
	}
}
