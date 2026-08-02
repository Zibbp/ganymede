package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	osExec "os/exec"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

const mediaDurationAnomalyTolerance = 10.0

// MediaDuration contains the useful duration values reported by ffprobe. A
// container can span hours when one stream has a bad starting timestamp even
// though every encoded stream is only a few seconds long, so callers should
// use Duration rather than FormatDuration.
type MediaDuration struct {
	Duration              float64
	FormatDuration        float64
	LongestStreamDuration float64
}

// HasTimestampAnomaly reports whether the container timeline is substantially
// longer than its encoded streams. Small differences are normal because audio
// and video do not necessarily end on the same timestamp.
func (d MediaDuration) HasTimestampAnomaly() bool {
	if d.LongestStreamDuration <= 0 || d.FormatDuration <= 0 {
		return false
	}
	tolerance := math.Max(mediaDurationAnomalyTolerance, d.LongestStreamDuration*0.1)
	return d.FormatDuration > d.LongestStreamDuration+tolerance
}

type mediaDurationProbe struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Duration  string `json:"duration"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// ProbeMediaDuration reads both stream and container durations. Video stream
// duration is preferred, with audio and finally the container as fallbacks.
func ProbeMediaDuration(ctx context.Context, path string) (MediaDuration, error) {
	cmd := osExec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "stream=codec_type,duration:format=duration",
		"-of", "json",
		path,
	)

	log.Debug().Msgf("Running ffprobe command: %s", strings.Join(cmd.Args, " "))

	out, err := cmd.Output()
	if err != nil {
		return MediaDuration{}, fmt.Errorf("error running ffprobe: %w", err)
	}

	var probe mediaDurationProbe
	if err := json.Unmarshal(out, &probe); err != nil {
		return MediaDuration{}, fmt.Errorf("error parsing ffprobe output: %w", err)
	}

	parseDuration := func(value string) float64 {
		duration, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(duration) || math.IsInf(duration, 0) || duration <= 0 {
			return 0
		}
		return duration
	}

	var videoDuration, audioDuration float64
	for _, stream := range probe.Streams {
		duration := parseDuration(stream.Duration)
		switch stream.CodecType {
		case "video":
			videoDuration = math.Max(videoDuration, duration)
		case "audio":
			audioDuration = math.Max(audioDuration, duration)
		}
	}

	formatDuration := parseDuration(probe.Format.Duration)
	streamDuration := videoDuration
	if streamDuration == 0 {
		streamDuration = audioDuration
	}
	if streamDuration == 0 {
		streamDuration = formatDuration
	}
	if streamDuration == 0 {
		return MediaDuration{}, fmt.Errorf("ffprobe returned no valid duration for %s", path)
	}

	return MediaDuration{
		Duration:              streamDuration,
		FormatDuration:        formatDuration,
		LongestStreamDuration: math.Max(videoDuration, audioDuration),
	}, nil
}

// GetVideoDuration runs ffprobe on the given video file and returns its
// validated media-stream duration in seconds.
func GetVideoDuration(ctx context.Context, path string) (int, error) {
	duration, err := ProbeMediaDuration(ctx, path)
	if err != nil {
		return 0, err
	}
	return max(1, int(math.Ceil(duration.Duration))), nil
}

// GetFfprobeData runs ffprobe on the given path and returns parsed JSON output.
func GetFfprobeData(ctx context.Context, path string) (map[string]interface{}, error) {
	cmd := osExec.CommandContext(ctx, "ffprobe",
		"-hide_banner", "-v", "quiet",
		"-print_format", "json",
		"-show_format", "-show_streams", path,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed for %s: %w", path, err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ffprobe output: %w", err)
	}
	return data, nil
}

type FFprobeJsonData struct {
	Streams []FFprobestream `json:"streams"`
	Format  FFprobeFormat   `json:"format"`
}

type FFprobeFormat struct {
	Filename       string `json:"filename"`
	NbStreams      int64  `json:"nb_streams"`
	NbPrograms     int64  `json:"nb_programs"`
	FormatName     string `json:"format_name"`
	FormatLongName string `json:"format_long_name"`
	StartTime      string `json:"start_time"`
	Duration       string `json:"duration"`
	Size           string `json:"size"`
	BitRate        string `json:"bit_rate"`
	ProbeScore     int64  `json:"probe_score"`
}

type FFprobestream struct {
	Index              int64            `json:"index"`
	CodecName          string           `json:"codec_name"`
	CodecLongName      string           `json:"codec_long_name"`
	Profile            string           `json:"profile"`
	CodecType          string           `json:"codec_type"`
	CodecTagString     string           `json:"codec_tag_string"`
	CodecTag           string           `json:"codec_tag"`
	Width              *int64           `json:"width,omitempty"`
	Height             *int64           `json:"height,omitempty"`
	CodedWidth         *int64           `json:"coded_width,omitempty"`
	CodedHeight        *int64           `json:"coded_height,omitempty"`
	ClosedCaptions     *int64           `json:"closed_captions,omitempty"`
	FilmGrain          *int64           `json:"film_grain,omitempty"`
	HasBFrames         *int64           `json:"has_b_frames,omitempty"`
	SampleAspectRatio  *string          `json:"sample_aspect_ratio,omitempty"`
	DisplayAspectRatio *string          `json:"display_aspect_ratio,omitempty"`
	PixFmt             *string          `json:"pix_fmt,omitempty"`
	Level              *int64           `json:"level,omitempty"`
	ColorRange         *string          `json:"color_range,omitempty"`
	ColorSpace         *string          `json:"color_space,omitempty"`
	ColorTransfer      *string          `json:"color_transfer,omitempty"`
	ColorPrimaries     *string          `json:"color_primaries,omitempty"`
	ChromaLocation     *string          `json:"chroma_location,omitempty"`
	Refs               *int64           `json:"refs,omitempty"`
	ID                 string           `json:"id"`
	RFrameRate         string           `json:"r_frame_rate"`
	AvgFrameRate       string           `json:"avg_frame_rate"`
	TimeBase           string           `json:"time_base"`
	StartPts           int64            `json:"start_pts"`
	StartTime          string           `json:"start_time"`
	DurationTs         int64            `json:"duration_ts"`
	Duration           string           `json:"duration"`
	ExtradataSize      *int64           `json:"extradata_size,omitempty"`
	Disposition        map[string]int64 `json:"disposition"`
	SampleFmt          *string          `json:"sample_fmt,omitempty"`
	SampleRate         *string          `json:"sample_rate,omitempty"`
	Channels           *int64           `json:"channels,omitempty"`
	ChannelLayout      *string          `json:"channel_layout,omitempty"`
	BitsPerSample      *int64           `json:"bits_per_sample,omitempty"`
	BitRate            *string          `json:"bit_rate,omitempty"`
}

// GetFfprobeVideoData runs ffprobe on the given video file and returns structured JSON data.
func GetFfprobeVideoData(ctx context.Context, path string) (*FFprobeJsonData, error) {
	cmd := osExec.CommandContext(ctx, "ffprobe",
		"-v", "quiet", "-show_format", "-show_streams",
		"-print_format", "json", path,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed for %s: %w", path, err)
	}
	var data FFprobeJsonData
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ffprobe output: %w", err)
	}
	if len(data.Streams) == 0 {
		return nil, fmt.Errorf("no streams found in ffprobe output for %s", path)
	}
	if data.Format.Filename == "" {
		return nil, fmt.Errorf("no filename found in ffprobe output for %s", path)
	}

	return &data, nil
}
