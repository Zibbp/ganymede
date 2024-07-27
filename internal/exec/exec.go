package exec

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
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
	file, err := os.Create(logFilePath)
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

	cmd.Stderr = file
	cmd.Stdout = file

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting streamlink: %w", err)
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
			return fmt.Errorf("failed to kill streamlink process: %v", err)
		}
		<-done // Wait for copying to finish
		return ctx.Err()
	case err := <-done:
		// Command finished normally
		if err != nil {
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
	file, err := os.Create(logFilePath)
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
		err := twitch.CheckUserAccessToken(ctx, configTwitchToken)
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

	filteredArgs := make([]string, 0)
	for _, arg := range cmdArgs {
		if arg != "" {
			filteredArgs = append(filteredArgs, arg) //nolint:staticcheck
		}
	}

	// output
	filteredArgs = append(cmdArgs, "-o", video.TmpVideoDownloadPath)

	log.Debug().Str("channel", channel.Name).Str("cmd", strings.Join(filteredArgs, " ")).Msgf("running streamlink")

	// start chat download
	startChat <- true

	cmd := osExec.CommandContext(ctx, "streamlink", filteredArgs...)

	cmd.Stderr = file
	cmd.Stdout = file

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting streamlink: %w", err)
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
			return fmt.Errorf("failed to kill streamlink process: %v", err)
		}
		<-done // Wait for copying to finish
		return ctx.Err()
	case err := <-done:
		// Command finished normally
		if err != nil {
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
	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()
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
	ffmpegArgs := []string{"-y", "-hide_banner", "-i", video.TmpVideoConvertPath, "-c", "copy", "-start_number", "0", "-hls_time", "10", "-hls_list_size", "0", "-hls_segment_filename", fmt.Sprintf("%s/%s_segment%s.ts", video.TmpVideoHlsPath, video.ExtID, "%d"), "-f", "hls", fmt.Sprintf("%s/%s-video.m3u8", video.TmpVideoHlsPath, video.ExtID)}

	// open log file
	logFilePath := fmt.Sprintf("/logs/%s-video-convert.log", video.ID.String())
	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

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

	// open log file
	logFilePath := fmt.Sprintf("/logs/%s-chat.log", video.ID.String())
	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()
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

	// set chat start time
	chatStarTime := time.Now()
	_, err := queue.Update().SetChatStart(chatStarTime).Save(ctx)
	if err != nil {
		return err
	}

	// open log file
	logFilePath := fmt.Sprintf("/logs/%s-chat.log", video.ID.String())
	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()
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
	// open log file
	logFilePath := fmt.Sprintf("/logs/%s-chat-render.log", video.ID.String())
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	log.Debug().Str("video_id", video.ID.String()).Msgf("logging chat_downloader output to %s", logFilePath)

	var cmdArgs []string
	configRenderArgs := viper.GetString("parameters.chat_render")
	configRenderArgsArr := strings.Fields(configRenderArgs)
	cmdArgs = append(cmdArgs, "chatrender", "-i", video.TmpChatDownloadPath, "--collision", "overwrite")
	cmdArgs = append(cmdArgs, configRenderArgsArr...)
	cmdArgs = append(cmdArgs, "-o", video.TmpChatRenderPath)

	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(cmdArgs, " ")).Msgf("running TwitchDownloaderCLI")

	cmd := osExec.CommandContext(ctx, "TwitchDownloaderCLI", cmdArgs...)

	// Use a buffered writer for better performance
	bufWriter := bufio.NewWriter(file)
	cmd.Stdout = bufWriter
	cmd.Stderr = bufWriter

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting TwitchDownloader: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Flush the buffer periodically
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context was cancelled, kill the process
			if err := cmd.Process.Kill(); err != nil {
				log.Error().Err(err).Msg("failed to kill TwitchDownloaderCLI process")
			}
			bufWriter.Flush()
			return ctx.Err()
		case err := <-done:
			// Command finished
			bufWriter.Flush()
			if err != nil {
				if exitError, ok := err.(*osExec.ExitError); ok {
					log.Error().Err(err).Msg("error running TwitchDownloaderCLI")
					return fmt.Errorf("error running TwitchDownloaderCLI exit code %d: %w", exitError.ExitCode(), exitError)
				}
				// Check if log output indicates no messages
				noElements, checkErr := checkLogForNoElements(logFilePath)
				if checkErr == nil && noElements {
					return errors.ErrNoChatMessages
				}
				log.Error().Err(err).Msg("error running TwitchDownloaderCLI")
				return fmt.Errorf("error running TwitchDownloaderCLI: %w", err)
			}
			return nil
		case <-ticker.C:
			// Flush the buffer periodically
			bufWriter.Flush()
		}
	}
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
	file, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()
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
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Error().Msgf("proxy server test returned status code %d", resp.StatusCode)
		return false
	}
	log.Debug().Msg("proxy server test successful")
	return true
}
