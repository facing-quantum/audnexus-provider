package models

// GoogleBooksResponse represents response from Google Books API
type GoogleBooksResponse struct {
	TotalItems int               `json:"totalItems"`
	Items      []GoogleBooksItem `json:"items"`
}

// GoogleBooksItem represents a volume item from Google Books API
type GoogleBooksItem struct {
	ID         string            `json:"id"`
	VolumeInfo GoogleVolumeInfo  `json:"volumeInfo"`
}

// GoogleVolumeInfo represents the volume information
type GoogleVolumeInfo struct {
	Title               string               `json:"title"`
	Subtitle            string               `json:"subtitle,omitempty"`
	Authors             []string             `json:"authors,omitempty"`
	Publisher           string               `json:"publisher,omitempty"`
	PublishedDate       string               `json:"publishedDate,omitempty"`
	Description         string               `json:"description,omitempty"`
	IndustryIdentifiers []IndustryIdentifier `json:"industryIdentifiers,omitempty"`
	Categories          []string             `json:"categories,omitempty"`
	ImageLinks          map[string]string    `json:"imageLinks,omitempty"`
}

// IndustryIdentifier represents ISBN or other book identifiers
type IndustryIdentifier struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}
