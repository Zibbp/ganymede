package utils

import (
	"regexp"
	"unicode"
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
