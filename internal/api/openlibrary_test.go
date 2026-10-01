package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"audnexus-provider/internal/models"
)

func TestOpenLibrarySearchBooks(t *testing.T) {
	expectedResponse := models.OpenLibrarySearchResponse{
		NumFound: 1,
		Docs: []models.OpenLibraryDoc{
			{
				Key:              "/works/OL12345W",
				Title:            "Project Hail Mary",
				AuthorName:       []string{"Andy Weir"},
				FirstPublishYear: 2021,
				CoverI:           987654,
				Publisher:        []string{"Ballantine Books"},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search.json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("title") != "Project Hail Mary" {
			t.Errorf("unexpected title param: %s", r.URL.Query().Get("title"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewOpenLibraryClientWithBaseURL(10, server.URL)
	docs, err := client.SearchBooks(context.Background(), "Project Hail Mary", "Andy Weir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].Key != "/works/OL12345W" {
		t.Errorf("expected Key /works/OL12345W, got %s", docs[0].Key)
	}
	if docs[0].Title != "Project Hail Mary" {
		t.Errorf("expected Title Project Hail Mary, got %s", docs[0].Title)
	}
}
