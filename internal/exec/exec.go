package exec

import (
	"bufio"
	"bytes"
	"context"
	stdErrors "errors"
	"fmt"
	"io"
	"os"
	osExec "os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/errors"
	"github.com/zibbp/ganymede/internal/exec/ytdlp"
	"github.com/zibbp/ganymede/internal/hls"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/utils"
)

const (
	archiveShutdownTimeout = 300 * time.Second

	archiveProcessForwarder = `
forward_term() {
	trap '' TERM
	# Give the group a brief grace period, then escalate to SIGKILL. Some
	# ffmpeg builds ignore the first SIGTERM while blocked on a network read
	# (e.g. Twitch HLS), which would otherwise orphan the capture process after
	# a worker crash. The subshell inherits the ignored TERM disposition, so it
	# survives the group-wide SIGTERM below and force-kills any stubborn child.
	( sleep 2; kill -s KILL -- "-$$" 2>/dev/null ) &
	escalation_pid=$!
	kill -s TERM -- "-$$"
}

trap 'forward_term' TERM

"$@" &
child_pid=$!
wait "$child_pid"
status=$?

# Do not report completion while the escalation helper is still running. If the
# child exited before the grace period elapsed we wait for the helper to finish
# (and reap it); if it never exited the helper has already escalated and this
# forwarder was killed with the rest of the group.
if [ -n "${escalation_pid:-}" ]; then
	wait "$escalation_pid" 2>/dev/null
fi

exit $status
`
)

func appendFFmpegLiveOutputStreamArgs(args []string, audioOnly bool) []string {
	streamMap := "0"
	if audioOnly {
		streamMap = "0:a"
	}

	return append(args,
		"-map", streamMap,
		"-dn",
		"-ignore_unknown",
		"-c", "copy",
	)
}

func appendYtDlpVideoConfigArgs(args []string, configArgs string) []string {
	skipValue := false
	for _, arg := range strings.Split(configArgs, ",") {
		trimmedArg := strings.TrimSpace(arg)
		if skipValue {
			skipValue = false
			continue
		}
		if trimmedArg == "-o" || trimmedArg == "--output" || trimmedArg == "-P" || trimmedArg == "--paths" {
			skipValue = true
			continue
		}
		if strings.HasPrefix(trimmedArg, "--output=") || strings.HasPrefix(trimmedArg, "--paths=") ||
			(len(trimmedArg) > 2 && (strings.HasPrefix(trimmedArg, "-o") || strings.HasPrefix(trimmedArg, "-P"))) {
			continue
		}
		if trimmedArg != "" {
			args = append(args, trimmedArg)
		}
	}
	return args
}

