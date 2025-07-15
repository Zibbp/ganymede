package exec

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/errors"
	"github.com/zibbp/ganymede/internal/exec/ytdlp"
	"github.com/zibbp/ganymede/internal/utils"
)

// DownloadTwitchVideo downloads a Twitch video.
func DownloadTwitchVideo(ctx context.Context, video ent.Vod) error {
	// Get video channel
	videoChannel := video.QueryChannel()
	channel, err := videoChannel.Only(ctx)
	if err != nil {
		return err
	}
	video.Edges.Channel = channel

	env := config.GetEnvConfig()

	// Open download log file
	logFilePath := fmt.Sprintf("%s/%s-video.log", env.LogsDir, video.ID.String())
	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Debug().Err(err).Msg("failed to close log file")
		}
	}()
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging output to %s", logFilePath)

	// Create the Twitch URL based on video type
	url := utils.CreateTwitchURL(video.ExtID, video.Type, video.Edges.Channel.Name)

	// Create yt-dlp service
	ytDlpCookies := []ytdlp.YtDlpCookie{}
	if config.Get().Parameters.TwitchToken != "" {
		ytDlpCookies = append(ytDlpCookies, ytdlp.YtDlpCookie{
			Domain: ".twitch.tv",
			Name:   "auth-token",
			Value:  config.Get().Parameters.TwitchToken,
		})
	}
	ytdlpSvc := ytdlp.NewYtDlpService(ytdlp.YtDlpOptions{Cookies: ytDlpCookies})

	// Select the closest quality for the video
	closestQuality := video.Resolution
	qualities, err := ytdlpSvc.GetVideoQualities(ctx, video)
	if err != nil {
		return fmt.Errorf("error getting video quality options: %w", err)
	}

	closestQuality = utils.SelectClosestQuality(video.Resolution, qualities)
	log.Info().Msgf("selected closest quality %s", closestQuality)

	// Create yt-dlp quality string
	qualityString := ytdlpSvc.CreateQualityOption(closestQuality)

	// Build output path
	// yt-dlp will sometimes download two separate files for audio and video
	// so we need to remove the extension and let yt-dlp add the extension
	tmpVideoDownloadExt := filepath.Ext(video.TmpVideoDownloadPath)
	tmpVideoDownloadPathNoExt := strings.TrimSuffix(video.TmpVideoDownloadPath, tmpVideoDownloadExt)

	// Get user arguments from config
	configYtDlpArgs := config.Get().Parameters.YtDlpVideo
	configYtDlpArgsArr := strings.Split(configYtDlpArgs, ",")

	var cmdArgs []string
	cmdArgs = append(cmdArgs,
		"-f", qualityString,
		url,
		"-o", fmt.Sprintf("%s.%%(ext)s", tmpVideoDownloadPathNoExt),
		"--merge-output-format", "mp4", "--no-part",
		"--no-warnings", "--progress", "--newline", "--no-check-certificate",
	)

	// Sanitize config args before appending
	for _, arg := range configYtDlpArgsArr {
		if strings.TrimSpace(arg) != "" {
			cmdArgs = append(cmdArgs, arg)
		}
	}

	// Create yt-dlp command
	cmd, cookieFile, err := ytdlpSvc.CreateCommand(ctx, cmdArgs)
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
		return fmt.Errorf("error creating yt-dlp command: %w", err)
	}

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(cmd.Args, " ")).Msgf("running yt-dlp")

	cmd.Stderr = file
	cmd.Stdout = file

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Set the process group ID to allow killing child processes
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting yt-dlp: %w", err)
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for the command to finish or context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled, kill the process
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill yt-dlp process: %v", err)
		}
		<-done // Wait for copying to finish
		return ctx.Err()
	case err := <-done:
		// Command finished normally
		if err != nil {
			if exitError, ok := err.(*osExec.ExitError); ok {
				log.Error().Err(err).Str("exitCode", strconv.Itoa(exitError.ExitCode())).Str("exit_error", exitError.Error()).Msg("error running yt-dlp")
				return fmt.Errorf("error running yt-dlp")
			}
			return fmt.Errorf("error running yt-dlp: %w", err)
		}
	}

	return nil
}

