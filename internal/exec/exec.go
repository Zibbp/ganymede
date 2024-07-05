package exec

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	osExec "os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/errors"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/utils"
)

func DownloadTwitchVideo(ctx context.Context, video ent.Vod) error {

	// open log file
	logFilePath := fmt.Sprintf("/logs/%s-video.log", video.ID.String())
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging streamlink output to %s", logFilePath)

	var cmdArgs []string
	cmdArgs = append(cmdArgs, fmt.Sprintf("https://twitch.tv/videos/%s", video.ExtID), fmt.Sprintf("%s,best", video.Resolution), "--force-progress", "--force")

	// check if user has twitch token set
	// if so, set token in streamlink command
	twitchToken := viper.GetString("parameters.twitch_token")
	if twitchToken != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--twitch-api-header=Authorization=OAuth %s", twitchToken))
	}

	// output
	cmdArgs = append(cmdArgs, "-o", video.TmpVideoDownloadPath)

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(cmdArgs, " ")).Msgf("running streamlink")

	cmd := osExec.CommandContext(ctx, "streamlink", cmdArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting streamlink: %w", err)
	}

	done := make(chan struct{})
	go func() {
		io.Copy(file, stdout)
		io.Copy(file, stderr)
		close(done)
	}()

	// Wait for the command to finish or context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled, kill the process
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill streamlink process: %v", err)
		}
		<-done // Wait for copying to finish
		return ctx.Err()
	case <-done:
		// Command finished normally
		if err := cmd.Wait(); err != nil {
			if exitError, ok := err.(*osExec.ExitError); ok {
				log.Error().Err(err).Str("exitCode", strconv.Itoa(exitError.ExitCode())).Str("exit_error", exitError.Error()).Msg("error running streamlink")
				return fmt.Errorf("error running streamlink")
			}
			return fmt.Errorf("error running streamlink: %w", err)
		}
	}

	return nil
}

func DownloadTwitchLiveVideo(ctx context.Context, video ent.Vod, channel ent.Channel, startChat chan bool) error {

	// open log file
	logFilePath := fmt.Sprintf("/logs/%s-video.log", video.ID.String())
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging streamlink output to %s", logFilePath)

	configStreamlinkArgs := viper.GetString("parameters.streamlink_live")

	configStreamlinkArgsArr := strings.Split(configStreamlinkArgs, ",")

	proxyEnabled := false
	proxyFound := false
	streamUrl := fmt.Sprintf("https://twitch.tv/%s", channel.Name)
	proxyHeader := ""

	// check if user has proxies enable
	proxyEnabled = viper.GetBool("livestream.proxy_enabled")
	whitelistedChannels := viper.GetStringSlice("livestream.proxy_whitelist") // list of channels that are not allowed to use the proxy
	if proxyEnabled {
		if utils.Contains(whitelistedChannels, channel.Name) {
			log.Debug().Str("channel_name", channel.Name).Msg("channel is whitelisted, not using proxy")
		} else {
			proxyParams := viper.GetString("livestream.proxy_parameters")
			proxyListString := viper.Get("livestream.proxies")
			var proxyList []config.ProxyListItem
			for _, proxy := range proxyListString.([]interface{}) {
				proxyList = append(proxyList, config.ProxyListItem{
					URL:    proxy.(map[string]interface{})["url"].(string),
					Header: proxy.(map[string]interface{})["header"].(string),
				})
			}
			log.Debug().Str("proxy_list", fmt.Sprintf("%v", proxyList)).Msg("proxy list")
			// test proxies
			for _, proxy := range proxyList {
				proxyUrl := fmt.Sprintf("%s/playlist/%s.m3u8%s", proxy.URL, channel.Name, proxyParams)
				if testProxyServer(proxyUrl, proxy.Header) {
					log.Debug().Str("channel_name", channel.Name).Str("proxy_url", proxy.URL).Msg("proxy found")
					proxyFound = true
					streamUrl = fmt.Sprintf("hls://%s", proxyUrl)
					proxyHeader = proxy.Header
					break
				}
			}
		}
	}

	twitchToken := ""
	// check if user has twitch token set
	configTwitchToken := viper.GetString("parameters.twitch_token")
	if configTwitchToken != "" {
		// check if token is valid
		err := twitch.CheckUserAccessToken(configTwitchToken)
		if err != nil {
			log.Error().Err(err).Msg("invalid twitch token")
		} else {
			twitchToken = configTwitchToken
		}
	}

	// streamlink livestreams do not use the 30 fps suffix
	video.Resolution = strings.Replace(video.Resolution, "30", "", 1)

	// streamlink livestreams expect 'audio_only' instead of 'audio'
	if video.Resolution == "audio" {
		video.Resolution = "audio_only"
	}

	var cmdArgs []string
	cmdArgs = append(cmdArgs, streamUrl, fmt.Sprintf("%s,best", video.Resolution), "--force-progress", "--force")

	// pass proxy header
	if proxyHeader != "" {
		cmdArgs = append(cmdArgs, "--add-headers", proxyHeader)
	}

	// pass twitch token as header if available
	// ! token is passed only if proxy is not enabled for security reasons
	if twitchToken != "" && !proxyFound {
		cmdArgs = append(cmdArgs, "--http-header", fmt.Sprintf("Authorization=OAuth %s", twitchToken))
	}

	// pass config args
	cmdArgs = append(cmdArgs, configStreamlinkArgsArr...)

	filteredArgs := make([]string, 0, len(cmdArgs))
	for _, arg := range cmdArgs {
		if arg != "" {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	// output
	filteredArgs = append(cmdArgs, "-o", video.TmpVideoDownloadPath)

	log.Debug().Str("channel", channel.Name).Str("cmd", strings.Join(filteredArgs, " ")).Msgf("running streamlink")

	// start chat download
	startChat <- true

	cmd := osExec.CommandContext(ctx, "streamlink", filteredArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting streamlink: %w", err)
	}

	done := make(chan struct{})
	go func() {
		io.Copy(file, stdout)
		io.Copy(file, stderr)
		close(done)
	}()

	// Wait for the command to finish or context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled, kill the process
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill streamlink process: %v", err)
		}
		<-done // Wait for copying to finish
		return ctx.Err()
	case <-done:
		// Command finished normally
		if err := cmd.Wait(); err != nil {
			// Streamlink will error when the stream goes offline - do not return an error
			log.Info().Str("channel", channel.Name).Str("exit_error", err.Error()).Msg("finished downloading live video")
			// Check if log output indicates no messages
			noStreams, err := checkLogForNoStreams(logFilePath)
			if err == nil && noStreams {
				return utils.NewLiveVideoDownloadNoStreamError("no streams found")
			}
			return nil
		}
	}

	return nil
}