func twitchVideoDownloadArgs(quality, url, outputPath, configArgs string) []string {
	args := []string{
		"-f", quality,
		url,
		"-o", outputPath,
		"--merge-output-format", "mp4", "--no-part",
		"--no-warnings", "--progress", "--newline", "--no-check-certificate",
		// Twitch VOD playlists can change their fMP4 initialization segment after a
		// discontinuity. The native HLS downloader rejects a second EXT-X-MAP, while
		// ffmpeg handles the new initialization section correctly.
		"--hls-prefer-ffmpeg",
	}

	// User arguments are intentionally last so an explicit downloader preference
	// in the configuration can override the default. Output options are excluded
	// so the application-owned temporary path remains authoritative.
	return appendYtDlpVideoConfigArgs(args, configArgs)
}

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

	cmdArgs := twitchVideoDownloadArgs(
		qualityString,
		url,
		fmt.Sprintf("%s.%%(ext)s", tmpVideoDownloadPathNoExt),
		config.Get().Parameters.YtDlpVideo,
	)

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

	cmd.SysProcAttr = vodArchiveProcessAttributes()
	done, err := startArchiveCommand(cmd)
	if err != nil {
		return fmt.Errorf("error starting yt-dlp: %w", err)
	}

	// Wait for the command to finish or context to be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled, kill the forwarder process group, including
		// yt-dlp and any ffmpeg process it spawned.
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && !stdErrors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("failed to kill yt-dlp process group: %v", err)
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
	var masterPlaylist *hls.Multivariant

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

	audioOnly := closestQuality == "audio_only"

	// Base ffmpeg args (shared between transport-stream and hls live archiving)
	ffmpegArgs := []string{
		"-y",
		"-hide_banner",
		"-fflags", "+genpts+discardcorrupt",
		"-rw_timeout", "30000000", // 30 second timeout for ffmpeg to connect/read before it gives up and retries
		"-timeout", "30000000", // 30 second timeout for ffmpeg to connect/read before it gives up and retries
		"-i", qualitiesURI[closestQuality],
	}
	ffmpegArgs = appendFFmpegLiveOutputStreamArgs(ffmpegArgs, audioOnly)

	// Decide archive format.
	archivingAsMP4 := (video.VideoHlsPath == "")

	// Append user-defined (global) params before outputs
	videoConvertString := config.Get().Parameters.VideoConvert
	videoConvertArgs := strings.Fields(videoConvertString)
	ffmpegArgs = append(ffmpegArgs, videoConvertArgs...)

	// Archive output
	if archivingAsMP4 {
		// Archive to crash-tolerant MPEG-TS while live; finalize to MP4 in post-process.
		ffmpegArgs = append(ffmpegArgs,
			"-f", "mpegts",
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
				appendFFmpegLiveOutputStreamArgs(nil, audioOnly)...,
			)
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
			appendFFmpegLiveOutputStreamArgs(nil, audioOnly)...,
		)
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
	cmd.SysProcAttr = liveArchiveProcessAttributes()

	log.Debug().Str("channel", channel.Name).Str("cmd", strings.Join(cmd.Args, " ")).Msgf("running ffmpeg")

	// start chat download
	startChat <- true

	cmd.Stderr = file
	cmd.Stdout = file

	done, err := startArchiveCommand(cmd)
	if err != nil {
		return fmt.Errorf("error starting ffmpeg: %w", err)
	}

	// Wait for the command to finish or for ctx cancellation.
	// When ctx is cancelled, allow ffmpeg to handle a graceful shutdown first:
	// send SIGTERM to the process group, wait up to archiveShutdownTimeout, then SIGKILL.
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
		case <-time.After(archiveShutdownTimeout):
			if cmd.Process != nil {
				log.Warn().Msg("ffmpeg process did not exit after SIGTERM, sending SIGKILL")
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

// startArchiveCommand launches a forwarding shim as the worker's direct child.
// If Pdeathsig is delivered to the shim, it forwards the signal to its process
// group so descendants such as yt-dlp's ffmpeg process terminate as well.
//
// The goroutine that creates the shim remains locked to its OS thread until
// Wait returns because Linux ties Pdeathsig to the creating thread.
func startArchiveCommand(cmd *osExec.Cmd) (<-chan error, error) {
	targetPath := cmd.Path
	targetArgs := append([]string(nil), cmd.Args[1:]...)
	shellPath, err := osExec.LookPath("sh")
	if err != nil {
		return nil, fmt.Errorf("find archive process forwarder shell: %w", err)
	}
	cmd.Path = shellPath
	cmd.Args = append(
		[]string{shellPath, "-c", archiveProcessForwarder, "archive-process-forwarder", targetPath},
		targetArgs...,
	)

	started := make(chan error, 1)
	done := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := cmd.Start(); err != nil {
			started <- err
			return
		}
		started <- nil
		done <- cmd.Wait()
	}()

	if err := <-started; err != nil {
		return nil, err
	}
	return done, nil
}

func liveArchiveProcessAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGTERM,
	}
}

func vodArchiveProcessAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGTERM,
	}
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

	return postProcessVideo(ctx, video, configFfmpegArgs, ShouldGeneratePlaylist(), file)
}

