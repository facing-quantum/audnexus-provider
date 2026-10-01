package models

// AudiobookCoverItem represents a cover item returned by AudiobookCovers.com
type AudiobookCoverItem struct {
	Versions AudiobookCoverVersions `json:"versions"`
}

type AudiobookCoverVersions struct {
	PNG AudiobookCoverSizes `json:"png"`
}

type AudiobookCoverSizes struct {
	Original string `json:"original"`
	Size1024 string `json:"1024"`
	Size512  string `json:"512"`
}
