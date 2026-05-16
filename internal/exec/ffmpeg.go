package exec

import (
	"context"
	"encoding/json"
	"fmt"
	osExec "os/exec"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// GetVideoDuration runs ffprobe on the given video file and returns its duration in seconds.
func GetVideoDuration(ctx context.Context, path string) (int, error) {
	cmd := osExec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)

	log.Debug().Msgf("Running ffprobe command: %s", strings.Join(cmd.Args, " "))

	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("error running ffprobe: %w", err)
	}
	durationOut := strings.TrimSpace(string(out))

	duration, err := strconv.ParseFloat(durationOut, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing duration: %w", err)
	}
	return int(duration), nil
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
