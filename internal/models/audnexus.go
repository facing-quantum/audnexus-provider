package models

// Contributor represents an author or narrator
type Contributor struct {
	ASIN string `json:"asin,omitempty"`
	Name string `json:"name"`
}

// Genre represents a genre or tag
type Genre struct {
	ASIN string `json:"asin,omitempty"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// Series represents a series from Audnexus API
type Series struct {
	ASIN     string `json:"asin,omitempty"`
	Name     string `json:"name"`
	Position string `json:"position,omitempty"`
}

// Author represents an author from Audnexus API
type Author struct {
	ASIN        string        `json:"asin"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Image       string        `json:"image,omitempty"`
	Genres      []Genre       `json:"genres,omitempty"`
	Similar     []Contributor `json:"similar,omitempty"`
	Region      string        `json:"region,omitempty"`
}

// Book represents a book from Audnexus API
type Book struct {
	ASIN             string        `json:"asin"`
	Title            string        `json:"title"`
	Subtitle         string        `json:"subtitle,omitempty"`
	Description      string        `json:"description,omitempty"`
	Summary          string        `json:"summary,omitempty"`
	Image            string        `json:"image,omitempty"`
	Rating           string        `json:"rating,omitempty"`
	RatingCount      int           `json:"ratingCount,omitempty"`
	ReleaseDate      string        `json:"releaseDate,omitempty"`
	Region           string        `json:"region,omitempty"`
	PublisherName    string        `json:"publisherName,omitempty"`
	Authors          []Contributor `json:"authors,omitempty"`
	Narrators        []Contributor `json:"narrators,omitempty"`
	SeriesPrimary    *Series       `json:"seriesPrimary,omitempty"`
	SeriesSecondary  *Series       `json:"seriesSecondary,omitempty"`
	Genres           []Genre       `json:"genres,omitempty"`
	Language         string        `json:"language,omitempty"`
	RuntimeLengthMin int           `json:"runtimeLengthMin,omitempty"`
}
