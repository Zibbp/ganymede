package utils

import (
	"math"
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
		if q == "chunked" {
			return Quality{Resolution: math.MaxInt, FPS: 60, Original: q}
		}

		if num, err := strconv.Atoi(q); err == nil {
			// Treat "720" as Resolution=720, FPS=60 (default)
			return Quality{Resolution: num, FPS: 60, Original: q}
		}
		return Quality{Original: q}
	}

	res, _ := strconv.Atoi(matches[1])
	fps := 60 // Default to 60 if not provided
	if len(matches) > 2 && matches[2] != "" {
		fps, _ = strconv.Atoi(matches[2])
	}
	return Quality{Resolution: res, FPS: fps, Original: q}
}

// SelectClosestQuality selects the best matching quality from available options.
// Behavior changes:
// - if target == "audio_only" -> return "audio_only" (if present in options).
// - when no resolution matches, pick the highest quality available (highest resolution, then highest FPS).
// - existing FPS-selection logic is preserved for matching resolutions.
func SelectClosestQuality(target string, options []string) string {
	if len(target) == 0 {
		return target
	}

	if target == "best" {
		// check if "chunked" is in options
		// this is typically the best quality for Twitch Enhanced Broadcast qualities in the HLS stream
		for _, opt := range options {
			if opt == "chunked" {
				return "chunked"
			}
		}
		return pickHighestQuality(options)
	}

	// If non-numeric target (not "best"), return directly
	if !unicode.IsDigit(rune(target[0])) {
		return target
	}

	targetQuality := parseQuality(target)

	var parsedOptions []Quality
	for _, opt := range options {
		parsedOptions = append(parsedOptions, parseQuality(opt))
	}

	// Match resolutions
	var matchingRes []Quality
	for _, opt := range parsedOptions {
		if opt.Resolution == targetQuality.Resolution {
			matchingRes = append(matchingRes, opt)
		}
	}

	if len(matchingRes) == 0 {
		// NEW: instead of returning "best", actually pick it
		return pickHighestQuality(options)
	}

	// FPS logic
	if regexp.MustCompile(`\d+p\d+`).MatchString(target) {
		for _, opt := range matchingRes {
			if opt.FPS == targetQuality.FPS {
				return opt.Original
			}
		}
		sort.Slice(matchingRes, func(i, j int) bool {
			return matchingRes[i].FPS > matchingRes[j].FPS
		})
		for _, opt := range matchingRes {
			if opt.FPS <= targetQuality.FPS {
				return opt.Original
			}
		}
		return matchingRes[len(matchingRes)-1].Original
	} else {
		sort.Slice(matchingRes, func(i, j int) bool {
			return matchingRes[i].FPS > matchingRes[j].FPS
		})
		return matchingRes[0].Original
	}
}

func pickHighestQuality(options []string) string {
	var parsed []Quality
	for _, o := range options {
		parsed = append(parsed, parseQuality(o))
	}

	// Sort by resolution DESC, FPS DESC
	sort.Slice(parsed, func(i, j int) bool {
		if parsed[i].Resolution == parsed[j].Resolution {
			return parsed[i].FPS > parsed[j].FPS
		}
		return parsed[i].Resolution > parsed[j].Resolution
	})

	return parsed[0].Original
}
