package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"audnexus-provider/internal/models"
)

func TestNewClient(t *testing.T) {
	client := NewClient(30)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.timeout != 30*time.Second {
		t.Errorf("expected timeout of 30s, got %v", client.timeout)
	}
}

func TestSearchAuthors(t *testing.T) {
	expectedAuthors := []models.Author{
		{
			ASIN:        "B0000001",
			Name:        "Test Author",
			Description: "A test author",
			Image:       "https://example.com/image.jpg",
			Genres:      []models.Genre{{Name: "Fiction"}},
			Similar:     []models.Contributor{{Name: "Other Author", ASIN: "B0000002"}},
			Region:      "us",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/authors" {
			t.Errorf("expected path /authors, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("name") != "Test Author" {
			t.Errorf("expected name=Test Author, got %s", r.URL.Query().Get("name"))
		}
		if r.URL.Query().Get("region") != "us" {
			t.Errorf("expected region=us, got %s", r.URL.Query().Get("region"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedAuthors)
	}))
	defer server.Close()

	client := NewClientWithBaseURL(30, server.URL)

	ctx := context.Background()
	params := url.Values{
		"name":   {"Test Author"},
		"region": {"us"},
	}

	body, err := client.doRequest(ctx, "GET", "/authors", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var authors []models.Author
	if err := json.Unmarshal(body, &authors); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(authors) != 1 {
		t.Errorf("expected 1 author, got %d", len(authors))
	}
	if authors[0].ASIN != "B0000001" {
		t.Errorf("expected ASIN B0000001, got %s", authors[0].ASIN)
	}
}

func TestGetAuthorByASIN(t *testing.T) {
	expectedAuthor := models.Author{
		ASIN:        "B0000001",
		Name:        "Test Author",
		Description: "A test author",
		Image:       "https://example.com/image.jpg",
		Genres:      []models.Genre{{Name: "Fiction"}},
		Similar:     []models.Contributor{{Name: "Other Author", ASIN: "B0000002"}},
		Region:      "us",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/authors/B0000001" {
			t.Errorf("expected path /authors/B0000001, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedAuthor)
	}))
	defer server.Close()

	client := NewClientWithBaseURL(30, server.URL)
	ctx := context.Background()
	params := url.Values{"region": {"us"}}

	body, err := client.doRequest(ctx, "GET", "/authors/B0000001", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var author models.Author
	if err := json.Unmarshal(body, &author); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if author.ASIN != "B0000001" {
		t.Errorf("expected ASIN B0000001, got %s", author.ASIN)
	}
}

func TestGetBookByASIN(t *testing.T) {
	expectedBook := models.Book{
		ASIN:          "B0000001",
		Title:         "Test Book",
		Subtitle:      "A Subtitle",
		Description:   "A test book",
		Summary:       "<p>A test book summary</p>",
		Image:         "https://example.com/cover.jpg",
		Rating:        "4.8",
		RatingCount:   100,
		ReleaseDate:   "2024-01-01",
		Region:        "us",
		PublisherName: "Test Publisher",
		Authors:       []models.Contributor{{Name: "Test Author", ASIN: "A1"}},
		Narrators:     []models.Contributor{{Name: "Test Narrator"}},
		SeriesPrimary: &models.Series{ASIN: "S0001", Name: "Test Series", Position: "1"},
		Genres:        []models.Genre{{Name: "Fiction"}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/books/B0000001" {
			t.Errorf("expected path /books/B0000001, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedBook)
	}))
	defer server.Close()

	client := NewClientWithBaseURL(30, server.URL)
	ctx := context.Background()
	params := url.Values{"region": {"us"}}

	body, err := client.doRequest(ctx, "GET", "/books/B0000001", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var book models.Book
	if err := json.Unmarshal(body, &book); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if book.ASIN != "B0000001" {
		t.Errorf("expected ASIN B0000001, got %s", book.ASIN)
	}
	if len(book.Authors) != 1 || book.Authors[0].Name != "Test Author" {
		t.Errorf("expected author Test Author, got %+v", book.Authors)
	}
}

func TestClientErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request"}`))
	}))
	defer server.Close()

	client := NewClientWithBaseURL(30, server.URL)
	ctx := context.Background()
	params := url.Values{"name": {"test"}}

	_, err := client.doRequest(ctx, "GET", "/authors", params)
	if err == nil {
		t.Error("expected error for 400 status code")
	}
}

func TestServerErrorRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClientWithBaseURL(30, server.URL)
	ctx := context.Background()
	params := url.Values{"name": {"test"}}

	_, err := client.doRequest(ctx, "GET", "/authors", params)
	if err == nil {
		t.Error("expected error after retries")
	}
	if attempts != maxRetries {
		t.Errorf("expected %d attempts, got %d", maxRetries, attempts)
	}
}
