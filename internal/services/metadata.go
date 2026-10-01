package services

import (
	"context"
	"fmt"
	"strings"

	"audnexus-provider/internal/api"
	"audnexus-provider/internal/cache"
	"audnexus-provider/internal/config"
	"audnexus-provider/internal/models"
	"audnexus-provider/internal/utils"
	"golang.org/x/sync/singleflight"
)

// MetadataService handles fetching full metadata for authors and books with caching and coalescing
type MetadataService struct {
	client                *api.Client
	audibleClient         *api.AudibleClient
	itunesClient          *api.ITunesClient
	audiobookCoversClient *api.AudiobookCoversClient
	cache                 *cache.PersistentCache
	cfg                   *config.Config
	sfGroup               singleflight.Group
}

// NewMetadataService creates a new MetadataService
func NewMetadataService(
	client *api.Client,
	audibleClient *api.AudibleClient,
	itunesClient *api.ITunesClient,
	cfg *config.Config,
) *MetadataService {
	return NewEnrichedMetadataService(client, audibleClient, itunesClient, nil, nil, cfg)
}

// NewResilientMetadataService creates a MetadataService with persistent caching and singleflight
func NewResilientMetadataService(
	client *api.Client,
	audibleClient *api.AudibleClient,
	itunesClient *api.ITunesClient,
	persistentCache *cache.PersistentCache,
	cfg *config.Config,
) *MetadataService {
	return NewEnrichedMetadataService(client, audibleClient, itunesClient, nil, persistentCache, cfg)
}

// NewEnrichedMetadataService creates a MetadataService with cover enrichment, persistent caching, and singleflight
func NewEnrichedMetadataService(
	client *api.Client,
	audibleClient *api.AudibleClient,
	itunesClient *api.ITunesClient,
	audiobookCoversClient *api.AudiobookCoversClient,
	persistentCache *cache.PersistentCache,
	cfg *config.Config,
) *MetadataService {
	return &MetadataService{
		client:                client,
		audibleClient:         audibleClient,
		itunesClient:          itunesClient,
		audiobookCoversClient: audiobookCoversClient,
		cache:                 persistentCache,
		cfg:                   cfg,
	}
}

// GetAuthorMetadata fetches full metadata for an author by ASIN
func (s *MetadataService) GetAuthorMetadata(ctx context.Context, asin string) (*models.Metadata, error) {
	key := "author:" + asin

	res, err, _ := s.sfGroup.Do(key, func() (interface{}, error) {
		fetchFn := func() (interface{}, error) {
			if s.client == nil {
				return nil, fmt.Errorf("audnexus client not configured")
			}
			author, err := s.client.GetAuthorByASIN(ctx, asin, s.cfg.Region)
			if err != nil {
				return nil, err
			}

			var genreNames []string
			for _, g := range author.Genres {
				if g.Name != "" {
					genreNames = append(genreNames, g.Name)
				}
			}

			var similarNames []string
			for _, sim := range author.Similar {
				if sim.Name != "" {
					similarNames = append(similarNames, sim.Name)
				}
			}

			titleSort := author.Name
			if s.cfg.SortAuthorByLastName {
				titleSort = utils.SortName(author.Name)
			}

			return &models.Metadata{
				RatingKey: "author_" + author.ASIN,
				GUID:      "audnexus://author/" + author.ASIN,
				Type:      "artist",
				Title:     author.Name,
				TitleSort: titleSort,
				Summary:   utils.CleanHTML(author.Description),
				Thumb:     author.Image,
				Genres:    genreNames,
				Similars:  similarNames,
			}, nil
		}

		if s.cache != nil {
			var metadata models.Metadata
			err := s.cache.GetOrFetch(key, &metadata, fetchFn)
			if err != nil {
				return nil, err
			}
			return &metadata, nil
		}

		val, err := fetchFn()
		if err != nil {
			return nil, err
		}
		return val.(*models.Metadata), nil
	})

	if err != nil {
		return nil, err
	}
	return res.(*models.Metadata), nil
}

