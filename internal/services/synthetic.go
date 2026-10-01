package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"audnexus-provider/internal/config"
	"audnexus-provider/internal/models"
	"audnexus-provider/internal/utils"
)

// GenerateSyntheticMatch creates clean local metadata from user/file input when all upstreams fail
func GenerateSyntheticMatch(title, author string, cfg *config.Config) ScoreResult {
	cleanTitle := strings.TrimSpace(title)
	if cfg.SimplifyTitle {
		cleanTitle = utils.SimplifyTitle(cleanTitle)
	}

	cleanAuthor := strings.TrimSpace(author)
	if cleanAuthor == "" {
		cleanAuthor = "Unknown Author"
	}

	hashInput := fmt.Sprintf("%s:%s", strings.ToLower(cleanTitle), strings.ToLower(cleanAuthor))
	h := sha256.Sum256([]byte(hashInput))
	keyHash := hex.EncodeToString(h[:8])
	ratingKey := "local_" + keyHash

	titleSort := cleanTitle
	var moods []string
	if cfg.StoreAuthorAsMood {
		moods = []string{cleanAuthor}
	}

	metadata := models.Metadata{
		RatingKey: ratingKey,
		GUID:      fmt.Sprintf("local://audiobook/%s", keyHash),
		Type:      "album",
		Title:     cleanTitle,
		TitleSort: titleSort,
		Summary:   fmt.Sprintf("Audiobook by %s. Generated from local metadata tags.", cleanAuthor),
		Moods:     moods,
	}

	return ScoreResult{
		Result: metadata,
		Score:  50, // Above ignoreScoreBoundary (45), providing a valid match
	}
}