func postProcessVideo(ctx context.Context, video ent.Vod, configFfmpegArgs string, generatePlaylist bool, output io.Writer) (err error) {
	// Without an input there is nothing to convert, and an earlier conversion
	// may well be the only copy left: the download is removed once it has been
	// converted. Returning here keeps the cleanup below from touching it.
	if !utils.FileExists(video.TmpVideoDownloadPath) {
		return fmt.Errorf("missing video to post-process: %s", video.TmpVideoDownloadPath)
	}

	// A playlist from an earlier run would survive this one and then describe a
	// video that no longer matches it, which no player can recover from: a
	// syntactically valid playlist is indistinguishable from a correct one. The
	// only safe state is that the sole playlist on disk is the one this run
	// produced, so anything older goes first.
	discardPostProcessedPlaylists(video)

	// Anything half finished is dropped rather than left for a retry to mistake
	// for a complete conversion.
	defer func() {
		if err != nil {
			discardPostProcessedVideo(video)
		}
	}()

	if err := runPostProcessVideoFFmpeg(ctx, video, postProcessVideoFFmpegArgs(video, configFfmpegArgs, generatePlaylist), output); err != nil {
		return err
	}

	duration, err := finalizePostProcessedVideo(ctx, video, generatePlaylist)
	if err != nil {
		return err
	}
	if !duration.HasTimestampAnomaly() {
		return promotePendingPlaylist(video, generatePlaylist)
	}

	log.Warn().
		Str("video_id", video.ID.String()).
		Float64("format_duration", duration.FormatDuration).
		Float64("stream_duration", duration.LongestStreamDuration).
		Msg("detected anomalous container timestamps; normalizing each stream")

	if err := runPostProcessVideoFFmpeg(ctx, video, normalizedPostProcessVideoFFmpegArgs(video, configFfmpegArgs, generatePlaylist), output); err != nil {
		return fmt.Errorf("normalize finalized video timestamps: %w", err)
	}

	duration, err = finalizePostProcessedVideo(ctx, video, generatePlaylist)
	if err != nil {
		return err
	}
	if duration.HasTimestampAnomaly() {
		return fmt.Errorf(
			"timestamp normalization did not repair container duration: format=%f stream=%f",
			duration.FormatDuration,
			duration.LongestStreamDuration,
		)
	}

	return promotePendingPlaylist(video, generatePlaylist)
}

// discardPostProcessedVideo removes what a failed conversion left behind, so
// that a retry cannot mistake it for a finished one.
func discardPostProcessedVideo(video ent.Vod) {
	removePostProcessedFiles(video, video.TmpVideoConvertPath)
	discardPostProcessedPlaylists(video)
}

// discardPostProcessedPlaylists removes the playlists belonging to a conversion.
func discardPostProcessedPlaylists(video ent.Vod) {
	playlistPath := PlaylistPathForVideo(video.TmpVideoConvertPath)
	if playlistPath == "" {
		return
	}
	removePostProcessedFiles(video, playlistPath, PendingPlaylistPath(video.TmpVideoConvertPath), PendingPlaylistPath(video.TmpVideoConvertPath)+".tmp")
}

func removePostProcessedFiles(video ent.Vod, paths ...string) {
	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !stdErrors.Is(err, os.ErrNotExist) {
			log.Error().Err(err).Str("video_id", video.ID.String()).Msgf("failed to remove %s", path)
		}
	}
}

