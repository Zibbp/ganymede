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

const (
	mediaDurationAnomalyTolerance = 10.0
	mediaStartOffsetTolerance     = 0.5
)

// MediaDuration contains the useful duration values reported by ffprobe. A
// container can span hours when one stream has a bad starting timestamp even
// though every encoded stream is only a few seconds long, so callers should
// use Duration rather than FormatDuration.
type MediaDuration struct {
	Duration              float64
	FormatDuration        float64
	LongestStreamDuration float64
	MinStreamStartTime    float64
	MaxStreamStartTime    float64
	MaxStreamEndTime      float64
}

// HasTimestampAnomaly reports a shared start offset or a container timeline
// exceeding the streams' durations and start offsets. A single late stream
// may represent a genuine gap and must not trigger normalization on its own.
func (d MediaDuration) HasTimestampAnomaly() bool {
	if d.MinStreamStartTime > mediaStartOffsetTolerance {
		return true
	}
	if d.LongestStreamDuration <= 0 || d.FormatDuration <= 0 {
		return false
	}
	tolerance := math.Max(mediaDurationAnomalyTolerance, d.LongestStreamDuration*0.1)
	return d.FormatDuration > d.MaxStreamEndTime+tolerance
}

type mediaDurationProbe struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Duration  string `json:"duration"`
		StartTime string `json:"start_time"`
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
		"-show_entries", "stream=codec_type,duration,start_time:format=duration",
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

	parseStartTime := func(value string) float64 {
		startTime, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(startTime) || math.IsInf(startTime, 0) {
			return 0
		}
		return startTime
	}

	var videoDuration, audioDuration, maxStreamStartTime, maxStreamEndTime float64
	minStreamStartTime := math.Inf(1)
	for _, stream := range probe.Streams {
		duration := parseDuration(stream.Duration)
		startTime := parseStartTime(stream.StartTime)
		if stream.CodecType != "video" && stream.CodecType != "audio" {
			continue
		}
		minStreamStartTime = math.Min(minStreamStartTime, startTime)
		maxStreamStartTime = math.Max(maxStreamStartTime, startTime)
		maxStreamEndTime = math.Max(maxStreamEndTime, startTime+duration)
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
	if math.IsInf(minStreamStartTime, 1) {
		minStreamStartTime = 0
	}

	return MediaDuration{
		Duration:              streamDuration,
		FormatDuration:        formatDuration,
		LongestStreamDuration: math.Max(videoDuration, audioDuration),
		MinStreamStartTime:    minStreamStartTime,
		MaxStreamStartTime:    maxStreamStartTime,
		MaxStreamEndTime:      maxStreamEndTime,
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

type mediaStreamStart struct {
	Specifier string
	Time      float64
	Anchor    string
}

// Read one packet per stream because HLS metadata may report a segment's start
// rather than the first packet's timestamp.
func probeMediaStreamStarts(ctx context.Context, path string) ([]mediaStreamStart, error) {
	data, err := GetFfprobeVideoData(ctx, path)
	if err != nil {
		return nil, err
	}
	var starts []mediaStreamStart
	ordinals := make(map[string]int)
	for _, stream := range data.Streams {
		if stream.CodecType != "video" && stream.CodecType != "audio" {
			continue
		}
		specifier := fmt.Sprintf("%s:%d", stream.CodecType[:1], ordinals[stream.CodecType])
		ordinals[stream.CodecType]++
		cmd := osExec.CommandContext(ctx, "ffprobe", "-v", "error",
			"-select_streams", strconv.FormatInt(stream.Index, 10),
			"-read_intervals", "%+#1", "-show_entries", "packet=pts_time,dts_time",
			"-of", "json", path)
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("probe first packet for %s: %w", specifier, err)
		}
		var probe struct {
			Packets []struct {
				PTS string `json:"pts_time"`
				DTS string `json:"dts_time"`
			} `json:"packets"`
		}
		if err := json.Unmarshal(out, &probe); err != nil {
			return nil, fmt.Errorf("parse first packet for %s: %w", specifier, err)
		}
		if len(probe.Packets) == 0 {
			continue
		}
		anchor := "STARTPTS"
		start, err := strconv.ParseFloat(probe.Packets[0].PTS, 64)
		if err != nil || math.IsNaN(start) || math.IsInf(start, 0) {
			anchor = "STARTDTS"
			start, err = strconv.ParseFloat(probe.Packets[0].DTS, 64)
		}
		if err != nil || math.IsNaN(start) || math.IsInf(start, 0) {
			return nil, fmt.Errorf("no valid first packet timestamp for %s in %s", specifier, path)
		}
		starts = append(starts, mediaStreamStart{Specifier: specifier, Time: start, Anchor: anchor})
	}
	return starts, nil
}

func appendMediaTimestampArgs(args []string, starts []mediaStreamStart, normalize bool) []string {
	origin := 0.0
	if normalize && len(starts) > 0 {
		origin = starts[0].Time
		for _, start := range starts[1:] {
			origin = math.Min(origin, start.Time)
		}
	}
	for _, start := range starts {
		// Restore the source offset after FFmpeg corrects discontinuities. Only
		// the shared origin is removed; late streams retain their original gap.
		shift := "-" + start.Anchor + "+" + strconv.FormatFloat(start.Time-origin, 'f', 9, 64) + "/TB"
		filter := "setts=pts=PTS" + shift + ":dts=DTS" + shift
		var existingFilter string
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-bsf" || args[i] == "-bsf:"+start.Specifier[:1] || args[i] == "-bsf:"+start.Specifier {
				existingFilter = args[i+1]
			}
		}
		if existingFilter != "" {
			filter = existingFilter + "," + filter
		}
		args = append(args, "-bsf:"+start.Specifier, filter)
	}
	return args
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