func PostProcessVideo(ctx context.Context, video ent.Vod) error {
	configFfmpegArgs := viper.GetString("parameters.video_convert")
	arr := strings.Fields(configFfmpegArgs)
	ffmpegArgs := []string{"-y", "-hide_banner", "-i", video.TmpVideoDownloadPath}

	ffmpegArgs = append(ffmpegArgs, arr...)
	ffmpegArgs = append(ffmpegArgs, video.TmpVideoConvertPath)

	// open log file
	logFilePath := fmt.Sprintf("/logs/%s-video-convert.log", video.ID.String())
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging ffmpeg output to %s", logFilePath)

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(ffmpegArgs, " ")).Msgf("running ffmpeg")

	cmd := osExec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting ffmpeg: %w", err)
	}

	done := make(chan struct{})
	go func() {
		io.Copy(file, stdout)
		io.Copy(file, stderr)
		close(done)
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
	case <-done:
		// Command finished normally
		if err := cmd.Wait(); err != nil {
			log.Error().Err(err).Msg("error running ffmpeg")
			return fmt.Errorf("error running ffmpeg: %w", err)
		}
	}

	return nil
}

func ConvertVideoToHLS(ctx context.Context, video ent.Vod) error {
	ffmpegArgs := []string{"-y", "-hide_banner", "-i", video.TmpVideoConvertPath, "-c", "copy", "-start_number", "0", "-hls_time", "10", "-hls_list_size", "0", "-hls_segment_filename", fmt.Sprintf("%s/%s_segment%s.ts", video.TmpVideoHlsPath, video.ExtID, "%d"), "-f", "hls", fmt.Sprintf("%s/%s-video.m3u8", video.TmpVideoHlsPath, video.ExtID)}

	// open log file
	logFilePath := fmt.Sprintf("/logs/%s-video-convert.log", video.ID.String())
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	log.Debug().Str("video_id", video.ID.String()).Msgf("logging ffmpeg output to %s", logFilePath)

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(ffmpegArgs, " ")).Msgf("running ffmpeg")

	cmd := osExec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting ffmpeg: %w", err)
	}

	done := make(chan struct{})
	go func() {
		io.Copy(file, stdout)
		io.Copy(file, stderr)
		close(done)
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
	case <-done:
		// Command finished normally
		if err := cmd.Wait(); err != nil {
			log.Error().Err(err).Msg("error running ffmpeg")
			return fmt.Errorf("error running ffmpeg: %w", err)
		}
	}

	return nil
}

