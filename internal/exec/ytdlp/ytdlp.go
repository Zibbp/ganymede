package ytdlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	osExec "os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
)

type YtDlpOptions struct {
	Cookies []YtDlpCookie
}

type YtDlpService struct {
	Options YtDlpOptions
}

type YTDLPVideoInfo struct {
	ID                     string      `json:"id"`
	Title                  string      `json:"title"`
	Description            interface{} `json:"description"`
	Duration               int64       `json:"duration"`
	Thumbnails             []Thumbnail `json:"thumbnails"`
	Uploader               string      `json:"uploader"`
	UploaderID             string      `json:"uploader_id"`
	Timestamp              int64       `json:"timestamp"`
	ViewCount              int64       `json:"view_count"`
	Chapters               []Chapter   `json:"chapters"`
	IsLive                 bool        `json:"is_live"`
	WasLive                bool        `json:"was_live"`
	Formats                []Format    `json:"formats"`
	Subtitles              Subtitles   `json:"subtitles"`
	WebpageURL             string      `json:"webpage_url"`
	OriginalURL            string      `json:"original_url"`
	WebpageURLBasename     string      `json:"webpage_url_basename"`
	WebpageURLDomain       string      `json:"webpage_url_domain"`
	Extractor              string      `json:"extractor"`
	ExtractorKey           string      `json:"extractor_key"`
	Playlist               interface{} `json:"playlist"`
	PlaylistIndex          interface{} `json:"playlist_index"`
	Thumbnail              string      `json:"thumbnail"`
	DisplayID              string      `json:"display_id"`
	Fulltitle              string      `json:"fulltitle"`
	DurationString         string      `json:"duration_string"`
	UploadDate             string      `json:"upload_date"`
	ReleaseYear            interface{} `json:"release_year"`
	LiveStatus             string      `json:"live_status"`
	RequestedSubtitles     interface{} `json:"requested_subtitles"`
	HasDRM                 interface{} `json:"_has_drm"`
	Epoch                  int64       `json:"epoch"`
	FormatID               string      `json:"format_id"`
	FormatIndex            interface{} `json:"format_index"`
	URL                    string      `json:"url"`
	ManifestURL            string      `json:"manifest_url"`
	Tbr                    float64     `json:"tbr"`
	EXT                    string      `json:"ext"`
	FPS                    float64     `json:"fps"`
	Protocol               string      `json:"protocol"`
	Preference             interface{} `json:"preference"`
	Quality                int64       `json:"quality"`
	YTDLPVideoInfoHasDRM   bool        `json:"has_drm"`
	Width                  int64       `json:"width"`
	Height                 int64       `json:"height"`
	Vcodec                 string      `json:"vcodec"`
	Acodec                 string      `json:"acodec"`
	DynamicRange           string      `json:"dynamic_range"`
	FormatNote             string      `json:"format_note"`
	VideoEXT               string      `json:"video_ext"`
	AudioEXT               string      `json:"audio_ext"`
	Vbr                    *float64    `json:"vbr"`
	ABR                    *float64    `json:"abr"`
	Resolution             string      `json:"resolution"`
	AspectRatio            float64     `json:"aspect_ratio"`
	HTTPHeaders            HTTPHeaders `json:"http_headers"`
	Format                 string      `json:"format"`
	Filename               string      `json:"_filename"`
	YTDLPVideoInfoFilename string      `json:"filename"`
	Type                   string      `json:"_type"`
	Version                Version     `json:"_version"`
}