// finalizePostProcessedVideo checks what ffmpeg produced and points the
// playlist at the name the video will carry once it is archived. The playlist
// keeps its temporary name here; promotePendingPlaylist renames it as the last
// step of post-processing.
//
// The video file alone has never shown whether a conversion finished, and a
// fragmented one cannot even fail to probe: cut short, it still reads as valid
// media. The playlist carries that instead, which is why it is written under a
// different name until everything has been checked.
func finalizePostProcessedVideo(ctx context.Context, video ent.Vod, generatePlaylist bool) (MediaDuration, error) {
	if !generatePlaylist {
		duration, err := ProbeMediaDuration(ctx, video.TmpVideoConvertPath)
		if err != nil {
			return MediaDuration{}, fmt.Errorf("probe finalized video duration: %w", err)
		}
		return duration, nil
	}

	// Configured arguments can switch the muxer, in which case ffmpeg reports
	// success and writes something else entirely under the playlist's name.
	playlistPath := PendingPlaylistPath(video.TmpVideoConvertPath)
	if err := ValidatePlaylist(playlistPath); err != nil {
		return MediaDuration{}, fmt.Errorf("post-processing did not produce a playlist: %w", err)
	}

	duration, err := ProbeMediaDuration(ctx, video.TmpVideoConvertPath)
	if err != nil {
		return MediaDuration{}, fmt.Errorf("probe finalized video duration: %w", err)
	}

	// ffmpeg writes the segment references as the bare file name it was given,
	// which is the temporary one. The archived video carries a different name,
	// so the references have to be rewritten before the pair is moved. A video
	// archived as a playlist of its own has no name to point at. That needs
	// saving as HLS to have been on when the record was created and off now,
	// since post-processing writes no playlist in that mode.
	if PlaylistPathForVideo(video.VideoPath) != "" {
		if err := RewritePlaylistMediaName(playlistPath, filepath.Base(video.TmpVideoConvertPath), filepath.Base(video.VideoPath)); err != nil {
			return MediaDuration{}, err
		}
	}

	if err := ValidatePlaylistCoversVideo(playlistPath, video.TmpVideoConvertPath); err != nil {
		return MediaDuration{}, err
	}

	return duration, nil
}

// promotePendingPlaylist gives the playlist the name the rest of the pipeline
// looks for. It is the last thing post-processing does, so that a playlist
// under that name always describes a video nothing will touch again.
func promotePendingPlaylist(video ent.Vod, generatePlaylist bool) error {
	if !generatePlaylist {
		return nil
	}
	if err := os.Rename(PendingPlaylistPath(video.TmpVideoConvertPath), PlaylistPathForVideo(video.TmpVideoConvertPath)); err != nil {
		return fmt.Errorf("name the finished playlist: %w", err)
	}
	return nil
}

// playlistLineNamesMedia reports whether a playlist line refers to a media file
// by name, either as a segment of its own or as the initialization section.
func playlistLineNamesMedia(line, mediaName string) bool {
	return line == mediaName ||
		(strings.HasPrefix(line, "#EXT-X-MAP:") && strings.Contains(line, `URI="`+mediaName+`"`))
}

// RewritePlaylistMediaName replaces the media file name a playlist refers to.
// Only whole lines and the URI attribute of the map tag are rewritten, so a
// name that also appears inside a title or a comment is left alone.
func RewritePlaylistMediaName(playlistPath, oldName, newName string) error {
	if oldName == newName {
		return nil
	}

	// The name ends up in a file browsers parse as a playlist, and it comes
	// from a database field that can be edited through the API. Anything that
	// could end a line or a quoted attribute would let a tag be smuggled in.
	if strings.ContainsAny(newName, "\r\n\"#") {
		return fmt.Errorf("refusing to write %q into a playlist", newName)
	}

	contents, err := os.ReadFile(playlistPath)
	if err != nil {
		return fmt.Errorf("read playlist: %w", err)
	}

	lines := strings.Split(string(contents), "\n")
	replaced := false
	for i, line := range lines {
		if !playlistLineNamesMedia(line, oldName) {
			continue
		}
		if line == oldName {
			lines[i] = newName
		} else {
			lines[i] = strings.Replace(line, `URI="`+oldName+`"`, `URI="`+newName+`"`, 1)
		}
		replaced = true
	}
	// A playlist that names something else does not belong to this video, and
	// writing the new name into it would produce a playlist that parses but
	// describes nothing.
	if !replaced {
		return fmt.Errorf("playlist %s does not reference %s", playlistPath, oldName)
	}

	// Written beside the playlist and renamed over it, so an interrupted write
	// cannot leave a half rewritten playlist behind.
	pending := playlistPath + ".tmp"
	if err := os.WriteFile(pending, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return fmt.Errorf("write playlist: %w", err)
	}
	if err := os.Rename(pending, playlistPath); err != nil {
		if removeErr := os.Remove(pending); removeErr != nil && !stdErrors.Is(removeErr, os.ErrNotExist) {
			log.Error().Err(removeErr).Msgf("failed to remove %s", pending)
		}
		return fmt.Errorf("replace playlist: %w", err)
	}
	return nil
}

