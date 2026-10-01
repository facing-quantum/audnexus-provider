package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"audnexus-provider/internal/models"
)

func TestAudibleSearchProducts(t *testing.T) {
	expectedProducts := models.AudibleSearchResponse{
		Products: []models.AudibleProduct{
			{
				ASIN:        "B08G9PRS1K",
				Title:       "Project Hail Mary",
				Authors:     []models.Contributor{{Name: "Andy Weir"}},
				Narrators:   []models.Contributor{{Name: "Ray Porter"}},
				ReleaseDate: "2021-05-04",
				Language:    "english",
				Publisher:   "Audible Studios",
			},
		},
		TotalResults: 1,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1.0/catalog/products" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("title") != "Project Hail Mary" {
			t.Errorf("unexpected title param: %s", r.URL.Query().Get("title"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedProducts)
	}))
	defer server.Close()

	client := NewAudibleClientWithBaseURL(10, server.URL)
	products, err := client.SearchProducts(context.Background(), "Project Hail Mary", "Andy Weir", "us")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}
	if products[0].ASIN != "B08G9PRS1K" {
		t.Errorf("expected ASIN B08G9PRS1K, got %s", products[0].ASIN)
	}
	if len(products[0].Authors) != 1 || products[0].Authors[0].Name != "Andy Weir" {
		t.Errorf("unexpected author: %+v", products[0].Authors)
	}
}