func DownloadTwitchChat(ctx context.Context, video ent.Vod) error {

	// open log file
	logFilePath := fmt.Sprintf("/logs/%s-chat.log", video.ID.String())
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging streamlink output to %s", logFilePath)

	var cmdArgs []string
	cmdArgs = append(cmdArgs, "chatdownload", "--id", video.ExtID, "--embed-images", "-o", video.TmpChatDownloadPath)

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(cmdArgs, " ")).Msgf("running TwitchDownloaderCLI")

	cmd := osExec.CommandContext(ctx, "TwitchDownloaderCLI", cmdArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting TwitchDownloaderCLI: %w", err)
	}

	done := make(chan struct{})
	go func() {
		io.Copy(file, stdout)
		io.Copy(file, stderr)
		close(done)
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
	case <-done:
		// Command finished normally
		if err := cmd.Wait(); err != nil {
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

	// set chat start time
	chatStarTime := time.Now()
	_, err := queue.Update().SetChatStart(chatStarTime).Save(ctx)
	if err != nil {
		return err
	}

	// open log file
	logFilePath := fmt.Sprintf("/logs/%s-chat.log", video.ID.String())
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging chat downloader output to %s", logFilePath)

	var cmdArgs []string
	cmdArgs = append(cmdArgs, fmt.Sprintf("https://twitch.tv/%s", channel.Name), "--output", video.TmpLiveChatDownloadPath, "-q")

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(cmdArgs, " ")).Msgf("running chat_downloader")

	cmd := osExec.CommandContext(ctx, "chat_downloader", cmdArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting TwitchDownloaderCLI: %w", err)
	}

	done := make(chan struct{})
	go func() {
		io.Copy(file, stdout)
		io.Copy(file, stderr)
		close(done)
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
	case <-done:
		// Command finished normally
		if err := cmd.Wait(); err != nil {
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

	// open log file
	logFilePath := fmt.Sprintf("/logs/%s-chat-render.log", video.ID.String())
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging TwitchDownloaderCLI output to %s", logFilePath)

	var cmdArgs []string

	configRenderArgs := viper.GetString("parameters.chat_render")
	configRenderArgsArr := strings.Fields(configRenderArgs)

	cmdArgs = append(cmdArgs, "chatrender", "-i", video.TmpChatDownloadPath)

	cmdArgs = append(cmdArgs, configRenderArgsArr...)
	cmdArgs = append(cmdArgs, "-o", video.TmpChatRenderPath)

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(cmdArgs, " ")).Msgf("running TwitchDownloaderCLI")

	cmd := osExec.CommandContext(ctx, "TwitchDownloaderCLI", cmdArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting TwitchDownloaderCLI: %w", err)
	}

	done := make(chan struct{})
	go func() {
		io.Copy(file, stdout)
		io.Copy(file, stderr)
		close(done)
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
	case <-done:
		// Command finished normally
		if err := cmd.Wait(); err != nil {
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
	defer file.Close()

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

	// open log file
	logFilePath := fmt.Sprintf("/logs/%s-chat-convert.log", video.ID.String())
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging TwitchDownloader output to %s", logFilePath)

	var cmdArgs []string
	cmdArgs = append(cmdArgs, "chatupdate", "-i", video.TmpLiveChatConvertPath, "--embed-missing", "-o", video.TmpChatDownloadPath)

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(cmdArgs, " ")).Msgf("running TwitchDownloader")

	cmd := osExec.CommandContext(ctx, "TwitchDownloaderCLI", cmdArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting streamlink: %w", err)
	}

	done := make(chan struct{})
	go func() {
		io.Copy(file, stdout)
		io.Copy(file, stderr)
		close(done)
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
	case <-done:
		// Command finished normally
		if err := cmd.Wait(); err != nil {
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
	defer file.Close()

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

func DownloadTwitchVodVideo(v *ent.Vod) error {

	var argArr []string
	// Check if twitch token is set
	argArr = append(argArr, fmt.Sprintf("https://twitch.tv/videos/%s", v.ExtID), fmt.Sprintf("%s,best", v.Resolution), "--force-progress", "--force")

	twitchToken := viper.GetString("parameters.twitch_token")
	if twitchToken != "" {
		// Note: if the token is invalid, streamlink will exit with "no playable streams found on this URL"
		argArr = append(argArr, fmt.Sprintf("--twitch-api-header=Authorization=OAuth %s", twitchToken))
	}

	argArr = append(argArr, "-o", v.TmpVideoDownloadPath)

	log.Debug().Msgf("running streamlink for vod video download: %s", strings.Join(argArr, " "))

	cmd := osExec.Command("streamlink", argArr...)

	videoLogfile, err := os.Create(fmt.Sprintf("/logs/%s-video.log", v.ID))
	if err != nil {
		return fmt.Errorf("error creating video logfile: %w", err)
	}

	defer videoLogfile.Close()
	cmd.Stdout = videoLogfile
	cmd.Stderr = videoLogfile

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*osExec.ExitError); ok {
			log.Error().Err(err).Msg("error running streamlink for vod download")
			return fmt.Errorf("error running streamlink for vod download with exit code %d: %w", exitError.ExitCode(), exitError)
		}
		return fmt.Errorf("error running streamlink for vod video download: %w", err)
	}

	log.Debug().Msgf("finished downloading vod video for %s", v.ExtID)
	return nil
}

func DownloadTwitchVodChat(v *ent.Vod) error {
	cmd := osExec.Command("TwitchDownloaderCLI", "chatdownload", "--id", v.ExtID, "--embed-images", "-o", v.TmpChatDownloadPath)

	chatLogfile, err := os.Create(fmt.Sprintf("/logs/%s-chat.log", v.ID))
	if err != nil {
		return fmt.Errorf("error creating chat logfile: %w", err)
	}
	defer chatLogfile.Close()
	cmd.Stdout = chatLogfile
	cmd.Stderr = chatLogfile

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*osExec.ExitError); ok {
			log.Error().Err(err).Msg("error running TwitchDownloaderCLI for vod chat download")
			return fmt.Errorf("error running TwitchDownloaderCLI for vod chat download with exit code %d: %w", exitError.ExitCode(), exitError)
		}
		log.Error().Err(err).Msg("error running TwitchDownloaderCLI for vod chat download")
		return fmt.Errorf("error running TwitchDownloaderCLI for vod chat download: %w", err)
	}

	log.Debug().Msgf("finished downloading vod chat for %s", v.ExtID)
	return nil
}

func RenderTwitchVodChat(v *ent.Vod) (error, bool) {
	// Fetch config params
	chatRenderParams := viper.GetString("parameters.chat_render")
	// Split supplied params into array
	arr := strings.Fields(chatRenderParams)
	// Generate args for exec
	argArr := []string{"chatrender", "-i", v.TmpChatDownloadPath}
	// add each config param to arg
	argArr = append(argArr, arr...)
	// add output file
	argArr = append(argArr, "-o", v.TmpChatRenderPath)
	log.Debug().Msgf("chat render args: %v", argArr)
	// Execute chat render
	cmd := osExec.Command("TwitchDownloaderCLI", argArr...)

	chatRenderLogfile, err := os.Create(fmt.Sprintf("/logs/%s-chat-render.log", v.ID))
	if err != nil {
		return fmt.Errorf("error creating chat render logfile: %w", err), true
	}
	defer chatRenderLogfile.Close()
	cmd.Stdout = chatRenderLogfile
	cmd.Stderr = chatRenderLogfile

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*osExec.ExitError); ok {
			log.Error().Err(err).Msg("error running TwitchDownloaderCLI for vod chat render")
			return fmt.Errorf("error running TwitchDownloaderCLI for vod chat render with exit code %d: %w", exitError.ExitCode(), exitError), true
		}
		log.Error().Err(err).Msg("error running TwitchDownloaderCLI for vod chat render")

		// Check if error is because of no messages
		checkCmd := fmt.Sprintf("cat /logs/%s-chat-render.log | grep 'Sequence contains no elements'", v.ID)
		_, err := osExec.Command("bash", "-c", checkCmd).Output()
		if err != nil {
			log.Error().Err(err).Msg("error checking chat render logfile for no messages")
			return fmt.Errorf("erreor checking chat render logfile for no messages %w", err), true
		}

		// TODO: re-implment this
		// log.Debug().Msg("no messages found in chat render logfile. setting vod and queue to reflect no chat.")
		// v.Update().SetChatPath("").SetChatVideoPath("").SaveX(context.Background())
		// q.Update().SetChatProcessing(false).SetTaskChatMove(utils.Success).SaveX(context.Background())
		return nil, false
	}

	log.Debug().Msgf("finished vod chat render for %s", v.ExtID)
	return nil, true
}

