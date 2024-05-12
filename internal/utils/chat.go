package utils

import "regexp"

// find the substring position in a string. Supports passing an occurrence to find the Nth place of the substring in the string
func findSubstringPositions(input string, substring string, occurrenceNumber int) (start int, end int, found bool) {
	re := regexp.MustCompile(regexp.QuoteMeta(substring))
	matches := re.FindAllStringIndex(input, -1)

	if occurrenceNumber <= len(matches) {
		startIndex := matches[occurrenceNumber-1][0]
		endIndex := matches[occurrenceNumber-1][1]
		return startIndex, endIndex, true
	}

	return -1, -1, false
}
