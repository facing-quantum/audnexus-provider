package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"audnexus-provider/internal/models"
)

// GoogleBooksClient queries the Google Books API
type GoogleBooksClient struct {
	httpClient *http.Client
	timeout    time.Duration
	baseURL    string
}

// NewGoogleBooksClient creates a new Google Books client
func NewGoogleBooksClient(timeout int) *GoogleBooksClient {
	return &GoogleBooksClient{
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		timeout: time.Duration(timeout) * time.Second,
	}
}

// NewGoogleBooksClientWithBaseURL creates a client with custom base URL
func NewGoogleBooksClientWithBaseURL(timeout int, baseURL string) *GoogleBooksClient {
	return &GoogleBooksClient{
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		timeout: time.Duration(timeout) * time.Second,
		baseURL: baseURL,
	}
}

// Search searches for volumes by title and author (matching Audiobookshelf GoogleBooks.js)
func (c *GoogleBooksClient) Search(ctx context.Context, title, author string) ([]models.GoogleBooksItem, error) {
	baseURL := "https://www.googleapis.com/books/v1/volumes"
	if c.baseURL != "" {
		baseURL = c.baseURL + "/books/v1/volumes"
	}

	q := fmt.Sprintf("intitle:%s", title)
	if author != "" {
		q += fmt.Sprintf("+inauthor:%s", author)
	}

	params := url.Values{
		"q":          {q},
		"maxResults": {"10"},
	}

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create google books request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google books error: %d", resp.StatusCode)
	}

	var gbResp models.GoogleBooksResponse
	if err := json.NewDecoder(resp.Body).Decode(&gbResp); err != nil {
		return nil, fmt.Errorf("failed to decode google books response: %w", err)
	}

	return gbResp.Items, nil
}
