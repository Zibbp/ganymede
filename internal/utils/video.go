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

		// Twitch clips may expose format IDs such as "1080-0" and "720-1".
		// The prefix is the video height; the suffix distinguishes variants.
		twitchClipFormat := regexp.MustCompile(`^(\d+)-\d+$`)
		if clipMatches := twitchClipFormat.FindStringSubmatch(q); len(clipMatches) > 1 {
			res, _ := strconv.Atoi(clipMatches[1])
			return Quality{Resolution: res, FPS: 60, Original: q}
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
// Missing resolutions fall back to the nearest numeric resolution, preferring
// the lower resolution on a tie. Unknown source resolutions are a last resort.
func SelectClosestQuality(target string, options []string) string {
	if len(target) == 0 || len(options) == 0 {
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

	closestResolution := 0
	closestDistance := math.MaxInt
	for _, opt := range parsedOptions {
		// Audio, unrecognized formats, and chunked have no known video height.
		if opt.Resolution <= 0 || opt.Resolution == math.MaxInt {
			continue
		}
		distance := opt.Resolution - targetQuality.Resolution
		if distance < 0 {
			distance = -distance
		}
		if distance < closestDistance || (distance == closestDistance && opt.Resolution < closestResolution) {
			closestResolution = opt.Resolution
			closestDistance = distance
		}
	}
	if closestResolution == 0 {
		return pickHighestQuality(options)
	}

	var matchingRes []Quality
	for _, opt := range parsedOptions {
		if opt.Resolution == closestResolution {
			matchingRes = append(matchingRes, opt)
		}
	}

	// FPS logic
	if regexp.MustCompile(`\d+p\d+`).MatchString(target) {
		for _, opt := range matchingRes {
			if opt.FPS == targetQuality.FPS {
				return opt.Original
			}
		}
		sort.SliceStable(matchingRes, func(i, j int) bool {
			return matchingRes[i].FPS > matchingRes[j].FPS
		})
		for _, opt := range matchingRes {
			if opt.FPS <= targetQuality.FPS {
				return opt.Original
			}
		}
		return matchingRes[len(matchingRes)-1].Original
	} else {
		sort.SliceStable(matchingRes, func(i, j int) bool {
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
