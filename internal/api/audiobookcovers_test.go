package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"audnexus-provider/internal/models"
)

func TestAudiobookCoversSearch(t *testing.T) {
	expectedResponse := []models.AudiobookCoverItem{
		{
			Versions: models.AudiobookCoverVersions{
				PNG: models.AudiobookCoverSizes{
					Original: "https://api.audiobookcovers.com/cover/original.png",
					Size1024: "https://api.audiobookcovers.com/cover/1024.png",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cover/bytext/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query().Get("q")
		if q != "Project Hail Mary" {
			t.Errorf("expected q=Project Hail Mary, got %s", q)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewAudiobookCoversClientWithBaseURL(10, server.URL)
	covers, err := client.Search(context.Background(), "Project Hail Mary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(covers) != 1 {
		t.Fatalf("expected 1 cover, got %d", len(covers))
	}
	if covers[0] != "https://api.audiobookcovers.com/cover/original.png" {
		t.Errorf("expected original url, got %s", covers[0])
	}
}
