package models

// AudibleSearchResponse represents the response from Audible catalog API
type AudibleSearchResponse struct {
	Products     []AudibleProduct `json:"products"`
	TotalResults int              `json:"total_results"`
}

// AudibleProduct represents an item returned by Audible catalog API
type AudibleProduct struct {
	ASIN        string        `json:"asin"`
	Title       string        `json:"title"`
	Authors     []Contributor `json:"authors"`
	Narrators   []Contributor `json:"narrators"`
	ReleaseDate string        `json:"release_date"`
	Language    string        `json:"language"`
	Publisher   string        `json:"publisher_name"`
	Summary     string        `json:"merchandising_summary"`
}
