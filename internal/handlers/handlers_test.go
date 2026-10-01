package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"audnexus-provider/internal/api"
	"audnexus-provider/internal/cache"
	"audnexus-provider/internal/config"
	"audnexus-provider/internal/models"
	"audnexus-provider/internal/services"
	"github.com/gin-gonic/gin"
)

func setupTestRouter() (*gin.Engine, *services.SearchService, *services.MetadataService) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	cfg := &config.Config{
		Region:   "us",
		CacheTTL: 60,
	}

	appCache := cache.New()
	searchService := services.NewSearchService(nil, nil, nil, cfg)
	metadataService := services.NewMetadataService(nil, nil, nil, cfg)

	RegisterProviderRoutes(r)
	matchesHandler := NewMatchesHandler(searchService, appCache, cfg)
	matchesHandler.RegisterRoutes(r)
	metadataHandler := NewMetadataHandler(metadataService, appCache, cfg)
	metadataHandler.RegisterRoutes(r)

	return r, searchService, metadataService
}

func TestProviderEndpoint_SpecCompliance(t *testing.T) {
	r, _, _ := setupTestRouter()

	req, _ := http.NewRequest("GET", "/audnexus", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp models.MediaProviderResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal MediaProviderResponse: %v, body: %s", err, w.Body.String())
	}

	provider := resp.MediaProvider
	if provider.Identifier != "tv.plex.agents.custom.audnexus" {
		t.Errorf("expected identifier 'tv.plex.agents.custom.audnexus', got '%s'", provider.Identifier)
	}

	if len(provider.Types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(provider.Types))
	}

	hasArtist := false
	hasAlbum := false
	for _, td := range provider.Types {
		if td.Type == 8 && len(td.Scheme) > 0 && td.Scheme[0].Scheme == "tv.plex.agents.custom.audnexus" {
			hasArtist = true
		}
		if td.Type == 9 && len(td.Scheme) > 0 && td.Scheme[0].Scheme == "tv.plex.agents.custom.audnexus" {
			hasAlbum = true
		}
	}

	if !hasArtist || !hasAlbum {
		t.Errorf("missing artist (8) or album (9) type definitions with custom scheme: %+v", provider.Types)
	}

	hasMetadataFeature := false
	hasMatchFeature := false
	for _, f := range provider.Feature {
		if f.Type == "metadata" && f.Key == "/audnexus/library/metadata" {
			hasMetadataFeature = true
		}
		if f.Type == "match" && f.Key == "/audnexus/library/metadata/matches" {
			hasMatchFeature = true
		}
	}

	if !hasMetadataFeature || !hasMatchFeature {
		t.Errorf("missing metadata or match feature endpoint paths: %+v", provider.Feature)
	}
}

func TestMatchesEndpoint_PlexNumericTypeAndManualFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	audnexusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authors := []models.Author{
			{
				ASIN: "B001",
				Name: "Brandon Sanderson",
			},
		}
		json.NewEncoder(w).Encode(authors)
	}))
	defer audnexusServer.Close()

	cfg := &config.Config{Region: "us", CacheTTL: 60}
	apiClient := api.NewClientWithBaseURL(5, audnexusServer.URL)
	searchService := services.NewSearchService(apiClient, nil, nil, cfg)
	matchesHandler := NewMatchesHandler(searchService, cache.New(), cfg)
	matchesHandler.RegisterRoutes(r)

	// Test: Accept Plex numeric type (8 = Artist)
	body := []byte(`{"type": 8, "title": "Brandon Sanderson", "manual": 1}`)
	req, _ := http.NewRequest("POST", "/audnexus/library/metadata/matches", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", w.Code, w.Body.String())
	}

	var resp models.MediaContainerResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal MediaContainerResponse: %v", err)
	}

	if resp.MediaContainer.Identifier != "tv.plex.agents.custom.audnexus" {
		t.Errorf("expected identifier tv.plex.agents.custom.audnexus, got %s", resp.MediaContainer.Identifier)
	}

	if resp.MediaContainer.Size == 0 || len(resp.MediaContainer.Metadata) == 0 {
		t.Fatalf("expected metadata items in match response")
	}

	meta := resp.MediaContainer.Metadata[0]
	if meta.Key != "/audnexus/library/metadata/author_B001" {
		t.Errorf("expected key /audnexus/library/metadata/author_B001, got %s", meta.Key)
	}

	if meta.GUID != "tv.plex.agents.custom.audnexus://artist/author_B001" {
		t.Errorf("expected GUID tv.plex.agents.custom.audnexus://artist/author_B001, got %s", meta.GUID)
	}
}

func TestMetadataAndImagesEndpoint_SpecCompliance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	audnexusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		book := models.Book{
			ASIN:        "B002",
			Title:       "Oathbringer",
			Image:       "https://images.example.com/cover.jpg",
			ReleaseDate: "2017-11-14",
			Genres: []models.Genre{
				{Name: "Epic Fantasy", Type: "genre"},
			},
		}
		json.NewEncoder(w).Encode(book)
	}))
	defer audnexusServer.Close()

	cfg := &config.Config{Region: "us", CacheTTL: 60}
	apiClient := api.NewClientWithBaseURL(5, audnexusServer.URL)
	metadataService := services.NewMetadataService(apiClient, nil, nil, cfg)
	metadataHandler := NewMetadataHandler(metadataService, cache.New(), cfg)
	metadataHandler.RegisterRoutes(r)

	// 1. Test Metadata endpoint
	req, _ := http.NewRequest("GET", "/audnexus/library/metadata/album_B002", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", w.Code, w.Body.String())
	}

	// Verify raw JSON response structure has root MediaContainer and Genre array of objects
	var rawJSON map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &rawJSON); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	mc, ok := rawJSON["MediaContainer"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected MediaContainer root property in response, got %s", w.Body.String())
	}

	metaList, ok := mc["Metadata"].([]interface{})
	if !ok || len(metaList) == 0 {
		t.Fatalf("expected Metadata array in MediaContainer")
	}

	firstItem := metaList[0].(map[string]interface{})
	genres, ok := firstItem["Genre"].([]interface{})
	if !ok || len(genres) == 0 {
		t.Fatalf("expected Genre array of objects in Metadata, got %+v", firstItem["Genre"])
	}

	firstGenre := genres[0].(map[string]interface{})
	if firstGenre["tag"] != "Epic Fantasy" {
		t.Errorf("expected Genre object with tag 'Epic Fantasy', got %+v", firstGenre)
	}

	// 2. Test Images endpoint
	imgReq, _ := http.NewRequest("GET", "/audnexus/library/metadata/album_B002/images", nil)
	imgW := httptest.NewRecorder()
	r.ServeHTTP(imgW, imgReq)

	if imgW.Code != http.StatusOK {
		t.Fatalf("expected 200 from images endpoint, got %d", imgW.Code)
	}

	var imgResp models.MediaContainerResponse
	if err := json.Unmarshal(imgW.Body.Bytes(), &imgResp); err != nil {
		t.Fatalf("failed to unmarshal images response: %v", err)
	}

	if len(imgResp.MediaContainer.Image) == 0 {
		t.Fatalf("expected Image items in images response, got %+v", imgResp.MediaContainer)
	}

	if imgResp.MediaContainer.Image[0].Type != "coverPoster" || imgResp.MediaContainer.Image[0].URL != "https://images.example.com/cover.jpg" {
		t.Errorf("unexpected image item: %+v", imgResp.MediaContainer.Image[0])
	}
}
