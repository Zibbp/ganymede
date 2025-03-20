package utils

import (
	"strings"
	"testing"
)

// TestSanitizeFileName tests the SanitizeFileName function
func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic filename",
			input:    "simplefilename.txt",
			expected: "simplefilename.txt",
		},
		{
			name:     "spaces",
			input:    "file with spaces",
			expected: "file_with_spaces",
		},
		{
			name:     "windows illegal characters",
			input:    "test\\file:name*?",
			expected: "test_file_name",
		},
		{
			name:     "multiple illegal characters",
			input:    "doc/<>*|\"test",
			expected: "doc_test",
		},
		{
			name:     "null character",
			input:    "file\x00with\x00null",
			expected: "file_with_null",
		},
		{
			name:     "tabs and newlines",
			input:    "file\twith\nspecial",
			expected: "file_with_special",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  spaced out  ",
			expected: "spaced_out",
		},
		{
			name:     "multiple consecutive illegal chars",
			input:    "test///file***name???",
			expected: "test_file_name",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "unnamed_file",
		},
		{
			name:     "dot",
			input:    ".",
			expected: "unnamed_file",
		},
		{
			name:     "double dot",
			input:    "..",
			expected: "unnamed_file",
		},
		{
			name:     "backslash",
			input:    "this\\is\\a\\path",
			expected: "this_is_a_path",
		},
		{
			name:     "mixed special characters",
			input:    "file%with&some;chars",
			expected: "file_with_some_chars",
		},
		{
			name:     "very long filename",
			input:    strings.Repeat("a", 300) + ".txt",
			expected: strings.Repeat("a", 255),
		},
		{
			name:     "leading and trailing illegal",
			input:    "/start>middle<end/",
			expected: "start_middle_end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFileName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFileName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestContains tests the Contains function
func TestContains(t *testing.T) {
	tests := []struct {
		slice    []string
		element  string
		expected bool
	}{
		{[]string{"apple", "banana", "cherry"}, "banana", true},
		{[]string{"apple", "banana", "cherry"}, "BANANA", true}, // Case insensitive
		{[]string{"apple", "banana", "cherry"}, "orange", false},
		{[]string{}, "apple", false},       // Empty slice
		{[]string{"apple"}, "Apple", true}, // Single element with case difference
	}

	for _, tt := range tests {
		result := Contains(tt.slice, tt.element)
		if result != tt.expected {
			t.Errorf("Contains(%v, %s) = %t, expected %t", tt.slice, tt.element, result, tt.expected)
		}
	}
}

// TestSecondsToHHMMSS tests the SecondsToHHMMSS function
func TestSecondsToHHMMSS(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "00:00:00"},      // Zero seconds
		{3661, "01:01:01"},   // 1 hour, 1 minute, 1 second
		{359999, "99:59:59"}, // Large number under 100 hours
		{60, "00:01:00"},     // 1 minute
		{3600, "01:00:00"},   // 1 hour
	}

	for _, tt := range tests {
		result := SecondsToHHMMSS(tt.input)
		if result != tt.expected {
			t.Errorf("SecondsToHHMMSS(%d) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

// TestGetPathBefore tests the GetPathBefore function
func TestGetPathBefore(t *testing.T) {
	tests := []struct {
		path      string
		delimiter string
		expected  string
	}{
		{"/home/user/docs", "/", ""},
		{"file.txt", ".", "file"},
		{"no-delimiter-here", "/", "no-delimiter-here"}, // No delimiter
		{"/start/middle/end", "/middle", "/start"},
		{"", "/", ""}, // Empty string
	}

	for _, tt := range tests {
		result := GetPathBefore(tt.path, tt.delimiter)
		if result != tt.expected {
			t.Errorf("GetPathBefore(%s, %s) = %s, expected %s", tt.path, tt.delimiter, result, tt.expected)
		}
	}
}

// TestGetPathBeforePartial tests the GetPathBeforePartial function
func TestGetPathBeforePartial(t *testing.T) {
	tests := []struct {
		fullPath     string
		partialMatch string
		expected     string
	}{
		{"/home/user/docs/file.txt", "docs", "/home/user"},
		{"/home/USER/docs", "user", "/home"},
		{"file.txt", "missing", "file.txt"}, // No match returns dir of full path
		{"/a/b/c/d", "C", "/a/b"},
		{"", "test", ""}, // Empty path
	}

	for _, tt := range tests {
		result := GetPathBeforePartial(tt.fullPath, tt.partialMatch)
		if result != tt.expected {
			t.Errorf("GetPathBeforePartial(%s, %s) = %s, expected %s", tt.fullPath, tt.partialMatch, result, tt.expected)
		}
	}
}
