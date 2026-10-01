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

// AudiobookCoversClient queries the AudiobookCovers.com API (matching Audiobookshelf AudiobookCovers.js)
type AudiobookCoversClient struct {
	httpClient *http.Client
	timeout    time.Duration
	baseURL    string
}

// NewAudiobookCoversClient creates a new AudiobookCovers client
func NewAudiobookCoversClient(timeout int) *AudiobookCoversClient {
	return &AudiobookCoversClient{
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		timeout: time.Duration(timeout) * time.Second,
	}
}

// NewAudiobookCoversClientWithBaseURL creates a client with custom base URL
func NewAudiobookCoversClientWithBaseURL(timeout int, baseURL string) *AudiobookCoversClient {
	return &AudiobookCoversClient{
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		timeout: time.Duration(timeout) * time.Second,
		baseURL: baseURL,
	}
}

// Search searches for cover images by text query
func (c *AudiobookCoversClient) Search(ctx context.Context, query string) ([]string, error) {
	baseURL := "https://api.audiobookcovers.com/cover/bytext/"
	if c.baseURL != "" {
		baseURL = c.baseURL + "/cover/bytext/"
	}

	params := url.Values{
		"q": {query},
	}

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create audiobookcovers request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("audiobookcovers error: %d", resp.StatusCode)
	}

	var items []models.AudiobookCoverItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode audiobookcovers response: %w", err)
	}

	covers := make([]string, 0, len(items))
	for _, item := range items {
		img := item.Versions.PNG.Original
		if img == "" {
			img = item.Versions.PNG.Size1024
		}
		if img != "" {
			covers = append(covers, img)
		}
	}

	return covers, nil
}