func DownloadTwitchLiveVideo(ctx context.Context, video ent.Vod, channel ent.Channel, startChat chan bool) error {
	video.Edges.Channel = &channel
	env := config.GetEnvConfig()

	// open video log file
	logFilePath := fmt.Sprintf("%s/%s-video.log", env.LogsDir, video.ID.String())
	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Debug().Err(err).Msg("failed to close log file")
		}
	}()

	log.Debug().Str("video_id", video.ID.String()).Msgf("logging yt-dlp output to %s", logFilePath)

	// Get user arguments from config
	configYtDlpArgs := config.Get().Parameters.YtDlpLive
	configYtDlpArgsArr := strings.Split(configYtDlpArgs, ",")

	// proxyEnabled := false
	// proxyFound := false
	// proxyHeader := ""

	url := utils.CreateTwitchURL(video.ExtID, video.Type, channel.Name)

	// TODO: setup with yt-dlp
	// check if user has proxies enable
	// proxyEnabled = config.Get().Livestream.ProxyEnabled
	// whitelistedChannels := config.Get().Livestream.ProxyWhitelist // list of channels that are whitelisted from using proxy
	// if proxyEnabled {
	// 	if utils.Contains(whitelistedChannels, channel.Name) {
	// 		log.Debug().Str("channel_name", channel.Name).Msg("channel is whitelisted, not using proxy")
	// 	} else {
	// 		proxyParams := config.Get().Livestream.ProxyParameters
	// 		proxyList := config.Get().Livestream.Proxies

	// 		log.Debug().Str("proxy_list", fmt.Sprintf("%v", proxyList)).Msg("proxy list")
	// 		// test proxies
	// 		for _, proxy := range proxyList {
	// 			proxyUrl := fmt.Sprintf("%s/playlist/%s.m3u8%s", proxy.URL, channel.Name, proxyParams)
	// 			if testProxyServer(proxyUrl, proxy.Header) {
	// 				log.Debug().Str("channel_name", channel.Name).Str("proxy_url", proxy.URL).Msg("proxy found")
	// 				proxyFound = true
	// 				url = fmt.Sprintf("hls://%s", proxyUrl)
	// 				proxyHeader = proxy.Header
	// 				break
	// 			}
	// 		}
	// 	}
	// }

	// Create yt-dlp service
	ytDlpCookies := []ytdlp.YtDlpCookie{}
	if config.Get().Parameters.TwitchToken != "" {
		ytDlpCookies = append(ytDlpCookies, ytdlp.YtDlpCookie{
			Domain: ".twitch.tv",
			Name:   "auth-token",
			Value:  config.Get().Parameters.TwitchToken,
		})
	}
	ytdlpSvc := ytdlp.NewYtDlpService(ytdlp.YtDlpOptions{Cookies: ytDlpCookies})

	// If not best or audio, get the closest quality for the video
	closestQuality := video.Resolution
	qualities, err := ytdlpSvc.GetVideoQualities(ctx, video)
	if err != nil {
		return fmt.Errorf("error getting video quality options: %w", err)
	}

	closestQuality = utils.SelectClosestQuality(video.Resolution, qualities)
	log.Info().Msgf("selected closest quality %s", closestQuality)

	// Create yt-dlp quality string
	qualityString := ytdlpSvc.CreateQualityOption(closestQuality)

	// Build output path
	// yt-dlp will sometimes download two separate files for audio and video
	// so we need to remove the extension and let yt-dlp add the extension
	tmpVideoDownloadExt := filepath.Ext(video.TmpVideoDownloadPath)
	tmpVideoDownloadPathNoExt := strings.TrimSuffix(video.TmpVideoDownloadPath, tmpVideoDownloadExt)

	var cmdArgs []string
	cmdArgs = append(cmdArgs,
		"-f", qualityString,
		url,
		"-o", fmt.Sprintf("%s.%%(ext)s", tmpVideoDownloadPathNoExt),
		"--merge-output-format", "mp4", "--no-part",
		"--no-warnings", "--progress", "--newline", "--no-check-certificate",
	)

	// TODO: setup with yt-dlp
	// pass proxy header
	// if proxyHeader != "" {
	// 	cmdArgs = append(cmdArgs, "--add-headers", proxyHeader)
	// }

	// TODO: setup with yt-dlp
	// pass twitch token as header if available
	// ! token is passed only if proxy is not enabled for security reasons
	// if twitchToken != "" && !proxyFound {
	// 	cmdArgs = append(cmdArgs, "--http-header", fmt.Sprintf("Authorization=OAuth %s", twitchToken))
	// }

	// Sanitize config args before appending
	for _, arg := range configYtDlpArgsArr {
		if strings.TrimSpace(arg) != "" {
			cmdArgs = append(cmdArgs, arg)
		}
	}

	// Create yt-dlp command
	cmd, cookieFile, err := ytdlpSvc.CreateCommand(ctx, cmdArgs)
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
		return fmt.Errorf("error creating yt-dlp command: %w", err)
	}

	log.Debug().Str("channel", channel.Name).Str("cmd", strings.Join(cmd.Args, " ")).Msgf("running yt-dlp")

	// start chat download
	startChat <- true

	cmd.Stderr = file
	cmd.Stdout = file

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Set the process group ID to allow killing child processes
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting yt-dlp: %w", err)
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for the command to finish or context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled, kill only ffmpeg child process if running
		if cmd.Process != nil {
			_ = killYtDlp(cmd.Process.Pid)
		}
		cmd.Process.Wait()
		<-done // Wait for copying to finish
		return ctx.Err()
	case err := <-done:
		// Command finished normally
		if err != nil {
			if exitError, ok := err.(*osExec.ExitError); ok {
				log.Error().Err(err).Str("exitCode", strconv.Itoa(exitError.ExitCode())).Str("exit_error", exitError.Error()).Msg("error running yt-dlp")
				return fmt.Errorf("error running yt-dlp")
			}
			return fmt.Errorf("error running yt-dlp: %w", err)
		}
	}

	return nil
}