// GetBookMetadata fetches full metadata for a book by ASIN or local ID
func (s *MetadataService) GetBookMetadata(ctx context.Context, id string) (*models.Metadata, error) {
	// Handle synthetic local items
	if strings.HasPrefix(id, "local_") {
		return &models.Metadata{
			RatingKey: id,
			GUID:      fmt.Sprintf("local://audiobook/%s", strings.TrimPrefix(id, "local_")),
			Type:      "album",
			Title:     "Local Audiobook",
			Summary:   "Metadata generated from local files.",
		}, nil
	}

	// Handle iTunes fallback items
	if strings.HasPrefix(id, "itunes_") {
		return s.getITunesMetadata(ctx, id)
	}

	key := "book:" + id

	res, err, _ := s.sfGroup.Do(key, func() (interface{}, error) {
		fetchFn := func() (interface{}, error) {
			asin := id
			if s.client == nil {
				return nil, fmt.Errorf("audnexus client not configured")
			}
			book, err := s.client.GetBookByASIN(ctx, asin, s.cfg.Region)
			if err != nil {
				return nil, err
			}

			// 1. Title formatting
			var displayTitle string
			if s.cfg.SimplifyTitle {
				displayTitle = utils.SimplifyTitle(book.Title)
			} else if book.Subtitle != "" {
				displayTitle = book.Title + ": " + book.Subtitle
			} else {
				displayTitle = book.Title
			}

			// 2. Sort title formatting (Series, Book X - Title)
			titleSort := displayTitle
			if book.SeriesPrimary != nil && book.SeriesPrimary.Name != "" {
				vol := utils.CleanSeriesSequence(book.SeriesPrimary.Position)
				if vol != "" && !strings.HasPrefix(strings.ToLower(vol), "book") {
					vol = "Book " + vol
				}
				if vol != "" {
					titleSort = fmt.Sprintf("%s, %s - %s", book.SeriesPrimary.Name, vol, displayTitle)
				} else {
					titleSort = fmt.Sprintf("%s - %s", book.SeriesPrimary.Name, displayTitle)
				}
			}

			// 3. Authors and Narrators
			var authorNames []string
			for _, a := range book.Authors {
				if a.Name != "" {
					authorNames = append(authorNames, a.Name)
				}
			}

			var narratorNames []string
			for _, n := range book.Narrators {
				if n.Name != "" {
					narratorNames = append(narratorNames, n.Name)
				}
			}

			// 4. Genres and Tags (Audiobookshelf separates genres and tags)
			var genreNames []string
			var tagNames []string
			if !s.cfg.KeepExistingGenres {
				for _, g := range book.Genres {
					if g.Name != "" {
						if strings.EqualFold(g.Type, "tag") {
							tagNames = append(tagNames, g.Name)
						} else {
							genreNames = append(genreNames, g.Name)
						}
					}
				}
			}

			// 5. Moods (Authors, Series, and Tags)
			var moods []string
			if s.cfg.StoreAuthorAsMood {
				moods = append(moods, authorNames...)
			}
			if book.SeriesPrimary != nil && book.SeriesPrimary.Name != "" {
				moods = append(moods, "Series: "+book.SeriesPrimary.Name)
			}
			if book.SeriesSecondary != nil && book.SeriesSecondary.Name != "" {
				moods = append(moods, "Series: "+book.SeriesSecondary.Name)
			}
			moods = append(moods, tagNames...)

			// 6. Summary cleanup
			summary := book.Summary
			if summary == "" {
				summary = book.Description
			}

			// 7. Cover art enrichment (Audiobookshelf AudiobookCovers.com provider)
			thumb := book.Image
			if thumb == "" && s.audiobookCoversClient != nil {
				covers, err := s.audiobookCoversClient.Search(ctx, displayTitle)
				if err == nil && len(covers) > 0 {
					thumb = covers[0]
				}
			}

			return &models.Metadata{
				RatingKey:             "album_" + book.ASIN,
				GUID:                  "audnexus://album/" + book.ASIN,
				Type:                  "album",
				Title:                 displayTitle,
				TitleSort:             titleSort,
				Summary:               utils.CleanHTML(summary),
				Year:                  parseYear(book.ReleaseDate),
				OriginallyAvailableAt: book.ReleaseDate,
				Thumb:                 thumb,
				Rating:                utils.ParseRating(book.Rating),
				Studio:                book.PublisherName,
				Genres:                genreNames,
				Moods:                 moods,
				Styles:                narratorNames,
			}, nil
		}

		if s.cache != nil {
			var metadata models.Metadata
			err := s.cache.GetOrFetch(key, &metadata, fetchFn)
			if err != nil {
				return nil, err
			}
			return &metadata, nil
		}

		val, err := fetchFn()
		if err != nil {
			return nil, err
		}
		return val.(*models.Metadata), nil
	})

	if err != nil {
		return nil, err
	}
	return res.(*models.Metadata), nil
}

func (s *MetadataService) getITunesMetadata(ctx context.Context, id string) (*models.Metadata, error) {
	trackID := strings.TrimPrefix(id, "itunes_")
	if s.itunesClient == nil {
		return nil, fmt.Errorf("itunes client not configured")
	}

	results, err := s.itunesClient.SearchAudiobooks(ctx, trackID, s.cfg.Region)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("item not found on itunes")
	}

	item := results[0]
	thumb := item.ArtworkUrl600
	if thumb == "" {
		thumb = item.ArtworkUrl100
	}

	displayTitle := item.TrackName
	if displayTitle == "" {
		displayTitle = item.CollectionName
	}
	if s.cfg.SimplifyTitle {
		displayTitle = utils.SimplifyTitle(displayTitle)
	}

	if thumb == "" && s.audiobookCoversClient != nil {
		covers, err := s.audiobookCoversClient.Search(ctx, displayTitle)
		if err == nil && len(covers) > 0 {
			thumb = covers[0]
		}
	}

	var moods []string
	if s.cfg.StoreAuthorAsMood && item.ArtistName != "" {
		moods = append(moods, item.ArtistName)
	}

	metadata := &models.Metadata{
		RatingKey:             id,
		GUID:                  fmt.Sprintf("itunes://audiobook/%s", trackID),
		Type:                  "album",
		Title:                 displayTitle,
		TitleSort:             displayTitle,
		Summary:               utils.CleanHTML(item.Description),
		Year:                  parseYear(item.ReleaseDate),
		OriginallyAvailableAt: item.ReleaseDate,
		Thumb:                 thumb,
		Genres:                []string{item.PrimaryGenreName},
		Moods:                 moods,
	}

	return metadata, nil
}
