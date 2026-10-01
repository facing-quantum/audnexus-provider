package services

import (
	"strings"

	"github.com/agnivade/levenshtein"
	"audnexus-provider/internal/models"
	"audnexus-provider/internal/utils"
)

const (
	initialScore        = 100
	ignoreScoreBoundary = 45
	goodScoreThreshold  = 98
)

type ScoreResult struct {
	Result models.Metadata
	Score  int
}

// ScoreAuthor scores author candidate based on Levenshtein distance
func ScoreAuthor(query string, author models.Author) int {
	normQuery := utils.NormalizeString(query)
	normName := utils.NormalizeString(author.Name)

	if normQuery == normName {
		return 100
	}

	dist := levenshtein.ComputeDistance(normQuery, normName)
	score := initialScore - (dist * 10)
	if score < 0 {
		score = 0
	}
	return score
}

// ScoreAudibleProduct scores an Audible product based on title and author
func ScoreAudibleProduct(queryTitle, queryAuthor string, product models.AudibleProduct, index int) int {
	allDeductions := 0

	// Title deduction
	normQueryTitle := utils.NormalizeString(queryTitle)
	normProdTitle := utils.NormalizeString(product.Title)
	if normQueryTitle != normProdTitle {
		titleDist := levenshtein.ComputeDistance(normQueryTitle, normProdTitle)
		allDeductions += titleDist * 2
	}

	// Author deduction
	if queryAuthor != "" && len(product.Authors) > 0 {
		normQueryAuthor := utils.NormalizeString(queryAuthor)
		bestAuthorDist := 999
		for _, author := range product.Authors {
			normAuthor := utils.NormalizeString(author.Name)
			dist := levenshtein.ComputeDistance(normQueryAuthor, normAuthor)
			if dist < bestAuthorDist {
				bestAuthorDist = dist
			}
		}
		if bestAuthorDist != 999 {
			allDeductions += bestAuthorDist * 10
		}
	} else if queryAuthor != "" && len(product.Authors) == 0 {
		allDeductions += 20
	}

	finalScore := initialScore - allDeductions - index
	if finalScore < 0 {
		finalScore = 0
	}
	return finalScore
}

// ScoreITunesResult scores an iTunes result based on title and artist
func ScoreITunesResult(queryTitle, queryAuthor string, item models.ITunesResult, index int) int {
	allDeductions := 0

	title := item.TrackName
	if title == "" {
		title = item.CollectionName
	}

	normQueryTitle := utils.NormalizeString(queryTitle)
	normItemTitle := utils.NormalizeString(title)
	if normQueryTitle != normItemTitle {
		titleDist := levenshtein.ComputeDistance(normQueryTitle, normItemTitle)
		allDeductions += titleDist * 2
	}

	if queryAuthor != "" && item.ArtistName != "" {
		normQueryAuthor := utils.NormalizeString(queryAuthor)
		normItemAuthor := utils.NormalizeString(item.ArtistName)
		authorDist := levenshtein.ComputeDistance(normQueryAuthor, normItemAuthor)
		allDeductions += authorDist * 10
	}

	finalScore := initialScore - allDeductions - index
	if finalScore < 0 {
		finalScore = 0
	}
	return finalScore
}

// ScoreBook maintains compatibility for direct models.Book scoring
func ScoreBook(queryTitle, queryAuthor string, book models.Book) int {
	allDeductions := 0

	normQueryTitle := utils.NormalizeString(queryTitle)
	normBookTitle := utils.NormalizeString(book.Title)
	if normQueryTitle != normBookTitle {
		titleDist := levenshtein.ComputeDistance(normQueryTitle, normBookTitle)
		allDeductions += titleDist * 2
	}

	if queryAuthor != "" && len(book.Authors) > 0 {
		normQueryAuthor := utils.NormalizeString(queryAuthor)
		bestDist := 999
		for _, a := range book.Authors {
			dist := levenshtein.ComputeDistance(normQueryAuthor, utils.NormalizeString(a.Name))
			if dist < bestDist {
				bestDist = dist
			}
		}
		if bestDist != 999 {
			allDeductions += bestDist * 10
		}
	}

	score := initialScore - allDeductions
	if score < 0 {
		score = 0
	}
	return score
}

func IsGoodScore(score int) bool {
	return score >= goodScoreThreshold
}

func IsAboveIgnoreBoundary(score int) bool {
	return score >= ignoreScoreBoundary
}

func FormatInitialsDescription(title, author, narrator string) string {
	titleTrunc := title
	if len(title) > 36 {
		if len(title) > 30 {
			titleTrunc = title[:30] + ".."
		}
	}
	return "\"" + titleTrunc + "\" by " + utils.ToInitials(author) + " w/ " + utils.ToInitials(narrator)
}

func ConcatContributors(contributors []models.Contributor) string {
	names := make([]string, 0, len(contributors))
	for _, c := range contributors {
		if c.Name != "" {
			names = append(names, c.Name)
		}
	}
	return strings.Join(names, ", ")
}