func runPostProcessVideoFFmpeg(ctx context.Context, video ent.Vod, ffmpegArgs []string, output io.Writer) error {
	log.Debug().Str("video_id", video.ID.String()).Str("cmd", strings.Join(ffmpegArgs, " ")).Msg("running ffmpeg")

	cmd := osExec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
	cmd.Stderr = output
	cmd.Stdout = output
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.Error().Err(err).Msg("error running ffmpeg")
		return fmt.Errorf("error running ffmpeg: %w", err)
	}
	return nil
}

func postProcessVideoFFmpegArgs(video ent.Vod, configFfmpegArgs string, generatePlaylist bool) []string {
	return buildPostProcessVideoFFmpegArgs(video, configFfmpegArgs, generatePlaylist, false)
}

func normalizedPostProcessVideoFFmpegArgs(video ent.Vod, configFfmpegArgs string, generatePlaylist bool) []string {
	return buildPostProcessVideoFFmpegArgs(video, configFfmpegArgs, generatePlaylist, true)
}

const (
	// playlistHeader is the first line every HLS playlist starts with.
	playlistHeader = "#EXTM3U"
	// playlistEnd is written once the recording has been described in full, so
	// it is what separates a finished playlist from an interrupted one.
	playlistEnd = "#EXT-X-ENDLIST"
)

// ShouldGeneratePlaylist reports whether post-processing writes a playlist
// beside the video it produces.
//
// Everything that waits for a playlist has to ask the same question, or it ends
// up waiting for a file nobody writes. Saving as HLS converts the output again
// and throws the pair away, so a playlist there would only be written to be
// deleted; a live archive finalized as MP4 is not converted again and simply
// goes without one.
func ShouldGeneratePlaylist() bool {
	conf := config.Get()
	return conf.Archive.GenerateVideoPlaylist && !conf.Archive.SaveAsHls
}

// PlaylistPathForVideo returns the playlist that belongs to a video file. The
// two share a directory and a base name, so the playlist can be derived from
// the video path alone. An empty path has no playlist.
func PlaylistPathForVideo(videoPath string) string {
	extension := filepath.Ext(videoPath)
	if videoPath == "" || strings.EqualFold(extension, ".m3u8") {
		return ""
	}
	return strings.TrimSuffix(videoPath, extension) + ".m3u8"
}

// PendingPlaylistPath is where a playlist is written while post-processing is
// still working on the video. It is renamed only after the last step, so a
// playlist under the name the rest of the code looks for always describes a
// video that is finished and will not be rewritten. ffmpeg closes a playlist
// even when it is interrupted part way through, so without this the two would
// be indistinguishable.
func PendingPlaylistPath(videoPath string) string {
	playlistPath := PlaylistPathForVideo(videoPath)
	if playlistPath == "" {
		return ""
	}
	return playlistPath + ".part"
}

// ValidatePlaylistNamesVideo reports whether a playlist refers to the video
// beside it by name.
//
// The name and the byte ranges can disagree: a rename elsewhere in the pipeline
// moves the pair together and leaves the ranges fitting a file the playlist no
// longer names.
func ValidatePlaylistNamesVideo(playlistPath, videoPath string) error {
	contents, err := os.ReadFile(playlistPath)
	if err != nil {
		return fmt.Errorf("read playlist: %w", err)
	}

	mediaName := filepath.Base(videoPath)
	for _, line := range strings.Split(string(contents), "\n") {
		if playlistLineNamesMedia(line, mediaName) {
			return nil
		}
	}
	return fmt.Errorf("playlist %s does not refer to %s", playlistPath, mediaName)
}

