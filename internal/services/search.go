package services

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"audnexus-provider/internal/api"
	"audnexus-provider/internal/config"
	"audnexus-provider/internal/models"
	"audnexus-provider/internal/resilience"
	"audnexus-provider/internal/utils"
	"golang.org/x/sync/singleflight"
)

type SearchService struct {
	client            *api.Client
	audibleClient     *api.AudibleClient
	itunesClient      *api.ITunesClient
	googleBooksClient *api.GoogleBooksClient
	openLibraryClient *api.OpenLibraryClient
	cfg               *config.Config

	audibleCB *resilience.CircuitBreaker
	itunesCB  *resilience.CircuitBreaker
	sfGroup   singleflight.Group
}

func NewSearchService(
	client *api.Client,
	audibleClient *api.AudibleClient,
	itunesClient *api.ITunesClient,
	cfg *config.Config,
) *SearchService {
	return NewResilientSearchService(client, audibleClient, itunesClient, nil, cfg)
}

func NewResilientSearchService(
	client *api.Client,
	audibleClient *api.AudibleClient,
	itunesClient *api.ITunesClient,
	openLibraryClient *api.OpenLibraryClient,
	cfg *config.Config,
) *SearchService {
	return NewAudiobookshelfPipelineSearchService(client, audibleClient, itunesClient, nil, openLibraryClient, cfg)
}

func NewAudiobookshelfPipelineSearchService(
	client *api.Client,
	audibleClient *api.AudibleClient,
	itunesClient *api.ITunesClient,
	googleBooksClient *api.GoogleBooksClient,
	openLibraryClient *api.OpenLibraryClient,
	cfg *config.Config,
) *SearchService {
	return &SearchService{
		client:            client,
		audibleClient:     audibleClient,
		itunesClient:      itunesClient,
		googleBooksClient: googleBooksClient,
		openLibraryClient: openLibraryClient,
		cfg:               cfg,
		audibleCB:         resilience.NewCircuitBreaker("audible", 3, 30*time.Second),
		itunesCB:          resilience.NewCircuitBreaker("itunes", 3, 30*time.Second),
	}
}

func (s *SearchService) getRegion(query string) string {
	if reg := utils.ExtractRegion(query); reg != "" {
		return reg
	}
	return s.cfg.Region
}

func (s *SearchService) SearchAuthors(ctx context.Context, query string) ([]ScoreResult, error) {
	region := s.getRegion(query)
	sfKey := fmt.Sprintf("author:%s:%s", region, strings.ToLower(query))

	res, err, _ := s.sfGroup.Do(sfKey, func() (interface{}, error) {
		// ASIN quick match
		if asin := utils.ExtractASIN(query); asin != "" && s.client != nil {
			author, err := s.client.GetAuthorByASIN(ctx, asin, region)
			if err == nil && author != nil && author.Name != "" {
				return []ScoreResult{{
					Result: models.Metadata{
						RatingKey: "author_" + author.ASIN,
						GUID:      "audnexus://author/" + author.ASIN,
						Type:      "artist",
						Title:     author.Name,
						Thumb:     author.Image,
					},
					Score: 100,
				}}, nil
			}
		}

		if s.client == nil {
			return []ScoreResult{}, nil
		}

		authors, err := s.client.SearchAuthors(ctx, query, region)
		if err != nil {
			return nil, err
		}

		var results []ScoreResult
		for _, author := range authors {
			score := ScoreAuthor(query, author)
			if !IsAboveIgnoreBoundary(score) {
				continue
			}

			results = append(results, ScoreResult{
				Result: models.Metadata{
					RatingKey: "author_" + author.ASIN,
					GUID:      "audnexus://author/" + author.ASIN,
					Type:      "artist",
					Title:     author.Name,
					Thumb:     author.Image,
				},
				Score: score,
			})
		}

		sort.Slice(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})

		return results, nil
	})

	if err != nil {
		return nil, err
	}
	return res.([]ScoreResult), nil
}

