package utils

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	asinRegex   = regexp.MustCompile(`(?i)[A-Z0-9]{10}`)
	regionRegex = regexp.MustCompile(`(?i)\[([A-Za-z]{2})\]`)
	htmlTagRe   = regexp.MustCompile(`<[^>]+>`)
	suffixRe    = regexp.MustCompile(`(?i)^(.*?)(?:,?\s+([JS]r\.?|III?|IV|V))$`)
)

// ExtractASIN extracts an ASIN (10 alphanumeric chars) from a string
func ExtractASIN(s string) string {
	matches := asinRegex.FindString(s)
	return strings.ToUpper(matches)
}

// ExtractRegion extracts a 2-letter region code in brackets like [uk] or [de]
func ExtractRegion(s string) string {
	m := regionRegex.FindStringSubmatch(s)
	if len(m) > 1 {
		return strings.ToLower(m[1])
	}
	return ""
}

// NormalizeString strips diacritics, converts to lowercase, and trims punctuation/spaces
func NormalizeString(s string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, _ := transform.String(t, s)
	lower := strings.ToLower(result)

	// Clean out punctuation and extra spaces for matching
	var b strings.Builder
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}

// CleanHTML formats HTML elements to readable text and strips all HTML tags
func CleanHTML(input string) string {
	if input == "" {
		return ""
	}
	s := input
	s = strings.ReplaceAll(s, "<ul>", "")
	s = strings.ReplaceAll(s, "</ul>", "\n")
	s = strings.ReplaceAll(s, "<ol>", "")
	s = strings.ReplaceAll(s, "</ol>", "\n")
	s = strings.ReplaceAll(s, "<li>", " • ")
	s = strings.ReplaceAll(s, "</li>", "\n")
	s = strings.ReplaceAll(s, "<br>", "")
	s = strings.ReplaceAll(s, "<br/>", "")
	s = strings.ReplaceAll(s, "<br />", "")
	s = strings.ReplaceAll(s, "<p>", "")
	s = strings.ReplaceAll(s, "</p>", "\n")

	s = htmlTagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)

	// Collapse 3 or more consecutive newlines to 2
	consecNewlines := regexp.MustCompile(`\n{3,}`)
	s = consecNewlines.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
}

// ParseRating parses a 5-star rating and scales it by 2 for Plex (10-point scale)
func ParseRating(r string) float64 {
	if r == "" {
		return 0
	}
	val, err := strconv.ParseFloat(strings.TrimSpace(r), 64)
	if err != nil {
		return 0
	}
	scaled := val * 2.0
	if scaled > 10.0 {
		return 10.0
	}
	return scaled
}

// ToInitials converts a name to initials with surname (e.g. 'Arthur Conan Doyle' -> 'A.C.Doyle')
func ToInitials(name string) string {
	name = strings.ReplaceAll(name, "\"", "")
	parts := strings.Fields(name)
	if len(parts) <= 1 {
		return name
	}

	var b strings.Builder
	for _, part := range parts[:len(parts)-1] {
		cleaned := strings.Trim(part, ".")
		if len(cleaned) > 0 {
			b.WriteByte(cleaned[0])
			b.WriteByte('.')
		}
	}
	b.WriteString(parts[len(parts)-1])
	return b.String()
}

// SimplifyTitle removes extra endings, series labels, and unabridged/abridged tags
func SimplifyTitle(title string) string {
	result := regexp.MustCompile(`(?i)\s*\(?(un)?abridged\)?\s*$`).ReplaceAllString(title, "")
	result = regexp.MustCompile(`(?i),?\s*book\s+[\w\s-]+\s*$`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`:\s*.+$`).ReplaceAllString(result, "")
	return strings.TrimSpace(result)
}

// SortName converts 'First Last' to 'Last, First' preserving suffixes (e.g. Jr, III)
func SortName(name string) string {
	name = strings.TrimSpace(name)
	suffix := ""
	if m := suffixRe.FindStringSubmatch(name); len(m) > 2 && m[2] != "" {
		name = m[1]
		suffix = ", " + m[2]
	}

	parts := strings.Fields(name)
	if len(parts) <= 1 {
		return name + suffix
	}
	return parts[len(parts)-1] + ", " + strings.Join(parts[:len(parts)-1], " ") + suffix
}

// IsPreOrder checks if a release date is in the future
func IsPreOrder(releaseDateStr string) bool {
	if releaseDateStr == "" {
		return false
	}
	// Try parsing RFC3339 or YYYY-MM-DD
	t, err := time.Parse(time.RFC3339, releaseDateStr)
	if err != nil {
		t, err = time.Parse("2006-01-02", releaseDateStr)
	}
	if err != nil {
		return false
	}
	return t.After(time.Now())
}

var sequenceNumberRe = regexp.MustCompile(`\.\d+|\d+(?:\.\d+)?`)

// CleanSeriesSequence cleans series sequence text to decimal/integer format (matching Audiobookshelf)
func CleanSeriesSequence(sequence string) string {
	sequence = strings.TrimSpace(sequence)
	if sequence == "" {
		return ""
	}
	m := sequenceNumberRe.FindString(sequence)
	if m != "" {
		return m
	}
	return sequence
}