func PostProcessVideo(ctx context.Context, video ent.Vod) error {
	env := config.GetEnvConfig()
	configFfmpegArgs := config.Get().Parameters.VideoConvert
	arr := strings.Fields(configFfmpegArgs)
	ffmpegArgs := []string{"-y", "-hide_banner", "-i", video.TmpVideoDownloadPath}

	ffmpegArgs = append(ffmpegArgs, arr...)
	ffmpegArgs = append(ffmpegArgs, video.TmpVideoConvertPath)

	// open log file
	logFilePath := fmt.Sprintf("%s/%s-video-convert.log", env.LogsDir, video.ID.String())
	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Debug().Err(err).Msg("failed to close log file")
		}
	}()
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging ffmpeg output to %s", logFilePath)

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(ffmpegArgs, " ")).Msgf("running ffmpeg")

	cmd := osExec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)

	cmd.Stderr = file
	cmd.Stdout = file

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting ffmpeg: %w", err)
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for the command to finish or context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled, kill the process
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill ffmpeg process: %v", err)
		}
		<-done // Wait for copying to finish
		return ctx.Err()
	case err := <-done:
		// Command finished normally
		if err != nil {
			log.Error().Err(err).Msg("error running ffmpeg")
			return fmt.Errorf("error running ffmpeg: %w", err)
		}
	}

	return nil
}

func ConvertVideoToHLS(ctx context.Context, video ent.Vod) error {
	env := config.GetEnvConfig()
	ffmpegArgs := []string{"-y", "-hide_banner", "-i", video.TmpVideoConvertPath, "-c", "copy", "-start_number", "0", "-hls_time", "10", "-hls_list_size", "0", "-hls_segment_filename", fmt.Sprintf("%s/%s_segment%s.ts", video.TmpVideoHlsPath, video.ExtID, "%d"), "-f", "hls", fmt.Sprintf("%s/%s-video.m3u8", video.TmpVideoHlsPath, video.ExtID)}

	// open log file
	logFilePath := fmt.Sprintf("%s/%s-video-convert.log", env.LogsDir, video.ID.String())
	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Debug().Err(err).Msg("failed to close log file")
		}
	}()

	log.Debug().Str("video_id", video.ID.String()).Msgf("logging ffmpeg output to %s", logFilePath)

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(ffmpegArgs, " ")).Msgf("running ffmpeg")

	cmd := osExec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)

	cmd.Stderr = file
	cmd.Stdout = file

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting ffmpeg: %w", err)
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for the command to finish or context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled, kill the process
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill ffmpeg process: %v", err)
		}
		<-done // Wait for copying to finish
		return ctx.Err()
	case err := <-done:
		// Command finished normally
		if err != nil {
			log.Error().Err(err).Msg("error running ffmpeg")
			return fmt.Errorf("error running ffmpeg: %w", err)
		}
	}

	return nil
}

