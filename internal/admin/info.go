package admin

import (
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/kv"
)

type InfoResp struct {
	Version         string `json:"version"`
	BuildTime       string `json:"build_time"`
	GitHash         string `json:"git_hash"`
	Uptime          string `json:"uptime"`
	ProgramVersions `json:"program_versions"`
}

type ProgramVersions struct {
	FFmpeg           string `json:"ffmpeg"`
	TwitchDownloader string `json:"twitch_downloader"`
	ChatDownloader   string `json:"chat_downloader"`
	Streamlink       string `json:"streamlink"`
}

func (s *Service) GetInfo(c echo.Context) (InfoResp, error) {
	var resp InfoResp
	resp.Version = kv.DB().Get("version")
	resp.BuildTime = kv.DB().Get("build_time")
	resp.GitHash = kv.DB().Get("git_hash")
	startTimeUnix := kv.DB().Get("start_time_unix")
	parsedStart, err := strconv.ParseInt(startTimeUnix, 10, 64)
	if err != nil {
		return resp, fmt.Errorf("error parsing start time: %v", err)
	}
	resp.Uptime = time.Since(time.Unix(parsedStart, 0)).String()

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

	chatDownloaderVersion, err := getChatDownloaderVersion()
	if err != nil {
		return resp, fmt.Errorf("error getting chat_downloader version: %v", err)
	}
	programVersion.ChatDownloader = chatDownloaderVersion

	streamlinkVersion, err := getStreamlinkVersion()
	if err != nil {
		return resp, fmt.Errorf("error getting streamlink version: %v", err)
	}
	programVersion.Streamlink = streamlinkVersion

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

func getChatDownloaderVersion() (string, error) {
	run := exec.Command("chat_downloader", "--version")
	out, err := run.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error getting chat_downloader version: %v", err)
	}
	return string(out), nil
}

func getStreamlinkVersion() (string, error) {
	run := exec.Command("streamlink", "--version")
	out, err := run.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error getting streamlink version: %v", err)
	}
	return string(out), nil
}
