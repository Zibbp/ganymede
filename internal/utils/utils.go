package utils

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// SanitizeFileName returns a sanitized version of the input string that is safe for use as a file name
func SanitizeFileName(fileName string) string {
	illegalChars := []string{
		"/", "\\", ":", "*", "?", "\"", "<", ">", "|",
		"\x00",
		"%",
		"&",
		";",
	}

	// Replace all illegal characters with underscore
	for _, char := range illegalChars {
		fileName = strings.ReplaceAll(fileName, char, "_")
	}

	// Handle whitespace (space, tab, newline)
	fileName = strings.TrimSpace(fileName)
	fileName = strings.ReplaceAll(fileName, " ", "_")
	fileName = strings.ReplaceAll(fileName, "\t", "_")
	fileName = strings.ReplaceAll(fileName, "\n", "_")

	// Collapse multiple consecutive underscores
	for strings.Contains(fileName, "__") {
		fileName = strings.ReplaceAll(fileName, "__", "_")
	}

	// Ensure reasonable length
	if len(fileName) > 255 {
		fileName = fileName[:255]
	}

	fileName = strings.Trim(fileName, "_")

	// Handle empty result or reserved names
	if fileName == "" || fileName == "." || fileName == ".." {
		return "unnamed_file"
	}

	return fileName
}

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
