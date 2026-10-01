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

// OpenLibraryClient queries the Open Library Search API
type OpenLibraryClient struct {
	httpClient *http.Client
	timeout    time.Duration
	baseURL    string
}

// NewOpenLibraryClient creates a new Open Library client
func NewOpenLibraryClient(timeout int) *OpenLibraryClient {
	return &OpenLibraryClient{
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		timeout: time.Duration(timeout) * time.Second,
	}
}

// NewOpenLibraryClientWithBaseURL creates a new Open Library client with custom baseURL
func NewOpenLibraryClientWithBaseURL(timeout int, baseURL string) *OpenLibraryClient {
	return &OpenLibraryClient{
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		timeout: time.Duration(timeout) * time.Second,
		baseURL: baseURL,
	}
}

// SearchBooks searches Open Library by title and author
func (c *OpenLibraryClient) SearchBooks(ctx context.Context, title, author string) ([]models.OpenLibraryDoc, error) {
	baseURL := "https://openlibrary.org"
	if c.baseURL != "" {
		baseURL = c.baseURL
	}

	params := url.Values{
		"limit": {"25"},
	}
	if title != "" {
		params.Set("title", title)
	}
	if author != "" {
		params.Set("author", author)
	}

	reqURL := fmt.Sprintf("%s/search.json?%s", baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create open library request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open library search error: %d", resp.StatusCode)
	}

	var olResp models.OpenLibrarySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&olResp); err != nil {
		return nil, fmt.Errorf("failed to decode open library response: %w", err)
	}

	return olResp.Docs, nil
}
