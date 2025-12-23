package exec

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/grafov/m3u8"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/errors"
	"github.com/zibbp/ganymede/internal/exec/ytdlp"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/utils"
)

const (
	sigtermTimeout = 30 * time.Second
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
	qualities, err := ytdlpSvc.GetVideoQualities(ctx, video)
	if err != nil {
		return fmt.Errorf("error getting video quality options: %w", err)
	}

	closestQuality := utils.SelectClosestQuality(video.Resolution, qualities)
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
	cmd, cookieFile, err := ytdlpSvc.CreateCommand(ctx, cmdArgs, true)
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

	log.Debug().Str("video_id", video.ID.String()).Msgf("logging ffmpeg output to %s", logFilePath)

	proxyFound := false // Whether a proxy was found
	var masterPlaylist *m3u8.MasterPlaylist

	twitchURL := utils.CreateTwitchURL(video.ExtID, video.Type, channel.Name)

	// Handle proxy setting
	proxyEnabled := config.Get().Livestream.ProxyEnabled
	whitelistedChannels := config.Get().Livestream.ProxyWhitelist // list of channels that are whitelisted from using proxy
	if proxyEnabled {
		if utils.Contains(whitelistedChannels, channel.Name) {
			log.Debug().Str("channel_name", channel.Name).Msg("channel is whitelisted, not using proxy")
		} else {
			proxyParams := config.Get().Livestream.ProxyParameters
			proxyList := config.Get().Livestream.Proxies

			log.Debug().Str("proxy_list", fmt.Sprintf("%v", proxyList)).Msg("proxy list")

			// Try proxies - the first one that works will be used
			for _, proxy := range proxyList {
				// proxyUrl is url that will be sent to ffmpeg for download
				// this can be a direct URL or a proxy URL
				proxyUrl := twitchURL
				if proxy.ProxyType == utils.ProxyTypeTwitchHLS {
					proxyUrl = fmt.Sprintf("%s/playlist/%s.m3u8%s", proxy.URL, channel.Name, proxyParams)
				}
				// Try the proxy server
				var ok bool
				masterPlaylist, ok = tryProxyServer(proxy.URL, proxyUrl, proxy.Header, proxy.ProxyType)
				if ok {
					log.Debug().Str("channel_name", channel.Name).Str("proxy_url", proxy.URL).Msg("proxy found")
					proxyFound = true
					break
				}
			}
		}
	}

	if !proxyFound {
		tc := &platform.TwitchConnection{}
		masterPlaylist, err = tc.GetStream(ctx, channel.Name)
		if err != nil {
			return fmt.Errorf("failed to get stream: %v", err)
		}
	}

	qualities := make([]string, 0, len(masterPlaylist.Variants))
	qualitiesURI := make(map[string]string, len(masterPlaylist.Variants))
	for _, variant := range masterPlaylist.Variants {
		qualities = append(qualities, variant.Video)
		qualitiesURI[variant.Video] = variant.URI
	}
	log.Debug().Strs("available_qualities", qualities).Msg("available stream qualities")
	for b, a := range qualitiesURI {
		log.Debug().Str("quality", b).Str("quality_uri", a).Msg("quality uri")
	}

	closestQuality := utils.SelectClosestQuality(video.Resolution, qualities)
	log.Info().Str("requested_quality", video.Resolution).Msgf("selected closest quality %s", closestQuality)

	if closestQuality == "audio" {
		closestQuality = "audio_only"
	}

	// Base ffmpeg args (shared between mp4 and hls)
	ffmpegArgs := []string{
		"-y",
		"-hide_banner",
		"-fflags", "+genpts+discardcorrupt",
		"-i", qualitiesURI[closestQuality],
		"-map", "0",
		"-dn",
		"-ignore_unknown",
		"-c", "copy",
		"-movflags", "+faststart",
	}

	// Decide archive format.
	archivingAsMP4 := (video.VideoHlsPath == "")

	// Append user-defined (global) params before outputs
	videoConvertString := config.Get().Parameters.VideoConvert
	videoConvertArgs := strings.Fields(videoConvertString)
	ffmpegArgs = append(ffmpegArgs, videoConvertArgs...)

	// Archive output
	if archivingAsMP4 {
		// Archive as MP4
		ffmpegArgs = append(ffmpegArgs,
			"-bsf:a", "aac_adtstoasc",
			"-f", "mp4",
			video.TmpVideoDownloadPath,
		)

		// Also archive HLS for watch-while-archiving
		if config.Get().Livestream.WatchWhileArchiving && video.TmpVideoHlsPath != "" {
			if err := utils.CreateDirectory(video.TmpVideoHlsPath); err != nil {
				return fmt.Errorf("error creating hls directory: %w", err)
			}

			playlistPath := fmt.Sprintf("%s/%s-video.m3u8", video.TmpVideoHlsPath, video.ExtID)
			segmentPattern := fmt.Sprintf("%s/%s_segment%%06d.ts", video.TmpVideoHlsPath, video.ExtID)

			ffmpegArgs = append(ffmpegArgs,
				"-start_number", "0",
				"-hls_time", "2",
				"-hls_list_size", "0",
				"-hls_playlist_type", "event",
				"-hls_flags", "append_list+independent_segments",
				"-hls_segment_filename", segmentPattern,
				"-f", "hls",
				playlistPath,
			)
		}
	} else {
		// Archive as HLS
		if err := utils.CreateDirectory(video.TmpVideoHlsPath); err != nil {
			return fmt.Errorf("error creating hls directory: %w", err)
		}

		playlistPath := fmt.Sprintf("%s/%s-video.m3u8", video.TmpVideoHlsPath, video.ExtID)
		segmentPattern := fmt.Sprintf("%s/%s_segment%%06d.ts", video.TmpVideoHlsPath, video.ExtID)

		ffmpegArgs = append(ffmpegArgs,
			"-start_number", "0",
			"-hls_time", "10",
			"-hls_list_size", "0",
			"-hls_playlist_type", "event",
			"-hls_flags", "append_list+independent_segments",
			"-hls_segment_filename", segmentPattern,
			"-f", "hls",
			playlistPath,
		)
	}

	// Run ffmpeg
	cmd := osExec.Command("ffmpeg", ffmpegArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	log.Debug().Str("channel", channel.Name).Str("cmd", strings.Join(cmd.Args, " ")).Msgf("running ffmpeg")

	// start chat download
	startChat <- true

	cmd.Stderr = file
	cmd.Stdout = file

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting ffmpeg: %w", err)
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for the command to finish or for ctx cancellation.
	// When ctx is cancelled, allow ffmpeg to handle a graceful shutdown first:
	// send SIGTERM to the process group, wait up to sigtermTimeout, then SIGKILL
	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			err = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
			if err != nil {
				log.Error().Err(err).Msg("failed to send SIGTERM to ffmpeg process")
			}
		}
		select {
		case <-done:
			// exited after SIGTERM
		case <-time.After(sigtermTimeout):
			if cmd.Process != nil {
				err = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				if err != nil {
					log.Error().Err(err).Msg("failed to send SIGKILL to ffmpeg process")
				}
			}
			// wait for it to actually exit (best effort)
			select {
			case <-done:
			case <-time.After(5 * time.Second):
			}
		}
		return ctx.Err()
	case err := <-done:
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

func PostProcessVideo(ctx context.Context, video ent.Vod) error {
	env := config.GetEnvConfig()
	configFfmpegArgs := config.Get().Parameters.VideoConvert
	arr := strings.Fields(configFfmpegArgs)
	ffmpegArgs := []string{"-y", "-hide_banner", "-fflags", "+genpts", "-i", video.TmpVideoDownloadPath, "-map", "0", "-dn", "-ignore_unknown", "-c", "copy", "-f", "mp4", "-bsf:a", "aac_adtstoasc", "-movflags", "+faststart"}

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
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging chatdownload output to %s", logFilePath)

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
	log.Debug().Str("video_id", video.ID.String()).Msgf("logging TwitchDownloaderCLI output to %s", logFilePath)

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
