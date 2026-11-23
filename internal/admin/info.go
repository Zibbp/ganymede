package admin

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/zibbp/ganymede/internal/utils"
)

type InfoResp struct {
	CommitHash      string `json:"commit_hash"`
	Tag             string `json:"tag"`
	BuildTime       string `json:"build_time"`
	Uptime          string `json:"uptime"`
	ProgramVersions `json:"program_versions"`
}

type ProgramVersions struct {
	FFmpeg           string `json:"ffmpeg"`
	TwitchDownloader string `json:"twitch_downloader"`
	YtDlp            string `json:"yt_dlp"`
}

func (s *Service) GetInfo(ctx context.Context) (InfoResp, error) {
	var resp InfoResp
	resp.CommitHash = utils.Commit
	resp.Tag = utils.Tag
	resp.BuildTime = utils.BuildTime
	resp.Uptime = time.Since(utils.StartTime).String()

	// Report missing tools without failing the whole response.
	var programVersion ProgramVersions
	programVersion.FFmpeg = versionOrUnavailable(getFFmpegVersion)
	programVersion.TwitchDownloader = versionOrUnavailable(getTwitchDownloaderVersion)
	programVersion.YtDlp = versionOrUnavailable(getYtDlpVersion)

	resp.ProgramVersions = programVersion
	return resp, nil
}

// versionOrUnavailable returns the tool version, or "unavailable" if it is missing.
func versionOrUnavailable(get func() (string, error)) string {
	out, err := get()
	if err != nil {
		return "unavailable"
	}
	return out
}

func getFFmpegVersion() (string, error) {
	run := exec.Command("ffmpeg", "-version")
	out, err := run.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error getting ffmpeg version: %v", err)
	}
	// Get only the version
	return string(out), nil
}

func getTwitchDownloaderVersion() (string, error) {
	run := exec.Command("TwitchDownloaderCLI", "--version")
	out, err := run.CombinedOutput()
	if err != nil && len(out) == 0 {
		return "", fmt.Errorf("error getting TwitchDownloaderCLI version: %v", err)
	}
	// TwitchDownloaderCLI exits 1 for --version, so accept its output. A missing
	// binary prints nothing and is reported as unavailable.
	return string(out), nil
}

func getYtDlpVersion() (string, error) {
	run := exec.Command("yt-dlp", "--version")
	out, err := run.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error getting yt-dlp version: %v", err)
	}
	return string(out), nil
}
