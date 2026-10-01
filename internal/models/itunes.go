package models

// ITunesSearchResponse represents the response from iTunes Search API
type ITunesSearchResponse struct {
	ResultCount int            `json:"resultCount"`
	Results     []ITunesResult `json:"results"`
}

// ITunesResult represents an audiobook result from iTunes Search API
type ITunesResult struct {
	TrackID          int64   `json:"trackId"`
	ArtistName       string  `json:"artistName"`
	CollectionName   string  `json:"collectionName"`
	TrackName        string  `json:"trackName"`
	ArtworkUrl100    string  `json:"artworkUrl100"`
	ArtworkUrl600    string  `json:"artworkUrl600"`
	ReleaseDate      string  `json:"releaseDate"`
	PrimaryGenreName string  `json:"primaryGenreName"`
	Description      string  `json:"description"`
}
