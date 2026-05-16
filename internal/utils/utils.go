package utils

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// Contains returns true if the slice contains the string
func Contains(s []string, e string) bool {
	for _, a := range s {
		if strings.EqualFold(a, e) {
			return true
		}
	}
	return false
}

func SecondsToHHMMSS(seconds int) string {
	duration := time.Duration(seconds) * time.Second

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds = int(duration.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// GetPathBefore returns the path before the delimiter
func GetPathBefore(path, delimiter string) string {
	index := strings.Index(path, delimiter)
	if index == -1 {
		return path
	}
	return path[:index]
}

// GetPathBeforePartial returns the path before the partialMatch
func GetPathBeforePartial(fullPath, partialMatch string) string {
	index := strings.Index(strings.ToLower(fullPath), strings.ToLower(partialMatch))
	if index == -1 {
		return fullPath
	}
	return filepath.Dir(fullPath[:index])
}

// SanitizeFileName sanitizes a string to be a safe POSIX filename (keeps non-English, strips emoji).
func SanitizeFileName(in string) string {
	const maxBytes = 255

	var b strings.Builder
	b.Grow(len(in))

	lastWasUnderscore := false
	for _, r := range in {
		// Drop emoji
		if isEmojiRune(r) {
			continue
		}

		// Treat all Unicode whitespace as separator
		if unicode.IsSpace(r) {
			if b.Len() > 0 && !lastWasUnderscore {
				b.WriteByte('_')
				lastWasUnderscore = true
			}
			continue
		}

		// Drop other control chars entirely
		if unicode.IsControl(r) || r == 0 {
			continue
		}

		// Disallow URL-unsafe/path-separator punctuation (normalize to underscore).
		// Keep non-English letters/numbers; those can be URL-encoded client-side.
		if isURLUnsafeRune(r) {
			if b.Len() > 0 && !lastWasUnderscore {
				b.WriteByte('_')
				lastWasUnderscore = true
			}
			continue
		}

		b.WriteRune(r)
		lastWasUnderscore = (r == '_')
	}

	out := b.String()
	out = strings.Trim(out, "_")

	// Handle empty/special names
	if out == "" || out == "." || out == ".." {
		out = "unnamed_file"
	}

	// Truncate to maxBytes if needed
	if len(out) > maxBytes {
		out = truncatePreserveExtUTF8(out, maxBytes)
		out = strings.Trim(out, "_")
		if out == "" || out == "." || out == ".." {
			out = "unnamed_file"
		}
	}

	return out
}

// isURLUnsafeRune reports whether r is punctuation that is unsafe in URL path segments
// Intentionally keep letters/digits (including non-English) and underscores
func isURLUnsafeRune(r rune) bool {
	if r == '/' || r == '%' {
		return true
	}

	switch r {
	case '!', '|', '#', '?', '&', ';', ':', '@', '=', '+', '$', ',', '[', ']',
		'(', ')', '*', '\'', '"', '<', '>', '\\', '^', '`', '{', '}':
		return true
	}

	return false
}

// truncatePreserveExtUTF8 truncates to max bytes without splitting UTF-8, preserving extension if any.
func truncatePreserveExtUTF8(s string, max int) string {
	if len(s) <= max {
		return s
	}

	// Preserve last extension if it looks reasonable.
	if i := strings.LastIndexByte(s, '.'); i > 0 && i < len(s)-1 {
		base := s[:i]
		ext := s[i:] // includes dot

		keep := max - len(ext)
		if keep <= 0 {
			return cutToMaxBytesUTF8(s, max)
		}

		base = cutToMaxBytesUTF8(base, keep)
		base = strings.TrimRight(base, "_")
		if base == "" {
			base = "file"
			base = cutToMaxBytesUTF8(base, max-len(ext))
		}
		return base + ext
	}

	return cutToMaxBytesUTF8(s, max)
}

// cutToMaxBytesUTF8 cuts s to at most max bytes without splitting UTF-8 runes.
func cutToMaxBytesUTF8(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := 0
	for i := range s { // i iterates rune boundaries
		if i > max {
			break
		}
		cut = i
	}
	if max < len(s) && max > 0 && max == cut {
		return s[:max]
	}
	return s[:cut]
}

// isEmojiRune reports whether r is an emoji or emoji-related rune
func isEmojiRune(r rune) bool {
	switch r {
	case 0x200D, // ZWJ
		0xFE0E, // VS15 (text)
		0xFE0F: // VS16 (emoji)
		return true
	}

	// Keycap combining mark
	if r == 0x20E3 {
		return true
	}

	// Skin tone modifiers
	if r >= 0x1F3FB && r <= 0x1F3FF {
		return true
	}

	// Regional indicator symbols (flags)
	if r >= 0x1F1E6 && r <= 0x1F1FF {
		return true
	}

	// Tag characters (subdivision flags, etc.)
	if r >= 0xE0020 && r <= 0xE007F {
		return true
	}

	// Common emoji blocks/ranges
	switch {
	case r >= 0x1F600 && r <= 0x1F64F: // Emoticons
		return true
	case r >= 0x1F300 && r <= 0x1F5FF: // Misc Symbols & Pictographs
		return true
	case r >= 0x1F680 && r <= 0x1F6FF: // Transport & Map
		return true
	case r >= 0x1F700 && r <= 0x1F77F: // Alchemical Symbols
		return true
	case r >= 0x1F780 && r <= 0x1F7FF: // Geometric Shapes Extended
		return true
	case r >= 0x1F800 && r <= 0x1F8FF: // Supplemental Arrows-C
		return true
	case r >= 0x1F900 && r <= 0x1F9FF: // Supplemental Symbols & Pictographs
		return true
	case r >= 0x1FA00 && r <= 0x1FAFF: // Symbols & Pictographs Extended-A
		return true
	case r >= 0x2600 && r <= 0x26FF: // Misc symbols (many emoji-capable)
		return true
	case r >= 0x2700 && r <= 0x27BF: // Dingbats
		return true
	case r >= 0x2300 && r <= 0x23FF: // Misc technical (some emoji-capable)
		return true
	}

	// A few singletons that commonly render as emoji
	switch r {
	case 0x00A9, 0x00AE, 0x2122: // © ® ™
		return true
	}

	return false
}
