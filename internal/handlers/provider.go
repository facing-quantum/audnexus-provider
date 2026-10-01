package handlers

import (
	"net/http"

	"audnexus-provider/internal/models"
	"github.com/gin-gonic/gin"
)

func RegisterProviderRoutes(r *gin.Engine) {
	r.GET("/health", healthHandler)
	r.GET("/audnexus", providerHandler)
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func providerHandler(c *gin.Context) {
	provider := models.MediaProvider{
		Identifier: models.ProviderIdentifier,
		Title:      "Audnexus Audiobook Provider",
		Version:    "2.0.0",
		Types: []models.TypeDefinition{
			{
				Type: 8, // Artist
				Scheme: []models.Scheme{
					{Scheme: models.ProviderIdentifier},
				},
			},
			{
				Type: 9, // Album
				Scheme: []models.Scheme{
					{Scheme: models.ProviderIdentifier},
				},
			},
		},
		Feature: []models.ProviderFeature{
			{Type: "metadata", Key: "/audnexus/library/metadata"},
			{Type: "match", Key: "/audnexus/library/metadata/matches"},
		},
	}
	c.JSON(http.StatusOK, models.MediaProviderResponse{
		MediaProvider: provider,
	})
}
