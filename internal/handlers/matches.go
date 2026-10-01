package handlers

import (
	"net/http"
	"strings"

	"audnexus-provider/internal/cache"
	"audnexus-provider/internal/config"
	"audnexus-provider/internal/models"
	"audnexus-provider/internal/services"
	"github.com/gin-gonic/gin"
)

// MatchesHandler handles match/search requests
type MatchesHandler struct {
	searchService *services.SearchService
	cache         *cache.Cache
	cfg           *config.Config
}

// NewMatchesHandler creates a new MatchesHandler
func NewMatchesHandler(searchService *services.SearchService, c *cache.Cache, cfg *config.Config) *MatchesHandler {
	return &MatchesHandler{
		searchService: searchService,
		cache:         c,
		cfg:           cfg,
	}
}

// RegisterRoutes registers the matches routes
func (h *MatchesHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/audnexus/library/metadata/matches", h.handleMatches)
}

// handleMatches handles POST /matches requests
func (h *MatchesHandler) handleMatches(c *gin.Context) {
	var req struct {
		Title  string      `json:"title"`
		Author string      `json:"author"`
		Type   interface{} `json:"type"`
		Manual int         `json:"manual"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Determine search type (support both string "artist"/"album" and Plex numeric types 8/9)
	searchType := "album"
	switch v := req.Type.(type) {
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		if s == "artist" || s == "show" {
			searchType = "artist"
		} else if s == "album" || s == "track" || s == "movie" {
			searchType = "album"
		}
	case float64:
		if int(v) == 8 || int(v) == 2 {
			searchType = "artist"
		} else if int(v) == 9 || int(v) == 10 || int(v) == 1 {
			searchType = "album"
		}
	case int:
		if v == 8 || v == 2 {
			searchType = "artist"
		} else if v == 9 || v == 10 || v == 1 {
			searchType = "album"
		}
	}

	var results []services.ScoreResult
	var err error

	switch searchType {
	case "artist":
		results, err = h.searchService.SearchAuthors(c.Request.Context(), req.Title)
	case "album":
		results, err = h.searchService.SearchBooks(c.Request.Context(), req.Title, req.Author)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type: must be 'artist' (8) or 'album' (9)"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If manual == 0 (automatic scanner match), return only best match per Plex specification
	if req.Manual != 1 && len(results) > 1 {
		results = results[:1]
	}

	// Convert results to metadata
	metadata := make([]models.Metadata, 0, len(results))
	for _, r := range results {
		item := r.Result
		item.SyncTags()
		metadata = append(metadata, item)
	}

	c.JSON(http.StatusOK, models.MediaContainerResponse{
		MediaContainer: models.MediaContainer{
			Offset:     0,
			TotalSize:  len(metadata),
			Identifier: models.ProviderIdentifier,
			Size:       len(metadata),
			Metadata:   metadata,
		},
	})
}