func DownloadTwitchChat(ctx context.Context, video ent.Vod) error {
	env := config.GetEnvConfig()
	// open log file
	logFilePath := fmt.Sprintf("%s/%s-chat.log", env.LogsDir, video.ID.String())
	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Debug().Err(err).Msg("failed to close log file")
		}
	}()
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging streamlink output to %s", logFilePath)

	var cmdArgs []string
	cmdArgs = append(cmdArgs, "chatdownload", "--id", video.ExtID, "--embed-images", "--collision", "overwrite", "-o", video.TmpChatDownloadPath)

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(cmdArgs, " ")).Msgf("running TwitchDownloaderCLI")

	cmd := osExec.CommandContext(ctx, "TwitchDownloaderCLI", cmdArgs...)

	cmd.Stderr = file
	cmd.Stdout = file

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting TwitchDownloader: %w", err)
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for the command to finish or context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled, kill the process
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill TwitchDownloaderCLI process: %v", err)
		}
		<-done // Wait for copying to finish
		return ctx.Err()
	case err := <-done:
		// Command finished normally
		if err != nil {
			if exitError, ok := err.(*osExec.ExitError); ok {
				log.Error().Err(err).Msg("error running TwitchDownloaderCLI")
				return fmt.Errorf("error running TwitchDownloaderCLI exit code %d: %w", exitError.ExitCode(), exitError)
			}
			log.Error().Err(err).Msg("error running TwitchDownloaderCLI")
			return fmt.Errorf("error running TwitchDownloaderCLI: %w", err)
		}
	}

	return nil
}

func DownloadTwitchLiveChat(ctx context.Context, video ent.Vod, channel ent.Channel, queue ent.Queue) error {
	env := config.GetEnvConfig()
	// set chat start time
	chatStarTime := time.Now()
	_, err := queue.Update().SetChatStart(chatStarTime).Save(ctx)
	if err != nil {
		return err
	}

	// open log file
	logFilePath := fmt.Sprintf("%s/%s-chat.log", env.LogsDir, video.ID.String())
	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Debug().Err(err).Msg("failed to close log file")
		}
	}()
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging chat downloader output to %s", logFilePath)

	var cmdArgs []string
	cmdArgs = append(cmdArgs, fmt.Sprintf("https://twitch.tv/%s", channel.Name), "--output", video.TmpLiveChatDownloadPath, "-q")

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(cmdArgs, " ")).Msgf("running chat_downloader")

	cmd := osExec.CommandContext(ctx, "chat_downloader", cmdArgs...)

	cmd.Stderr = file
	cmd.Stdout = file

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting TwitchDownloader: %w", err)
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for the command to finish or context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled, kill the process
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill TwitchDownloaderCLI process: %v", err)
		}
		<-done // Wait for copying to finish
		return ctx.Err()
	case err := <-done:
		// Command finished normally
		if err != nil {
			if exitError, ok := err.(*osExec.ExitError); ok {
				if status, ok := exitError.Sys().(interface{ ExitStatus() int }); ok {
					if status.ExitStatus() != -1 {
						fmt.Println("chat_downloader terminated - exit code:", status.ExitStatus())
					}
				}
			}
			log.Error().Err(err).Msg("error running chat_downloader")
			return fmt.Errorf("error running chat_downloader: %w", err)
		}
	}

	return nil
}