func (s *SearchService) SearchBooks(ctx context.Context, title, author string) ([]ScoreResult, error) {
	region := s.getRegion(title + " " + author)
	sfKey := fmt.Sprintf("book:%s:%s:%s", region, strings.ToLower(title), strings.ToLower(author))

	res, err, _ := s.sfGroup.Do(sfKey, func() (interface{}, error) {
		// 1. ASIN quick match
		if asin := utils.ExtractASIN(title); asin != "" && s.client != nil {
			book, err := s.client.GetBookByASIN(ctx, asin, region)
			if err == nil && book != nil && book.Title != "" {
				return []ScoreResult{{Result: s.bookToMetadata(*book), Score: 100}}, nil
			}
		}

		var results []ScoreResult

		// 2. Primary: Audible with Circuit Breaker
		if s.audibleClient != nil {
			_ = s.audibleCB.Execute(func() error {
				audibleProducts, err := s.audibleClient.SearchProducts(ctx, title, author, region)
				if err != nil {
					return err
				}
				for i, product := range audibleProducts {
					if utils.IsPreOrder(product.ReleaseDate) {
						continue
					}

					score := ScoreAudibleProduct(title, author, product, i)
					if !IsAboveIgnoreBoundary(score) {
						continue
					}

					results = append(results, ScoreResult{
						Result: s.audibleProductToMetadata(product),
						Score:  score,
					})

					if len(results) > 1 && IsGoodScore(score) {
						break
					}
				}
				return nil
			})
		}

		// 3. Fallback: iTunes with Circuit Breaker
		if len(results) == 0 && s.itunesClient != nil {
			_ = s.itunesCB.Execute(func() error {
				searchTerm := title
				if author != "" {
					searchTerm = title + " " + author
				}
				itunesItems, err := s.itunesClient.SearchAudiobooks(ctx, searchTerm, region)
				if err != nil {
					return err
				}
				for i, item := range itunesItems {
					score := ScoreITunesResult(title, author, item, i)
					if !IsAboveIgnoreBoundary(score) {
						continue
					}

					results = append(results, ScoreResult{
						Result: s.itunesResultToMetadata(item),
						Score:  score,
					})

					if len(results) > 1 && IsGoodScore(score) {
						break
					}
				}
				return nil
			})
		}

		// 3. Fallback: Google Books (matching Audiobookshelf)
		if len(results) == 0 && s.googleBooksClient != nil {
			gbItems, err := s.googleBooksClient.Search(ctx, title, author)
			if err == nil && len(gbItems) > 0 {
				for _, item := range gbItems {
					results = append(results, ScoreResult{
						Result: s.googleBooksItemToMetadata(item),
						Score:  65,
					})
				}
			}
		}

		// 4. Fallback: Open Library
		if len(results) == 0 && s.openLibraryClient != nil {
			docs, err := s.openLibraryClient.SearchBooks(ctx, title, author)
			if err == nil && len(docs) > 0 {
				for _, doc := range docs {
					thumb := ""
					if doc.CoverI > 0 {
						thumb = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", doc.CoverI)
					}
					cleanTitle := doc.Title
					if s.cfg.SimplifyTitle {
						cleanTitle = utils.SimplifyTitle(cleanTitle)
					}
					results = append(results, ScoreResult{
						Result: models.Metadata{
							RatingKey: "openlibrary_" + strings.TrimPrefix(doc.Key, "/works/"),
							GUID:      "openlibrary://book" + doc.Key,
							Type:      "album",
							Title:     cleanTitle,
							Year:      doc.FirstPublishYear,
							Thumb:     thumb,
							Moods:     doc.AuthorName,
						},
						Score: 60,
					})
				}
			}
		}

		// 5. Final Safety Net: Clean Synthetic Match
		if len(results) == 0 {
			results = append(results, GenerateSyntheticMatch(title, author, s.cfg))
		}

		sort.Slice(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})

		return results, nil
	})

	if err != nil {
		return nil, err
	}
	return res.([]ScoreResult), nil
}

func (s *SearchService) audibleProductToMetadata(product models.AudibleProduct) models.Metadata {
	displayTitle := product.Title
	if s.cfg.SimplifyTitle {
		displayTitle = utils.SimplifyTitle(displayTitle)
	}

	var authors []string
	for _, a := range product.Authors {
		if a.Name != "" {
			authors = append(authors, a.Name)
		}
	}

	var narrators []string
	for _, n := range product.Narrators {
		if n.Name != "" {
			narrators = append(narrators, n.Name)
		}
	}

	year := parseYear(product.ReleaseDate)

	metadata := models.Metadata{
		RatingKey:             "album_" + product.ASIN,
		GUID:                  "audnexus://album/" + product.ASIN,
		Type:                  "album",
		Title:                 displayTitle,
		Year:                  year,
		OriginallyAvailableAt: product.ReleaseDate,
		Studio:                product.Publisher,
		Styles:                narrators,
	}

	if s.cfg.StoreAuthorAsMood {
		metadata.Moods = authors
	}

	return metadata
}

