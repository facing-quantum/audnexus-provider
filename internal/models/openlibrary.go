package models

// OpenLibrarySearchResponse represents response from Open Library Search API
type OpenLibrarySearchResponse struct {
	NumFound int              `json:"numFound"`
	Docs     []OpenLibraryDoc `json:"docs"`
}

// OpenLibraryDoc represents a book result from Open Library
type OpenLibraryDoc struct {
	Key              string   `json:"key"`
	Title            string   `json:"title"`
	AuthorName       []string `json:"author_name"`
	FirstPublishYear int      `json:"first_publish_year"`
	CoverI           int64    `json:"cover_i"`
	Publisher        []string `json:"publisher"`
}
