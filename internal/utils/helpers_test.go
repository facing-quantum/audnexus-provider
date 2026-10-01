package utils

import (
	"testing"
)

func TestCleanHTML(t *testing.T) {
	input := "<p><b>Winner</b> of the award.</p><ul><li>First item</li><li>Second item</li></ul><br />Enjoy!"
	expected := "Winner of the award.\n • First item\n • Second item\n\nEnjoy!"

	got := CleanHTML(input)
	if got != expected {
		t.Errorf("CleanHTML failed.\nGot:\n%s\nExpected:\n%s", got, expected)
	}
}

func TestParseRating(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"4.9", 9.8},
		{"5.0", 10.0},
		{"3.5", 7.0},
		{"", 0.0},
		{"invalid", 0.0},
	}

	for _, tc := range tests {
		got := ParseRating(tc.input)
		if got != tc.expected {
			t.Errorf("ParseRating(%q) = %v, expected %v", tc.input, got, tc.expected)
		}
	}
}

func TestExtractRegion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Project Hail Mary [uk]", "uk"},
		{"Harry Potter [DE] (Audiobook)", "de"},
		{"No Region Included", ""},
	}

	for _, tc := range tests {
		got := ExtractRegion(tc.input)
		if got != tc.expected {
			t.Errorf("ExtractRegion(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestExtractASIN(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"B08G9PRS1K_us", "B08G9PRS1K"},
		{"Book Title [B00G0WYW92]", "B00G0WYW92"},
		{"Short text", ""},
	}

	for _, tc := range tests {
		got := ExtractASIN(tc.input)
		if got != tc.expected {
			t.Errorf("ExtractASIN(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestSortName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Andy Weir", "Weir, Andy"},
		{"Arthur Conan Doyle", "Doyle, Arthur Conan"},
		{"Martin Luther King Jr.", "King, Martin Luther, Jr."},
		{"John Smith III", "Smith, John, III"},
		{"Madonna", "Madonna"},
	}

	for _, tc := range tests {
		got := SortName(tc.input)
		if got != tc.expected {
			t.Errorf("SortName(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestSimplifyTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Project Hail Mary (Unabridged)", "Project Hail Mary"},
		{"Dune, Book 1", "Dune"},
		{"The Way of Kings: The Stormlight Archive", "The Way of Kings"},
	}

	for _, tc := range tests {
		got := SimplifyTitle(tc.input)
		if got != tc.expected {
			t.Errorf("SimplifyTitle(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestCleanSeriesSequence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Book 1", "1"},
		{"2, Dramatized Adaptation", "2"},
		{"Book 3.5", "3.5"},
		{".5", ".5"},
		{"Prequel", "Prequel"},
		{"", ""},
	}

	for _, tc := range tests {
		got := CleanSeriesSequence(tc.input)
		if got != tc.expected {
			t.Errorf("CleanSeriesSequence(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}