func RenderTwitchChat(ctx context.Context, video ent.Vod) error {
	env := config.GetEnvConfig()
	// open log file
	logFilePath := fmt.Sprintf("%s/%s-chat-render.log", env.LogsDir, video.ID.String())
	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Debug().Err(err).Msg("failed to close log file")
		}
	}()
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging chat_downloader output to %s", logFilePath)

	var cmdArgs []string

	configRenderArgs := config.Get().Parameters.ChatRender
	configRenderArgsArr := strings.Fields(configRenderArgs)

	cmdArgs = append(cmdArgs, "chatrender", "-i", video.TmpChatDownloadPath, "--collision", "overwrite")

	cmdArgs = append(cmdArgs, configRenderArgsArr...)
	cmdArgs = append(cmdArgs, "-o", video.TmpChatRenderPath)

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(cmdArgs, " ")).Msgf("running TwitchDownloaderCLI")

	cmd := osExec.CommandContext(ctx, "TwitchDownloaderCLI", cmdArgs...)

	cmd.Stderr = file
	cmd.Stdout = file

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting TwitchDownloader: %w", err)
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for the command to finish or context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled, kill the process
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill TwitchDownloaderCLI process: %v", err)
		}
		<-done // Wait for copying to finish
		return ctx.Err()
	case err := <-done:
		// Command finished normally
		if err != nil {
			if exitError, ok := err.(*osExec.ExitError); ok {
				log.Error().Err(err).Msg("error running TwitchDownloaderCLI")
				return fmt.Errorf("error running TwitchDownloaderCLI exit code %d: %w", exitError.ExitCode(), exitError)
			}

			// Check if log output indicates no messages
			noElements, err := checkLogForNoElements(logFilePath)
			if err == nil && noElements {
				return errors.ErrNoChatMessages
			}

			log.Error().Err(err).Msg("error running TwitchDownloaderCLI")
			return fmt.Errorf("error running TwitchDownloaderCLI: %w", err)
		}
	}

	return nil
}

// checkLogForNoElements returns true if the log file contains the expected message.
//
// Used to check if the chat render failure was caused by no messages in the chat.
func checkLogForNoElements(logFilePath string) (bool, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		return false, fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Debug().Err(err).Msg("failed to close log file")
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "Sequence contains no elements") {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("error reading log file: %w", err)
	}

	return false, nil
}

func GetVideoDuration(ctx context.Context, path string) (int, error) {
	cmd := osExec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)

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

func UpdateTwitchChat(ctx context.Context, video ent.Vod) error {
	env := config.GetEnvConfig()
	// open log file
	logFilePath := fmt.Sprintf("%s/%s-chat-convert.log", env.LogsDir, video.ID.String())
	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Debug().Err(err).Msg("failed to close log file")
		}
	}()
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging TwitchDownloader output to %s", logFilePath)

	var cmdArgs []string
	cmdArgs = append(cmdArgs, "chatupdate", "-i", video.TmpLiveChatConvertPath, "--embed-missing", "--collision", "overwrite", "-o", video.TmpChatDownloadPath)

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(cmdArgs, " ")).Msgf("running TwitchDownloaderCLI")

	cmd := osExec.CommandContext(ctx, "TwitchDownloaderCLI", cmdArgs...)

	cmd.Stderr = file
	cmd.Stdout = file

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting TwitchDownloader: %w", err)
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for the command to finish or context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled, kill the process
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill TwitchDownloader process: %v", err)
		}
		<-done // Wait for copying to finish
		return ctx.Err()
	case err := <-done:
		// Command finished normally
		if err != nil {
			if exitError, ok := err.(*osExec.ExitError); ok {
				log.Error().Err(err).Str("exitCode", strconv.Itoa(exitError.ExitCode())).Str("exit_error", exitError.Error()).Msg("error running TwitchDownloader")
				return fmt.Errorf("error running TwitchDownloader")
			}
			return fmt.Errorf("error running TwitchDownloader: %w", err)
		}
	}

	return nil
}

// checkLogForNoStreams returns true if the log file contains the expected message.
//
// Used to check if live stream download failed because no streams were found.
func checkLogForNoStreams(logFilePath string) (bool, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		return false, fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Debug().Err(err)
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "No playable streams found on this URL") {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("error reading log file: %w", err)
	}

	return false, nil
}

