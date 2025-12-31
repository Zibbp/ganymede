package utils

import (
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
		{"illegal_fs_chars", `a/b\c:d*e?f"g<h>i|j`, "a_b_c_d_e_f_g_h_i_j"},
		{"percent_amp_semicolon", "100% legit & safe;", "100_legit_safe"},
		{"emoji_removed_inline", "🤖robot🚀", "robot"},
		{"emoji_zwj_sequence_removed", "family 👨‍👩‍👧‍👦 time", "family_time"},
		{"emoji_only_becomes_unnamed", "💯", "unnamed_file"},
		{"dot_becomes_unnamed", ".", "unnamed_file"},
		{"dotdot_becomes_unnamed", "..", "unnamed_file"},
		{"trailing_underscore_trim", " file...name  ", "file...name"},
		{"unicode_non_ascii_drops_to_separator", "café", "caf"},
		{"unicode_only_becomes_unnamed", "東京", "unnamed_file"},
		{"trim_dashes_dots_spaces", " -._~abc~_.- ", "abc"},
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
			if !isURLAndFSSafeUnreserved(got) {
				t.Fatalf("output contains non-unreserved characters: %q", got)
			}
		})
	}
}

func isURLAndFSSafeUnreserved(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '-' || c == '.' || c == '_' || c == '~':
		default:
			return false
		}
	}
	return true
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
