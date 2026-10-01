package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"audnexus-provider/internal/api"
	"audnexus-provider/internal/cache"
	"audnexus-provider/internal/config"
	"audnexus-provider/internal/handlers"
	"audnexus-provider/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var (
	help    bool
	version bool
	logger  = logrus.New()
)

func init() {
	flag.BoolVar(&help, "help", false, "Show help message")
	flag.BoolVar(&version, "version", false, "Show version information")
}

func main() {
	flag.Parse()

	if help {
		fmt.Println("audnexus-provider - Audnexus Provider Service")
		fmt.Println()
		fmt.Println("Usage: audnexus-provider [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	if version {
		fmt.Println("audnexus-provider version 2.0.0")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	logger.Infof("audnexus-provider starting...")
	logger.Infof("Region: %s", cfg.Region)
	logger.Infof("Log Level: %s", cfg.LogLevel)
	logger.Infof("Cache TTL: %d seconds", cfg.CacheTTL)
	logger.Infof("Audnexus Timeout: %d seconds", cfg.AudnexusTimeout)

	// Set log level from config
	switch cfg.LogLevel {
	case "DEBUG":
		logger.SetLevel(logrus.DebugLevel)
	case "INFO":
		logger.SetLevel(logrus.InfoLevel)
	case "WARN":
		logger.SetLevel(logrus.WarnLevel)
	case "ERROR":
		logger.SetLevel(logrus.ErrorLevel)
	}

	// Set Gin to release mode in production
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(loggerMiddleware())

	// Initialize persistent cache and API clients
	appCache := cache.New()
	persistentCache, err := cache.NewPersistentCache(cfg.CacheDir, time.Duration(cfg.CacheTTL)*time.Second)
	if err != nil {
		logger.Warnf("Failed to initialize persistent disk cache at %s: %v, falling back to memory only", cfg.CacheDir, err)
	}

	apiClient := api.NewClient(cfg.AudnexusTimeout)
	audibleClient := api.NewAudibleClient(cfg.AudnexusTimeout)
	itunesClient := api.NewITunesClient(cfg.AudnexusTimeout)
	googleBooksClient := api.NewGoogleBooksClient(cfg.AudnexusTimeout)
	openLibClient := api.NewOpenLibraryClient(cfg.AudnexusTimeout)
	audiobookCoversClient := api.NewAudiobookCoversClient(cfg.AudnexusTimeout)

	searchService := services.NewAudiobookshelfPipelineSearchService(apiClient, audibleClient, itunesClient, googleBooksClient, openLibClient, cfg)
	metadataService := services.NewEnrichedMetadataService(apiClient, audibleClient, itunesClient, audiobookCoversClient, persistentCache, cfg)

	// Register routes
	handlers.RegisterProviderRoutes(router)
	matchesHandler := handlers.NewMatchesHandler(searchService, appCache, cfg)
	matchesHandler.RegisterRoutes(router)
	metadataHandler := handlers.NewMetadataHandler(metadataService, appCache, cfg)
	metadataHandler.RegisterRoutes(router)

	logger.Info("Server initialization complete")

	addr := ":" + strconv.Itoa(cfg.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		logger.Infof("Listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown:", err)
	}
	logger.Info("Server exiting")
}

func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Infof("%s %s %s", c.Request.Method, c.Request.URL.Path, c.ClientIP())
		c.Next()
	}
}