func (s *SearchService) itunesResultToMetadata(item models.ITunesResult) models.Metadata {
	displayTitle := item.TrackName
	if displayTitle == "" {
		displayTitle = item.CollectionName
	}
	if s.cfg.SimplifyTitle {
		displayTitle = utils.SimplifyTitle(displayTitle)
	}

	thumb := item.ArtworkUrl600
	if thumb == "" {
		thumb = item.ArtworkUrl100
	}

	metadata := models.Metadata{
		RatingKey:             fmt.Sprintf("itunes_%d", item.TrackID),
		GUID:                  fmt.Sprintf("itunes://audiobook/%d", item.TrackID),
		Type:                  "album",
		Title:                 displayTitle,
		Summary:               utils.CleanHTML(item.Description),
		Thumb:                 thumb,
		Year:                  parseYear(item.ReleaseDate),
		OriginallyAvailableAt: item.ReleaseDate,
		Genres:                []string{item.PrimaryGenreName},
	}

	if s.cfg.StoreAuthorAsMood && item.ArtistName != "" {
		metadata.Moods = []string{item.ArtistName}
	}

	return metadata
}

func (s *SearchService) googleBooksItemToMetadata(item models.GoogleBooksItem) models.Metadata {
	displayTitle := item.VolumeInfo.Title
	if s.cfg.SimplifyTitle {
		displayTitle = utils.SimplifyTitle(displayTitle)
	}

	thumb := ""
	if item.VolumeInfo.ImageLinks != nil {
		if t, ok := item.VolumeInfo.ImageLinks["extraLarge"]; ok {
			thumb = t
		} else if t, ok := item.VolumeInfo.ImageLinks["large"]; ok {
			thumb = t
		} else if t, ok := item.VolumeInfo.ImageLinks["medium"]; ok {
			thumb = t
		} else if t, ok := item.VolumeInfo.ImageLinks["thumbnail"]; ok {
			thumb = t
		}
	}
	thumb = strings.Replace(thumb, "http://", "https://", 1)

	metadata := models.Metadata{
		RatingKey:             "googlebooks_" + item.ID,
		GUID:                  "googlebooks://book/" + item.ID,
		Type:                  "album",
		Title:                 displayTitle,
		Summary:               utils.CleanHTML(item.VolumeInfo.Description),
		Thumb:                 thumb,
		Year:                  parseYear(item.VolumeInfo.PublishedDate),
		OriginallyAvailableAt: item.VolumeInfo.PublishedDate,
		Studio:                item.VolumeInfo.Publisher,
		Genres:                item.VolumeInfo.Categories,
	}

	if s.cfg.StoreAuthorAsMood && len(item.VolumeInfo.Authors) > 0 {
		metadata.Moods = item.VolumeInfo.Authors
	}

	return metadata
}

func (s *SearchService) bookToMetadata(book models.Book) models.Metadata {
	displayTitle := book.Title
	if s.cfg.SimplifyTitle {
		displayTitle = utils.SimplifyTitle(book.Title)
	}

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

	metadata := models.Metadata{
		RatingKey:             "album_" + book.ASIN,
		GUID:                  "audnexus://album/" + book.ASIN,
		Type:                  "album",
		Title:                 displayTitle,
		Year:                  parseYear(book.ReleaseDate),
		OriginallyAvailableAt: book.ReleaseDate,
		Thumb:                 book.Image,
		Rating:                utils.ParseRating(book.Rating),
		Studio:                book.PublisherName,
		Styles:                narratorNames,
	}

	if s.cfg.StoreAuthorAsMood {
		metadata.Moods = append(metadata.Moods, authorNames...)
	}

	if book.SeriesPrimary != nil && book.SeriesPrimary.Name != "" {
		metadata.Moods = append(metadata.Moods, "Series: "+book.SeriesPrimary.Name)
	}
	if book.SeriesSecondary != nil && book.SeriesSecondary.Name != "" {
		metadata.Moods = append(metadata.Moods, "Series: "+book.SeriesSecondary.Name)
	}

	return metadata
}

func parseYear(dateStr string) int {
	if len(dateStr) >= 4 {
		if y, err := strconv.Atoi(dateStr[:4]); err == nil {
			return y
		}
	}
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t.Year()
	}
	return 0
}