func ConvertTwitchVodVideo(v *ent.Vod) error {
	env := config.GetEnvConfig()
	// Fetch config params
	ffmpegParams := config.Get().Parameters.VideoConvert
	// Split supplied params into array
	arr := strings.Fields(ffmpegParams)
	// Generate args for exec
	argArr := []string{"-y", "-hide_banner", "-i", v.TmpVideoDownloadPath}
	// add each config param to arg
	argArr = append(argArr, arr...)
	// add output file
	argArr = append(argArr, v.TmpVideoConvertPath)
	log.Debug().Msgf("video convert args: %v", argArr)
	// Execute ffmpeg
	cmd := osExec.Command("ffmpeg", argArr...)

	videoConvertLogfile, err := os.Create(fmt.Sprintf("%s/%s-video-convert.log", env.LogsDir, v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error creating video convert logfile")
		return err
	}
	defer func() {
		if err := videoConvertLogfile.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing video convert logfile")
		}
	}()
	cmd.Stdout = videoConvertLogfile
	cmd.Stderr = videoConvertLogfile

	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("error running ffmpeg for vod video convert")
		return err
	}

	log.Debug().Msgf("finished vod video convert for %s", v.ExtID)
	return nil
}

func GetFfprobeData(path string) (map[string]interface{}, error) {
	cmd := osExec.Command("ffprobe", "-hide_banner", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	out, err := cmd.Output()
	if err != nil {
		log.Error().Err(err).Msgf("error getting ffprobe data for %s - err: %v", path, err)
		return nil, fmt.Errorf("error getting ffprobe data for %s - err: %w ", path, err)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(out, &data); err != nil {
		log.Error().Err(err).Msg("error unmarshalling ffprobe data")
		return nil, err
	}
	return data, nil
}

// test proxy server by making http request to proxy server
// if request is successful return true
// timeout after 5 seconds
func testProxyServer(url string, header string) bool {
	log.Debug().Msgf("testing proxy server: %s", url)
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error().Err(err).Msg("error creating request for proxy server test")
		return false
	}
	if header != "" {
		log.Debug().Msgf("adding header %s to proxy server test", header)
		splitHeader := strings.SplitN(header, ":", 2)
		req.Header.Add(splitHeader[0], splitHeader[1])
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("error making request for proxy server test")
		return false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing response body for proxy server test")
		}
	}()
	if resp.StatusCode != 200 {
		log.Error().Msgf("proxy server test returned status code %d", resp.StatusCode)
		return false
	}
	log.Debug().Msg("proxy server test successful")
	return true
}

// GenerateStaticThumbnail generates static thumbnail for video.
//
// Resolution is optional and if not set the thumbnail will be generated at the original resolution.
func GenerateStaticThumbnail(ctx context.Context, videoPath string, position int, thumbnailPath string, resolution string) error {
	log.Info().Str("videoPath", videoPath).Str("position", strconv.Itoa(position)).Str("thumbnailPath", thumbnailPath).Str("resolution", resolution).Msg("generating static thumbnail")
	// placing -ss 1 before the input is faster
	// https://stackoverflow.com/questions/27568254/how-to-extract-1-screenshot-for-a-video-with-ffmpeg-at-a-given-time
	ffmpegArgs := []string{"-y", "-hide_banner", "-ss", strconv.Itoa(position), "-i", videoPath, "-vframes", "1", "-update", "1"}
	if resolution != "" {
		ffmpegArgs = append(ffmpegArgs, "-s", resolution)
	}

	ffmpegArgs = append(ffmpegArgs, thumbnailPath)

	cmd := osExec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting ffmpeg: %w", err)
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for the command to finish or context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled, kill the process
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill ffmpeg process: %v", err)
		}
		<-done // Wait for copying to finish
		return ctx.Err()
	case err := <-done:
		// Command finished normally
		if err != nil {
			log.Error().Err(err).Str("ffmpeg_stderr", stderr.String()).Str("ffmpeg_stdout", stdout.String()).Msg("error running ffmpeg")
			return fmt.Errorf("error running ffmpeg: %w", err)
		}
	}

	return nil
}
