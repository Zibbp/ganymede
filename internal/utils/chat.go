package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/rs/zerolog/log"
)

// find the substring position in a string. Supports passing an occurrence to find the Nth place of the substring in the string
func findSubstringPositions(input string, substring string, occurrenceNumber int) (start int, end int, found bool) {
	var re *regexp.Regexp
	if isAlphanumeric(substring) {
		// add word boundaries for alphanumeric substrings
		re = regexp.MustCompile(`\b` + regexp.QuoteMeta(substring) + `\b`)
	} else {
		// use exact match for non-alphanumeric substrings
		re = regexp.MustCompile(regexp.QuoteMeta(substring))
	}
	matches := re.FindAllStringIndex(input, -1)

	if occurrenceNumber > 0 && occurrenceNumber <= len(matches) {
		startIndex := matches[occurrenceNumber-1][0]
		endIndex := matches[occurrenceNumber-1][1]
		return startIndex, endIndex, true
	}

	return -1, -1, false
}

// checks if the string contains only alphanumeric characters
func isAlphanumeric(str string) bool {
	for _, char := range str {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			return false
		}
	}
	return true
}

// ConvertNDJSONToJSONArray converts a newline-delimited JSON file to a JSON array.
func ConvertNDJSONToJSONArrayInPlace(filePath string) error {
	// Check if file starts with '['
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	head := make([]byte, 1)
	if _, err := f.Read(head); err != nil {
		return fmt.Errorf("read head: %w", err)
	}
	if head[0] == '[' {
		log.Info().Msg("File already JSON array. Skipping conversion.")
		return nil
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek start: %w", err)
	}

	// Temp output file
	tmpPath := filePath + ".tmp"
	tmp, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer tmp.Close()

	scanner := bufio.NewScanner(f)
	if _, err := tmp.WriteString("[\n"); err != nil {
		return err
	}

	first := true
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !first {
			tmp.WriteString(",\n")
		}
		tmp.WriteString(line)
		first = false
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	if _, err := tmp.WriteString("\n]\n"); err != nil {
		return fmt.Errorf("write closing: %w", err)
	}
	tmp.Close()
	f.Close()

	// Replace original file
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("replace original: %w", err)
	}

	return nil
}