func ConvertTwitchVodVideo(v *ent.Vod) error {
	// Fetch config params
	ffmpegParams := viper.GetString("parameters.video_convert")
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

	videoConvertLogfile, err := os.Create(fmt.Sprintf("/logs/%s-video-convert.log", v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error creating video convert logfile")
		return err
	}
	defer videoConvertLogfile.Close()
	cmd.Stdout = videoConvertLogfile
	cmd.Stderr = videoConvertLogfile

	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("error running ffmpeg for vod video convert")
		return err
	}

	log.Debug().Msgf("finished vod video convert for %s", v.ExtID)
	return nil
}

func ConvertToHLS(v *ent.Vod) error {
	// Delete original video file to save space
	log.Debug().Msgf("deleting original video file for %s to save space", v.ExtID)
	if err := os.Remove(v.TmpVideoDownloadPath); err != nil {
		log.Error().Err(err).Msg("error deleting original video file")
		return err
	}

	cmd := osExec.Command("ffmpeg", "-y", "-hide_banner", "-i", v.TmpVideoConvertPath, "-c", "copy", "-start_number", "0", "-hls_time", "10", "-hls_list_size", "0", "-hls_segment_filename", fmt.Sprintf("/tmp/%s_%s-video_hls%s/%s_segment%s.ts", v.ExtID, v.ID, "%v", v.ExtID, "%d"), "-f", "hls", fmt.Sprintf("/tmp/%s_%s-video_hls%s/%s-video.m3u8", v.ExtID, v.ID, "%v", v.ExtID))

	videoConverLogFile, err := os.Open(fmt.Sprintf("/logs/%s-video-convert.log", v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error opening video convert logfile")
		return err
	}
	defer videoConverLogFile.Close()
	cmd.Stdout = videoConverLogFile
	cmd.Stderr = videoConverLogFile

	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("error running ffmpeg for vod video convert - hls")
		return err
	}

	log.Debug().Msgf("finished vod video convert - hls for %s", v.ExtID)
	return nil

}