// ValidatePlaylistCoversVideo reports whether a playlist describes exactly the
// video beside it.
//
// The playlist addresses the video by byte offset, so the two only work as a
// pair: a video that was rewritten, truncated or replaced leaves the offsets
// pointing at something else, and a playlist that still parses cannot be
// recognised as wrong by any player. The last byte range therefore has to end
// exactly at the end of the video.
func ValidatePlaylistCoversVideo(playlistPath, videoPath string) error {
	contents, err := os.ReadFile(playlistPath)
	if err != nil {
		return fmt.Errorf("read playlist: %w", err)
	}

	info, err := os.Stat(videoPath)
	if err != nil {
		return fmt.Errorf("stat video: %w", err)
	}

	var covered int64
	for _, line := range strings.Split(string(contents), "\n") {
		byteRange := ""
		switch {
		case strings.HasPrefix(line, "#EXT-X-BYTERANGE:"):
			byteRange = strings.TrimPrefix(line, "#EXT-X-BYTERANGE:")
		case strings.HasPrefix(line, "#EXT-X-MAP:"):
			_, rest, found := strings.Cut(line, `BYTERANGE="`)
			if !found {
				continue
			}
			byteRange, _, _ = strings.Cut(rest, `"`)
		default:
			continue
		}

		// A range is written as <length>@<offset>.
		lengthText, offsetText, found := strings.Cut(byteRange, "@")
		if !found {
			return fmt.Errorf("playlist %s has an unreadable byte range %q", playlistPath, byteRange)
		}
		length, lengthErr := strconv.ParseInt(lengthText, 10, 64)
		offset, offsetErr := strconv.ParseInt(offsetText, 10, 64)
		if lengthErr != nil || offsetErr != nil || length < 0 || offset < 0 {
			return fmt.Errorf("playlist %s has an unreadable byte range %q", playlistPath, byteRange)
		}
		covered = max(covered, offset+length)
	}

	if covered != info.Size() {
		return fmt.Errorf("playlist %s describes %d bytes of a %d byte video", playlistPath, covered, info.Size())
	}
	return nil
}

// ValidatePlaylist reports whether the file at playlistPath is a playlist that
// was written in full. It says nothing about the video beside it; see
// PendingPlaylistPath for how the two are tied together.
func ValidatePlaylist(playlistPath string) error {
	contents, err := os.ReadFile(playlistPath)
	if err != nil {
		return fmt.Errorf("read playlist: %w", err)
	}
	if !bytes.HasPrefix(contents, []byte(playlistHeader)) {
		return fmt.Errorf("%s is not a playlist", playlistPath)
	}
	if !bytes.Contains(contents, []byte(playlistEnd)) {
		return fmt.Errorf("playlist %s is incomplete", playlistPath)
	}
	return nil
}