type Chapter struct {
	Title     string `json:"title"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
}

type Format struct {
	FormatID       string      `json:"format_id"`
	FormatNote     *string     `json:"format_note,omitempty"`
	EXT            string      `json:"ext"`
	Protocol       string      `json:"protocol"`
	Acodec         string      `json:"acodec"`
	Vcodec         string      `json:"vcodec"`
	URL            string      `json:"url"`
	Width          *int64      `json:"width,omitempty"`
	Height         *int64      `json:"height,omitempty"`
	FPS            *float64    `json:"fps"`
	Rows           *int64      `json:"rows,omitempty"`
	Columns        *int64      `json:"columns,omitempty"`
	Fragments      []Fragment  `json:"fragments,omitempty"`
	AudioEXT       string      `json:"audio_ext"`
	VideoEXT       string      `json:"video_ext"`
	Vbr            *float64    `json:"vbr"`
	ABR            *float64    `json:"abr"`
	Tbr            *float64    `json:"tbr"`
	Resolution     string      `json:"resolution"`
	AspectRatio    *float64    `json:"aspect_ratio"`
	FilesizeApprox interface{} `json:"filesize_approx"`
	HTTPHeaders    HTTPHeaders `json:"http_headers"`
	Format         string      `json:"format"`
	FormatIndex    interface{} `json:"format_index"`
	ManifestURL    *string     `json:"manifest_url,omitempty"`
	Preference     interface{} `json:"preference"`
	Quality        *int64      `json:"quality"`
	HasDRM         *bool       `json:"has_drm,omitempty"`
	DynamicRange   *string     `json:"dynamic_range"`
}

type Fragment struct {
	URL      string  `json:"url"`
	Duration float64 `json:"duration"`
}

type HTTPHeaders struct {
	UserAgent      string `json:"User-Agent"`
	Accept         string `json:"Accept"`
	AcceptLanguage string `json:"Accept-Language"`
	SECFetchMode   string `json:"Sec-Fetch-Mode"`
}

type Subtitles struct {
	Rechat []Rechat `json:"rechat"`
}

type Rechat struct {
	URL string `json:"url"`
	EXT string `json:"ext"`
}

type Thumbnail struct {
	URL        string `json:"url"`
	ID         string `json:"id"`
	Preference *int64 `json:"preference,omitempty"`
}

type Version struct {
	Version        string      `json:"version"`
	CurrentGitHead interface{} `json:"current_git_head"`
	ReleaseGitHead string      `json:"release_git_head"`
	Repository     string      `json:"repository"`
}

type YtDlpCookie struct {
	Domain string `json:"domain"`
	Name   string `json:"name"`
	Value  string `json:"value"`
}

func NewYtDlpService(options YtDlpOptions) *YtDlpService {
	return &YtDlpService{
		Options: options,
	}
}

// YtDlpGetVideoInfo retrieves video information using yt-dlp for a given video entity.
func (s *YtDlpService) GetVideoInfo(ctx context.Context, video ent.Vod) (*YTDLPVideoInfo, error) {
	url := utils.CreateTwitchURL(video.ExtID, video.Type, video.Edges.Channel.Name)

	args := []string{"-q", "-j", url}
	log.Info().Msgf("running yt-dlp with args: %s", strings.Join(args, " "))

	cmd, cookieFile, err := s.CreateCommand(ctx, args, true)
	defer func() {
		if cookieFile != nil {
			if err := cookieFile.Close(); err != nil {
				log.Debug().Err(err).Msg("failed to close cookies file")
			}
			if err := os.Remove(cookieFile.Name()); err != nil {
				log.Debug().Err(err).Msg("failed to remove cookies file")
			}
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("error creating yt-dlp command: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Str("stderr", stderr.String()).Str("stdout", stdout.String()).Msg("error running yt-dlp")
		return nil, fmt.Errorf("error running yt-dlp: %w", err)
	}
	var videoInfo YTDLPVideoInfo
	if err := json.Unmarshal(stdout.Bytes(), &videoInfo); err != nil {
		log.Error().Err(err).Msg("error unmarshalling yt-dlp data")
		return nil, fmt.Errorf("error unmarshalling yt-dlp data: %w", err)
	}

	return &videoInfo, nil
}

// GetVideoQualities retrieves the available video qualities for a given video entity using yt-dlp.
// This returns the raw format IDs as qualities.
//
// Example: [1080p60 1440p60__source_ 360p30 480p30 720p60 audio_only]
func (s *YtDlpService) GetVideoQualities(ctx context.Context, video ent.Vod) ([]string, error) {
	info, err := s.GetVideoInfo(ctx, video)
	if err != nil {
		log.Error().Err(err).Msg("error getting video info")
		return nil, fmt.Errorf("error getting video info: %w", err)
	}

	// Check if the video has formats
	if len(info.Formats) == 0 {
		log.Error().Msg("video has no formats")
		return nil, fmt.Errorf("video has no formats")
	}

	// Extract unique qualities from the formats
	qualities := make(map[string]struct{})
	for _, format := range info.Formats {
		if (format.Vbr != nil && *format.Vbr == 0) && (format.ABR != nil && *format.ABR == 0) {
			// Skip formats without bitrate information
			continue
		}
		if format.FormatID == "" {
			// Skip formats without a format ID
			continue
		}
		// use formatID as quality
		qualities[format.FormatID] = struct{}{}
	}

	// Convert map keys to a slice
	qualityList := make([]string, 0, len(qualities))
	for quality := range qualities {
		qualityList = append(qualityList, quality)
	}

	// Sort the qualities slice
	sort.Strings(qualityList)
	log.Info().Str("video_id", video.ID.String()).Msgf("available qualities: %v", qualityList)
	return qualityList, nil
}

// createYtDlpCommand creates a yt-dlp command with the provided input arguments and cookies.
// It returns the command, a file handle for the cookies file, and any error encountered.
func (s *YtDlpService) CreateCommand(ctx context.Context, inputArgs []string, enableCookies bool) (*osExec.Cmd, *os.File, error) {
	args := []string{
		"--force-overwrites",
		"--external-downloader-args",
		"-loglevel warning -stats",
	}
	args = append(args, inputArgs...)

	var cookiesFile *os.File

	if len(s.Options.Cookies) > 0 && enableCookies {
		var err error
		cookiesFile, err = createYtDlpCookiesFile(ctx, s.Options.Cookies)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create cookies file: %w", err)
		}
		args = append(args, "--cookies", cookiesFile.Name())
	}

	cmd := osExec.CommandContext(ctx, "yt-dlp", args...)

	return cmd, cookiesFile, nil
}

// createYtDlpTwitchCookies creates a yt-dlp cookies file with the provided cookies.
// Returns the file handle for the cookies file.
func createYtDlpCookiesFile(ctx context.Context, cookies []YtDlpCookie) (*os.File, error) {
	expiration := time.Now().Add(7 * 24 * time.Hour).Unix()
	cookieStr := "# Netscape HTTP Cookie File\n# This file is generated by yt-dlp.  Do not edit.\n\n"
	for _, c := range cookies {
		cookieStr += fmt.Sprintf("%s\tTRUE\t/\tTRUE\t%d\t%s\t%s\n", c.Domain, expiration, c.Name, c.Value)
	}
	cookiesFile, err := os.CreateTemp("", "cookies-*.txt")
	if err != nil {
		return nil, err
	}
	_, err = cookiesFile.WriteString(cookieStr)
	if err != nil {
		closeErr := cookiesFile.Close()
		if closeErr != nil {
			log.Error().Err(closeErr).Msg("failed to close cookies file after write error")
		}
		return nil, err
	}
	log.Debug().Msgf("created yt-dlp cookies file: %s", cookiesFile.Name())
	return cookiesFile, nil
}

// CreateQualityOption creates a yt-dlp format string for Twitch content,
// which uses combined audio+video streams even for VODs.
// This only supports 'single stream' format and not split video/audio formats.
func (s *YtDlpService) CreateQualityOption(quality string) string {
	// Strip odd yt-dlp quality suffixes like "__source_"
	reSuffix := regexp.MustCompile(`__source_+$`)
	quality = reSuffix.ReplaceAllString(quality, "")

	switch quality {
	case "best":
		return "best"
	case "audio", "audio_only":
		return "bestaudio"
	}

	// Handle exact resolutions
	// Always fallback to "best" if for some reason the quality is not recognized

	// Match resolution from formats like "1080p60" or "1080p"
	re := regexp.MustCompile(`^(\d+)[pP]`)
	if matches := re.FindStringSubmatch(quality); len(matches) > 1 {
		res := matches[1]
		return fmt.Sprintf("best[height=%s]/best", res)
	}

	// Match pure resolution like "1080"
	reRes := regexp.MustCompile(`^\d+$`)
	if reRes.MatchString(quality) {
		return fmt.Sprintf("best[height=%s]/best", quality)
	}

	// Fallback: match up to resolution
	return fmt.Sprintf("best[height<=?%s]/best", quality)
}