// func DownloadTwitchLiveVideo(ctx context.Context, v *ent.Vod, ch *ent.Channel, liveChatWorkflowId string) error {
// 	// Fetch config params
// 	liveStreamlinkParams := viper.GetString("parameters.streamlink_live")
// 	// Split supplied params into array
// 	splitStreamlinkParams := strings.Split(liveStreamlinkParams, ",")
// 	// remove param if contains 'twith-api-header' (set by different config value)
// 	for i, param := range splitStreamlinkParams {
// 		if strings.Contains(param, "twitch-api-header") {
// 			log.Info().Msg("twitch-api-header found in streamlink paramters. Please move your token to the dedicated 'twitch token' field.")
// 			splitStreamlinkParams = append(splitStreamlinkParams[:i], splitStreamlinkParams[i+1:]...)
// 		}
// 	}

// 	proxyFound := false
// 	streamURL := ""
// 	proxyHeader := ""

// 	// check if user has proxies enabled
// 	proxyEnabled := viper.GetBool("livestream.proxy_enabled")
// 	whitelistedChannels := viper.GetStringSlice("livestream.proxy_whitelist")
// 	if proxyEnabled {
// 		// check if channel is whitelisted
// 		if utils.Contains(whitelistedChannels, ch.Name) {
// 			log.Debug().Msgf("channel %s is whitelisted - not using proxy", ch.Name)
// 		} else {
// 			// Get proxy parameters
// 			proxyParams := viper.GetString("livestream.proxy_parameters")
// 			// Get proxy list
// 			proxyListString := viper.Get("livestream.proxies")
// 			var proxyList []config.ProxyListItem
// 			for _, proxy := range proxyListString.([]interface{}) {
// 				proxyListItem := config.ProxyListItem{
// 					URL:    proxy.(map[string]interface{})["url"].(string),
// 					Header: proxy.(map[string]interface{})["header"].(string),
// 				}
// 				proxyList = append(proxyList, proxyListItem)
// 			}
// 			log.Debug().Msgf("proxy list: %v", proxyList)
// 			// test proxies
// 			for i, proxy := range proxyList {
// 				proxyUrl := fmt.Sprintf("%s/playlist/%s.m3u8%s", proxy.URL, ch.Name, proxyParams)
// 				if testProxyServer(proxyUrl, proxy.Header) {
// 					log.Debug().Msgf("proxy %d is good", i)
// 					log.Debug().Msgf("setting stream url to %s", proxyUrl)
// 					proxyFound = true
// 					// set proxy stream url (include hls:// so streamlink can download it)
// 					streamURL = fmt.Sprintf("hls://%s", proxyUrl)
// 					// set proxy header
// 					proxyHeader = proxy.Header
// 					break
// 				}
// 			}
// 		}
// 	}

