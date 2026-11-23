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

	// Program versions. A host-native API may not have every media tool
	// installed, so report the missing one instead of failing the whole page.
	var programVersion ProgramVersions
	programVersion.FFmpeg = versionOrUnavailable(getFFmpegVersion)
	programVersion.TwitchDownloader = versionOrUnavailable(getTwitchDownloaderVersion)
	programVersion.YtDlp = versionOrUnavailable(getYtDlpVersion)

	resp.ProgramVersions = programVersion
	return resp, nil
}

// versionOrUnavailable runs a version command and reports a missing tool
// without failing the rest of the response.
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
	// TwitchDownloaderCLI throws exit status 1 on --version, so accept output
	// from a non-zero exit. Missing binaries produce no output and are reported.
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
