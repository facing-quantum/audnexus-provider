package services

import (
	"strings"
	"testing"

	"audnexus-provider/internal/config"
)

func TestGenerateSyntheticMatch(t *testing.T) {
	cfg := &config.Config{
		SimplifyTitle:     true,
		StoreAuthorAsMood: true,
	}

	rawTitle := "The Martian (Unabridged)"
	rawAuthor := "Andy Weir"

	result := GenerateSyntheticMatch(rawTitle, rawAuthor, cfg)

	if !strings.HasPrefix(result.Result.RatingKey, "local_") {
		t.Errorf("expected rating key to start with 'local_', got %s", result.Result.RatingKey)
	}
	if result.Result.Title != "The Martian" {
		t.Errorf("expected simplified title 'The Martian', got %s", result.Result.Title)
	}
	if len(result.Result.Moods) != 1 || result.Result.Moods[0] != "Andy Weir" {
		t.Errorf("expected author Andy Weir in Moods, got %+v", result.Result.Moods)
	}
	if result.Score < 45 {
		t.Errorf("expected synthetic score >= 45, got %d", result.Score)
	}
}