// 	twitchToken := ""
// 	// check if user has twitch token set
// 	configTwitchToken := viper.GetString("parameters.twitch_token")
// 	if configTwitchToken != "" {
// 		// check token is valid
// 		err := twitch.CheckUserAccessToken(configTwitchToken)
// 		if err != nil {
// 			log.Error().Err(err).Msg("error checking twitch token")
// 		} else {
// 			twitchToken = configTwitchToken
// 		}
// 	}

// 	// if proxy not enabled, or none are working, use twitch URL
// 	if streamURL == "" {
// 		streamURL = fmt.Sprintf("https://twitch.tv/%s", ch.Name)
// 	}

// 	// streamlink livestreams do not use the 30 fps suffix
// 	v.Resolution = strings.Replace(v.Resolution, "30", "", 1)

// 	// streamlink livestreams expect 'audio_only' instead of 'audio'
// 	if v.Resolution == "audio" {
// 		v.Resolution = "audio_only"
// 	}

// 	// Generate args for exec
// 	args := []string{"--progress=force", "--force", streamURL, fmt.Sprintf("%s,best", v.Resolution)}

// 	// if proxy requires headers, pass them
// 	if proxyHeader != "" {
// 		args = append(args, "--add-headers", proxyHeader)
// 	}
// 	// pass twitch token as header if available
// 	// only pass if not using proxy for security reasons
// 	if twitchToken != "" && !proxyFound {
// 		args = append(args, "--http-header", fmt.Sprintf("Authorization=OAuth %s", twitchToken))
// 	}

// 	// pass config params
// 	args = append(args, splitStreamlinkParams...)

// 	filteredArgs := make([]string, 0, len(args))
// 	for _, arg := range args {
// 		if arg != "" {
// 			filteredArgs = append(filteredArgs, arg)
// 		}
// 	}

// 	cmdArgs := append(filteredArgs, "-o", v.TmpVideoDownloadPath)

// 	log.Debug().Msgf("streamlink live args: %v", cmdArgs)
// 	log.Debug().Msgf("running: streamlink %s", strings.Join(cmdArgs, " "))

// 	// Start chat download workflow if liveChatWorkflowId is set (chat is being archived)
// 	if liveChatWorkflowId != "" {
// 		// Notify chat download that video download is about to start
// 		log.Debug().Msg("notifying chat download that video download is about to start")

// 		// !send signal to workflow to start chat download
// 		temporal.InitializeTemporalClient()
// 		signal := utils.ArchiveTwitchLiveChatStartSignal{
// 			Start: true,
// 		}
// 		err := temporal.GetTemporalClient().Client.SignalWorkflow(ctx, liveChatWorkflowId, "", "start-chat-download", signal)
// 		if err != nil {
// 			return fmt.Errorf("error sending signal to workflow to start chat download: %w", err)
// 		}
// 	}

// 	// Execute streamlink
// 	cmd := osExec.Command("streamlink", cmdArgs...)

// 	videoLogfile, err := os.Create(fmt.Sprintf("/logs/%s-video.log", v.ID))
// 	if err != nil {
// 		log.Error().Err(err).Msg("error creating video logfile")
// 		return err
// 	}
// 	defer videoLogfile.Close()
// 	cmd.Stderr = videoLogfile
// 	var stdout bytes.Buffer

// 	multiWriterStdout := io.MultiWriter(videoLogfile, &stdout)

// 	cmd.Stdout = multiWriterStdout

