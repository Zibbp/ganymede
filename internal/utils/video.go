package utils

import (
	"regexp"
	"sort"
	"strconv"
	"unicode"
)

// Quality represents a video quality option with resolution, FPS, and original string representation.
type Quality struct {
	Resolution int
	FPS        int
	Original   string
}

// parseQuality extracts resolution and FPS from a quality string (e.g., "1080p60").
// If no FPS is provided, it defaults to 60.
func parseQuality(q string) Quality {
	re := regexp.MustCompile(`(\d+)p(\d+)?`)
	matches := re.FindStringSubmatch(q)

	// If it doesn’t look like "123p" or "123p45", see if it's just digits:
	if len(matches) == 0 {
		if num, err := strconv.Atoi(q); err == nil {
			// Treat "720" as Resolution=720, FPS=0
			return Quality{Resolution: num, FPS: 0, Original: q}
		}
		return Quality{Original: q}
	}

	// (all your existing regex‐based parsing unchanged)
	res, _ := strconv.Atoi(matches[1])
	fps := 0
	if len(matches) > 2 && matches[2] != "" {
		fps, _ = strconv.Atoi(matches[2])
	}
	return Quality{Resolution: res, FPS: fps, Original: q}
}

// SelectClosestQuality selects the best matching quality from available options.
// Streams and video can vary quite a bit in quality, so it's important to select the closest match.
// If an exact match is found, it returns that.
// If an exact FPS match isn't found, it selects the closest lower FPS.
// If no matching resolution is found, it falls back to "best".
// If the target is a non-numeric string, it returns the input directly.
func SelectClosestQuality(target string, options []string) string {
	if len(target) == 0 || !unicode.IsDigit(rune(target[0])) {
		return target // Return if non-numeric string
	}

	targetQuality := parseQuality(target)

	// Parse all options
	var parsedOptions []Quality
	for _, opt := range options {
		parsedOptions = append(parsedOptions, parseQuality(opt))
	}

	// Filter options that match the target resolution
	var matchingRes []Quality
	for _, opt := range parsedOptions {
		if opt.Resolution == targetQuality.Resolution {
			matchingRes = append(matchingRes, opt)
		}
	}

	if len(matchingRes) == 0 {
		return "best" // Fallback if no resolution matches
	}

	// If target specifies FPS (e.g., "720p60")
	if regexp.MustCompile(`\d+p\d+`).MatchString(target) {
		// Look for an exact FPS match
		for _, opt := range matchingRes {
			if opt.FPS == targetQuality.FPS {
				return opt.Original
			}
		}
		// No exact match, find closest lower FPS or lowest if all higher
		sort.Slice(matchingRes, func(i, j int) bool {
			return matchingRes[i].FPS > matchingRes[j].FPS
		})
		for _, opt := range matchingRes {
			if opt.FPS <= targetQuality.FPS {
				return opt.Original
			}
		}
		return matchingRes[len(matchingRes)-1].Original // Lowest FPS if all higher
	} else {
		// Target has no FPS (e.g., "720p"), pick highest FPS available
		sort.Slice(matchingRes, func(i, j int) bool {
			return matchingRes[i].FPS > matchingRes[j].FPS
		})
		return matchingRes[0].Original // Highest FPS option
	}
}
