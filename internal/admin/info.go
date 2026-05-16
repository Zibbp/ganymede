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

	// Program versions
	var programVersion ProgramVersions
	ffmpegVersion, err := getFFmpegVersion()
	if err != nil {
		return resp, fmt.Errorf("error getting ffmpeg version: %v", err)
	}
	programVersion.FFmpeg = ffmpegVersion

	twitchDownloaderVersion, err := getTwitchDownloaderVersion()
	if err != nil {
		return resp, fmt.Errorf("error getting TwitchDownloaderCLI version: %v", err)
	}
	programVersion.TwitchDownloader = twitchDownloaderVersion

	ytdlpVersion, err := getYtDlpVersion()
	if err != nil {
		return resp, fmt.Errorf("error getting yt-dlp version: %v", err)
	}
	programVersion.YtDlp = ytdlpVersion

	resp.ProgramVersions = programVersion
	return resp, nil
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
	if err != nil {
		// TwitchDownloaderCLI throws exit status 1 on --version
		// so we ignore the error
		return string(out), nil
	}
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
