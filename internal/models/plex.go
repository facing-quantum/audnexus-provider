package models

import (
	"encoding/json"
	"strings"
)

// ProviderIdentifier is the Plex specification identifier for this custom agent
const ProviderIdentifier = "tv.plex.agents.custom.audnexus"

// Scheme defines the GUID-scheme for items returned by this provider
type Scheme struct {
	Scheme string `json:"scheme"`
}

// TypeDefinition defines what metadata types are supported (8 for artist, 9 for album)
type TypeDefinition struct {
	Type   int      `json:"type"`
	Scheme []Scheme `json:"Scheme"`
}

// ProviderFeature defines an endpoint feature
type ProviderFeature struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

// MediaProvider represents Plex provider capabilities (matching Plex specification)
type MediaProvider struct {
	Identifier string            `json:"identifier"`
	Title      string            `json:"title"`
	Version    string            `json:"version,omitempty"`
	Types      []TypeDefinition  `json:"Types"`
	Feature    []ProviderFeature `json:"Feature"`
}

// MediaProviderResponse is the top-level response envelope for provider discovery
type MediaProviderResponse struct {
	MediaProvider MediaProvider `json:"MediaProvider"`
}

// TagItem represents a tag in Plex metadata (Genre, Mood, Style, Similar)
type TagItem struct {
	Tag string `json:"tag"`
}

// ImageItem represents an image asset in Plex metadata
type ImageItem struct {
	Type string `json:"type"` // "coverPoster", "background", "clearLogo", "snapshot"
	URL  string `json:"url"`
	Alt  string `json:"alt,omitempty"`
}

// GuidItem represents an external identifier mapping
type GuidItem struct {
	ID string `json:"id"` // e.g. "audible://B001", "itunes://123"
}

// PersonItem represents a person (Role, Narrator, Author)
type PersonItem struct {
	Tag   string `json:"tag"`
	Role  string `json:"role,omitempty"`
	Order int    `json:"order,omitempty"`
	Thumb string `json:"thumb,omitempty"`
}

// MediaContainer is the Plex response container
type MediaContainer struct {
	Offset     int         `json:"offset"`
	TotalSize  int         `json:"totalSize"`
	Identifier string      `json:"identifier,omitempty"`
	Size       int         `json:"size"`
	MediaType  string      `json:"mediaType,omitempty"`
	Metadata   []Metadata  `json:"Metadata,omitempty"`
	Image      []ImageItem `json:"Image,omitempty"`
}

// MediaContainerResponse is the top-level response envelope for Plex responses
type MediaContainerResponse struct {
	MediaContainer MediaContainer `json:"MediaContainer"`
}

type Metadata struct {
	RatingKey             string       `json:"ratingKey"`
	Key                   string       `json:"key"`
	GUID                  string       `json:"guid"`
	Type                  string       `json:"type"`
	Title                 string       `json:"title"`
	TitleSort             string       `json:"titleSort,omitempty"`
	Summary               string       `json:"summary,omitempty"`
	Year                  int          `json:"year,omitempty"`
	OriginallyAvailableAt string       `json:"originallyAvailableAt,omitempty"`
	Thumb                 string       `json:"thumb,omitempty"`
	Rating                float64      `json:"rating,omitempty"`
	Studio                string       `json:"studio,omitempty"`
	Genre                 []TagItem    `json:"Genre,omitempty"`
	Mood                  []TagItem    `json:"Mood,omitempty"`
	Style                 []TagItem    `json:"Style,omitempty"`
	Role                  []PersonItem `json:"Role,omitempty"`
	Similar               []TagItem    `json:"Similar,omitempty"`
	Image                 []ImageItem  `json:"Image,omitempty"`
	Guid                  []GuidItem   `json:"Guid,omitempty"`

	// Backward-compatible slice fields for internal logic & testing
	Genres   []string `json:"-"`
	Moods    []string `json:"-"`
	Styles   []string `json:"-"`
	Similars []string `json:"-"`
}

// ToTags converts string slice to TagItem slice
func ToTags(names []string) []TagItem {
	if len(names) == 0 {
		return nil
	}
	tags := make([]TagItem, 0, len(names))
	for _, n := range names {
		cleaned := strings.TrimSpace(n)
		if cleaned != "" {
			tags = append(tags, TagItem{Tag: cleaned})
		}
	}
	return tags
}

// FromTags converts TagItem slice to string slice
func FromTags(tags []TagItem) []string {
	if len(tags) == 0 {
		return nil
	}
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Tag
	}
	return names
}

// SyncTags ensures bidirectional sync between string slices and Plex TagItem arrays
func (m *Metadata) SyncTags() {
	if len(m.Genres) > 0 && len(m.Genre) == 0 {
		m.Genre = ToTags(m.Genres)
	} else if len(m.Genre) > 0 && len(m.Genres) == 0 {
		m.Genres = FromTags(m.Genre)
	}

	if len(m.Moods) > 0 && len(m.Mood) == 0 {
		m.Mood = ToTags(m.Moods)
	} else if len(m.Mood) > 0 && len(m.Moods) == 0 {
		m.Moods = FromTags(m.Mood)
	}

	if len(m.Styles) > 0 && len(m.Style) == 0 {
		m.Style = ToTags(m.Styles)
	} else if len(m.Style) > 0 && len(m.Styles) == 0 {
		m.Styles = FromTags(m.Style)
	}

	if len(m.Similars) > 0 && len(m.Similar) == 0 {
		m.Similar = ToTags(m.Similars)
	} else if len(m.Similar) > 0 && len(m.Similars) == 0 {
		m.Similars = FromTags(m.Similar)
	}

	// Auto-populate Key if missing
	if m.Key == "" && m.RatingKey != "" {
		m.Key = "/audnexus/library/metadata/" + m.RatingKey
	}

	// Auto-populate Image if Thumb is provided
	if m.Thumb != "" && len(m.Image) == 0 {
		m.Image = []ImageItem{
			{Type: "coverPoster", URL: m.Thumb, Alt: m.Title},
		}
	}

	// Auto-ensure Plex-compliant GUID
	if m.RatingKey != "" && !strings.HasPrefix(m.GUID, "tv.plex.agents.custom.") {
		originalGuid := m.GUID
		m.GUID = ProviderIdentifier + "://" + m.Type + "/" + m.RatingKey
		if originalGuid != "" && originalGuid != m.GUID {
			// Record original GUID in external Guid list if not present
			found := false
			for _, g := range m.Guid {
				if g.ID == originalGuid {
					found = true
					break
				}
			}
			if !found {
				m.Guid = append(m.Guid, GuidItem{ID: originalGuid})
			}
		}
	}
}

// MarshalJSON populates Plex spec fields prior to JSON serialization
func (m *Metadata) MarshalJSON() ([]byte, error) {
	m.SyncTags()
	type Alias Metadata
	return json.Marshal((*Alias)(m))
}

// UnmarshalJSON parses JSON and synchronizes fields
func (m *Metadata) UnmarshalJSON(data []byte) error {
	type Alias Metadata
	if err := json.Unmarshal(data, (*Alias)(m)); err != nil {
		return err
	}
	m.SyncTags()
	return nil
}
