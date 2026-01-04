package utils

import (
	"strings"
	"testing"
)

// TestSanitizeFileName tests the SanitizeFileName function
func TestSanitizeFileName_Table(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"basic_spaces", "hello world", "hello_world"},
		{"tabs_newlines_trim", "  hello\tworld\n", "hello_world"},

		// POSIX + URL-safety: treat "/" and "%" as separators.
		{"posix_slash_and_percent", `a/b%c`, "a_b_c"},

		// We keep other punctuation (since this never runs on Windows).
		{"punctuation_kept", `100 legit & safe;`, `100_legit_&_safe;`},

		{"emoji_removed_inline", "🤖robot🚀", "robot"},
		{"emoji_zwj_sequence_removed", "family 👨‍👩‍👧‍👦 time", "family_time"},
		{"emoji_only_becomes_unnamed", "💯", "unnamed_file"},
		{"dot_becomes_unnamed", ".", "unnamed_file"},
		{"dotdot_becomes_unnamed", "..", "unnamed_file"},
		{"trailing_underscore_trim", " file...name  ", "file...name"},

		// Non-English preserved
		{"unicode_non_english_preserved", "café", "café"},
		{"unicode_cjk_preserved", "東京", "東京"},
		{"unicode_mixed_preserved", "東京 café", "東京_café"},

		// Only underscores are trimmed at ends
		{"trim_underscores_only", " _abc_ ", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFileName(tt.in)
			if got != tt.want {
				t.Fatalf("SanitizeFileName(%q) = %q; want %q", tt.in, got, tt.want)
			}
			if len(got) == 0 {
				t.Fatalf("output must not be empty")
			}
			if len(got) > 255 {
				t.Fatalf("output too long: %d", len(got))
			}
			if strings.ContainsRune(got, '/') {
				t.Fatalf("output must not contain '/': %q", got)
			}
			if strings.Contains(got, "\x00") {
				t.Fatalf("output must not contain NUL: %q", got)
			}
			if strings.Contains(got, "__") {
				t.Fatalf("output must not contain double underscores: %q", got)
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
