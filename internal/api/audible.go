package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"audnexus-provider/internal/models"
)

var audibleTLDs = map[string]string{
	"au": "com.au",
	"ca": "ca",
	"de": "de",
	"es": "es",
	"fr": "fr",
	"in": "in",
	"it": "it",
	"jp": "co.jp",
	"us": "com",
	"uk": "co.uk",
}

// AudibleClient queries Audible's catalog API for products
type AudibleClient struct {
	httpClient *http.Client
	timeout    time.Duration
	baseURL    string // override for tests
}

// NewAudibleClient creates a new Audible API client
func NewAudibleClient(timeout int) *AudibleClient {
	return &AudibleClient{
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		timeout: time.Duration(timeout) * time.Second,
	}
}

// NewAudibleClientWithBaseURL creates a new Audible API client with custom base URL (for testing)
func NewAudibleClientWithBaseURL(timeout int, baseURL string) *AudibleClient {
	return &AudibleClient{
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		timeout: time.Duration(timeout) * time.Second,
		baseURL: baseURL,
	}
}

func (c *AudibleClient) getBaseURL(region string) string {
	if c.baseURL != "" {
		return c.baseURL
	}
	tld, ok := audibleTLDs[region]
	if !ok {
		tld = "com"
	}
	return fmt.Sprintf("https://api.audible.%s", tld)
}

// SearchProducts queries Audible catalog search for products
func (c *AudibleClient) SearchProducts(ctx context.Context, title, author, region string) ([]models.AudibleProduct, error) {
	baseURL := c.getBaseURL(region)
	endpoint := fmt.Sprintf("%s/1.0/catalog/products", baseURL)

	params := url.Values{
		"response_groups":  {"contributors,product_desc,product_attrs"},
		"num_results":      {"25"},
		"products_sort_by": {"Relevance"},
	}

	if author != "" && title != "" {
		params.Set("title", title)
		params.Set("author", author)
	} else if title != "" {
		params.Set("keywords", title)
	} else if author != "" {
		params.Set("keywords", author)
	}

	reqURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create audible request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("audible api error: %d - %s", resp.StatusCode, string(body))
	}

	var searchResp models.AudibleSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode audible response: %w", err)
	}

	return searchResp.Products, nil
}
