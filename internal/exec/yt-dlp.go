package exec

import (
	"context"
	"fmt"
	"os"
	osExec "os/exec"
	"regexp"
	"time"

	"github.com/zibbp/ganymede/internal/config"
)

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
	Vbr            *int64      `json:"vbr"`
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

// createYtDlpCommand creates a yt-dlp command with the provided input arguments.
func createYtDlpCommand(ctx context.Context, inputArgs []string) (*osExec.Cmd, error) {
	args := []string{}
	// Add the input arguments to the command
	args = append(args, inputArgs...)

	jsonConfig := config.Get()
	envConfig := config.GetEnvConfig()

	// Check if we need to create a cookies file if token is set in config
	if jsonConfig.Parameters.TwitchToken != "" {
		cookiesFile, err := createYtDlpTwitchCookies(ctx, envConfig.TempDir, jsonConfig.Parameters.TwitchToken)
		if err != nil {
			return nil, fmt.Errorf("failed to create cookies file: %w", err)
		}
		// Add the cookies file to the arguments
		args = append(args, "--cookies", cookiesFile.Name())
	}

	// Create the yt-dlp command
	cmd := osExec.CommandContext(ctx, "yt-dlp", args...)

	return cmd, nil
}

func createYtDlpTwitchCookies(ctx context.Context, tempDirectory string, token string) (*os.File, error) {
	expiration := time.Now().Add(7 * 24 * time.Hour).Unix()

	// Create the cookie string in Netscape format
	cookie := fmt.Sprintf(`# Netscape HTTP Cookie File
# This file is generated by yt-dlp.  Do not edit.

.twitch.tv	TRUE	/	TRUE	%d	auth-token	%s
`, expiration, token)

	cookiesFile, err := os.CreateTemp(tempDirectory, "cookies-*.txt")
	if err != nil {
		return nil, err
	}

	defer cookiesFile.Close()

	_, err = cookiesFile.WriteString(cookie)
	if err != nil {
		return nil, err
	}

	return cookiesFile, nil
}

// createQualityOption creates a yt-dlp quality option string based on the provided quality input.
func createQualityOption(quality string) string {
	if quality == "best" {
		return "bestvideo+bestaudio/best"
	}
	if quality == "audio" {
		return "bestaudio"
	}
	// Check for resolution+fps (e.g., 720p60)
	re := regexp.MustCompile(`^(\d+)[pP](\d+)$`)
	if matches := re.FindStringSubmatch(quality); len(matches) == 3 {
		res := matches[1]
		fps := matches[2]
		return fmt.Sprintf("bestvideo[height=%s][fps=%s]+bestaudio/best", res, fps)
	}
	// Pure resolution (e.g., 720)
	reRes := regexp.MustCompile(`^\d+$`)
	if reRes.MatchString(quality) {
		return fmt.Sprintf("bestvideo[height=%s]+bestaudio/best", quality)
	}
	// fallback: less than or equal to resolution
	return fmt.Sprintf("bestvideo[height<=?%s]+bestaudio/best[height<=?%s]", quality, quality)
}