// 	if err := cmd.Run(); err != nil {
// 		// Streamlink will error when the stream is offline - do not log this as an error
// 		log.Debug().Msgf("finished downloading live video for %s - %s", v.ExtID, err.Error())
// 		log.Debug().Msgf("streamlink live stdout: %s", stdout.String())
// 		if strings.Contains(stdout.String(), "No playable streams found on this URL") {
// 			log.Error().Msgf("no playable streams found on this URL for %s", v.ExtID)
// 			return utils.NewLiveVideoDownloadNoStreamError("no playable streams found on this URL")
// 		}
// 		return nil
// 	}

// 	log.Debug().Msgf("finished downloading live video for %s", v.ExtID)
// 	return nil
// }

// func DownloadTwitchLiveChat(ctx context.Context, v *ent.Vod, ch *ent.Channel, q *ent.Queue) error {

// 	log.Debug().Msg("setting chat start time")
// 	chatStartTime := time.Now()
// 	_, err := database.DB().Client.Queue.UpdateOneID(q.ID).SetChatStart(chatStartTime).Save(ctx)
// 	if err != nil {
// 		log.Error().Err(err).Msg("error setting chat start time")
// 		return err
// 	}

// 	cmd := osExec.Command("chat_downloader", fmt.Sprintf("https://twitch.tv/%s", ch.Name), "--output", v.TmpLiveChatDownloadPath, "-q")

// 	chatLogfile, err := os.Create(fmt.Sprintf("/logs/%s-chat.log", v.ID))
// 	if err != nil {
// 		log.Error().Err(err).Msg("error creating chat logfile")
// 		return err
// 	}
// 	defer chatLogfile.Close()
// 	cmd.Stdout = chatLogfile
// 	cmd.Stderr = chatLogfile
// 	// Append string to chatLogFile
// 	_, err = chatLogfile.WriteString("Chat downloader started. It it unlikely that you will see further output in this log.")
// 	if err != nil {
// 		log.Error().Err(err).Msg("error writing to chat logfile")
// 	}

// 	if err := cmd.Start(); err != nil {
// 		log.Error().Err(err).Msg("error starting chat_downloader for live chat download")
// 		return err
// 	}

// 	// Wait for the command to finish
// 	if err := cmd.Wait(); err != nil {
// 		// Check if the error is due to a signal
// 		if exitErr, ok := err.(*exec.ExitError); ok {
// 			if status, ok := exitErr.Sys().(interface{ ExitStatus() int }); ok {
// 				if status.ExitStatus() != -1 {
// 					fmt.Println("chat_downloader terminated by signal:", status.ExitStatus())
// 				}
// 			}
// 		}

// 		fmt.Println("error in chat_downloader for live chat download:", err)
// 	}

// 	log.Debug().Msgf("finished downloading live chat for %s", v.ExtID)
// 	return nil
// }

// func GetVideoDuration(path string) (int, error) {
// 	log.Debug().Msg("getting video duration")
// 	cmd := osExec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
// 	out, err := cmd.Output()
// 	if err != nil {
// 		log.Error().Err(err).Msg("error getting video duration")
// 		return 1, err
// 	}
// 	durOut := strings.TrimSpace(string(out))
// 	durFloat, err := strconv.ParseFloat(durOut, 64)
// 	if err != nil {
// 		log.Error().Err(err).Msg("error converting video duration")
// 		return 1, err
// 	}
// 	duration := int(durFloat)
// 	log.Debug().Msgf("video duration: %d", duration)
// 	return duration, nil
// }

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

func TwitchChatUpdate(v *ent.Vod) error {

	cmd := osExec.Command("TwitchDownloaderCLI", "chatupdate", "-i", v.TmpLiveChatConvertPath, "--embed-missing", "-o", v.TmpChatDownloadPath)

	chatLogfile, err := os.Create(fmt.Sprintf("/logs/%s-chat-convert.log", v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error creating chat convert logfile")
		return err
	}
	defer chatLogfile.Close()
	cmd.Stdout = chatLogfile
	cmd.Stderr = chatLogfile

	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("error running TwitchDownloaderCLI for chat update")
		return err
	}

	log.Debug().Msgf("finished updating chat for %s", v.ExtID)
	return nil
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
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Error().Msgf("proxy server test returned status code %d", resp.StatusCode)
		return false
	}
	log.Debug().Msg("proxy server test successful")
	return true
}