func buildPostProcessVideoFFmpegArgs(video ent.Vod, configFfmpegArgs string, generatePlaylist bool, normalizeTimestamps bool) []string {
	arr := strings.Fields(configFfmpegArgs)
	ffmpegArgs := []string{"-y", "-hide_banner", "-fflags", "+genpts", "-i", video.TmpVideoDownloadPath, "-map", "0", "-dn", "-ignore_unknown", "-c", "copy"}
	audioBitstreamFilters := []string{"aac_adtstoasc"}
	if normalizeTimestamps {
		// One filter chain per stream type, so STARTDTS stays local to each and
		// large audio/video start offsets are repaired without decoding. The
		// audio filter has to survive: the fragmented muxer rejects ADTS AAC,
		// which is what a live archive downloads, and unlike the MP4 muxer it
		// does not insert the filter itself.
		audioBitstreamFilters = append(audioBitstreamFilters, "setts=ts=TS-STARTDTS")
	}
	ffmpegArgs = append(ffmpegArgs, "-bsf:a", strings.Join(audioBitstreamFilters, ","))

	// The video is written as one fragmented MP4 with a playlist beside it that
	// addresses the file by byte ranges. A progressive MP4 of several hours
	// carries a sample table large enough that Apple devices load the metadata,
	// decode a frame and then never start playing. Reading the layout from the
	// playlist avoids that. The video file stays a valid MP4 that plays and
	// seeks on its own, but a fragmented one carries its duration in the
	// fragments rather than the header, so a tool that reads only the header
	// sees none.
	if generatePlaylist {
		// The HLS muxer rejects the subtitle codecs an MP4 can carry, and a
		// rejected stream fails the whole conversion.
		ffmpegArgs = append(ffmpegArgs,
			"-sn",
			"-f", "hls",
			// Apple's authoring specification asks for six second segments,
			// which suits a live stream switching between renditions. For a
			// multi-hour recording addressed by byte ranges the whole playlist
			// is read before the first frame, and the larger index costs more
			// than the shorter segments save: on iOS 26.5, playing a 4.83 hour
			// recording over a local network, six second segments took 796 ms
			// to the first frame where ten took 512 ms. Ten also matches the
			// segment length this project already uses for its transport
			// stream playlists.
			"-hls_time", "10",
			"-hls_list_size", "0",
			"-hls_playlist_type", "vod",
			"-hls_segment_type", "fmp4",
			"-hls_flags", "single_file",
			"-hls_segment_filename", video.TmpVideoConvertPath,
		)
	} else {
		ffmpegArgs = append(ffmpegArgs, "-f", "mp4", "-movflags", "+faststart")
	}

	if normalizeTimestamps {
		ffmpegArgs = append(ffmpegArgs, "-bsf:v", "setts=ts=TS-STARTDTS")
	}

	ffmpegArgs = append(ffmpegArgs, "-metadata", "title="+video.Title)
	// Configured arguments come last so they can override any of the above,
	// which includes the bitstream filters: ffmpeg applies the last one given
	// per stream.
	ffmpegArgs = append(ffmpegArgs, arr...)
	if generatePlaylist {
		ffmpegArgs = append(ffmpegArgs, PendingPlaylistPath(video.TmpVideoConvertPath))
	} else {
		ffmpegArgs = append(ffmpegArgs, video.TmpVideoConvertPath)
	}

	return ffmpegArgs
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
	cmdArgs = append(cmdArgs, "chatdownload", "--id", video.ExtID, "--embed-images", "--collision", "overwrite")

	// Forward shared chat flags from the chat render config so the user's
	// --bttv/--ffz/--stv/--temp-path preferences also apply to chatdownload,
	// which fetches emotes for embedding when --embed-images is set.
	configRenderArgs := config.Get().Parameters.ChatRender
	cmdArgs = append(cmdArgs, extractSharedChatArgs(strings.Fields(configRenderArgs))...)

	cmdArgs = append(cmdArgs, "-o", video.TmpChatDownloadPath)

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

// extractSharedChatArgs returns the subset of args from a chatrender arg list
// that are also valid for chatupdate and chatdownload.
func extractSharedChatArgs(args []string) []string {
	// --collision is omitted: every caller hardcodes it.
	sharedFlagNames := []string{"--bttv", "--ffz", "--stv", "--temp-path"}
	var result []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		for _, flag := range sharedFlagNames {
			if arg == flag {
				result = append(result, arg)
				// Only consume the next token as a value if it isn't itself a flag,
				// otherwise `--stv --temp-path /p` would swallow `--temp-path`.
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
					result = append(result, args[i+1])
					i++
				}
				break
			}
			if strings.HasPrefix(arg, flag+"=") {
				result = append(result, arg)
				break
			}
		}
	}
	return result
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
	cmdArgs = append(cmdArgs, "chatupdate", "-i", video.TmpLiveChatConvertPath, "--embed-missing", "--collision", "overwrite")

	// Forward shared chat flags from the chat render config so the user's
	// --bttv/--ffz/--stv/--temp-path preferences also apply to chatupdate,
	// which fetches emotes for embedding and can otherwise hang on a
	// third-party timeout.
	configRenderArgs := config.Get().Parameters.ChatRender
	cmdArgs = append(cmdArgs, extractSharedChatArgs(strings.Fields(configRenderArgs))...)

	cmdArgs = append(cmdArgs, "-o", video.TmpChatDownloadPath)

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
