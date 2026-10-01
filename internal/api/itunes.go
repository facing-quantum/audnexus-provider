package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"audnexus-provider/internal/models"
)

// ITunesClient queries the iTunes Search API for audiobooks
type ITunesClient struct {
	httpClient *http.Client
	timeout    time.Duration
	baseURL    string // override for tests
}

// NewITunesClient creates a new iTunes Search API client
func NewITunesClient(timeout int) *ITunesClient {
	return &ITunesClient{
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		timeout: time.Duration(timeout) * time.Second,
	}
}

// NewITunesClientWithBaseURL creates a new iTunes API client with custom base URL
func NewITunesClientWithBaseURL(timeout int, baseURL string) *ITunesClient {
	return &ITunesClient{
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		timeout: time.Duration(timeout) * time.Second,
		baseURL: baseURL,
	}
}

// SearchAudiobooks searches iTunes for audiobooks matching term and country
func (c *ITunesClient) SearchAudiobooks(ctx context.Context, term, country string) ([]models.ITunesResult, error) {
	baseURL := "https://itunes.apple.com"
	if c.baseURL != "" {
		baseURL = c.baseURL
	}

	if country == "" {
		country = "us"
	}
	if strings.ToLower(country) == "uk" {
		country = "gb"
	}

	params := url.Values{
		"media":   {"audiobook"},
		"entity":  {"audiobook"},
		"limit":   {"25"},
		"term":    {term},
		"country": {strings.ToLower(country)},
	}

	reqURL := fmt.Sprintf("%s/search?%s", baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create itunes request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("itunes api error: %d", resp.StatusCode)
	}

	var itunesResp models.ITunesSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&itunesResp); err != nil {
		return nil, fmt.Errorf("failed to decode itunes response: %w", err)
	}

	return itunesResp.Results, nil
}
