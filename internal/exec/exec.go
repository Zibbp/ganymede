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
	"slices"
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

	return postProcessVideo(ctx, video, configFfmpegArgs, config.Get().Archive.TagHevcAsHvc1, file)
}

func postProcessVideo(ctx context.Context, video ent.Vod, configFfmpegArgs string, tagHevcAsHvc1 bool, output io.Writer) error {
	// An unknown codec keeps ffmpeg's default tagging, so a failed probe must
	// not fail the archive.
	sourceVideoCodec, err := ProbeVideoCodec(ctx, video.TmpVideoDownloadPath)
	if err != nil {
		log.Warn().Err(err).Str("video_id", video.ID.String()).Msg("could not probe source video codec")
	}

	retagHevc := tagHevcAsHvc1 && shouldTagHevcAsHvc1(sourceVideoCodec, configFfmpegArgs)
	// ffmpeg reports a rejected tag in the conversion log only, so record what
	// was decided next to the arguments it was decided from.
	log.Debug().
		Str("video_id", video.ID.String()).
		Str("source_video_codec", sourceVideoCodec).
		Bool("retag_hevc_as_hvc1", retagHevc).
		Msg("post-processing video")

	if err := runPostProcessVideoFFmpeg(ctx, video, postProcessVideoFFmpegArgs(video, configFfmpegArgs, retagHevc), output); err != nil {
		return err
	}

	duration, err := ProbeMediaDuration(ctx, video.TmpVideoConvertPath)
	if err != nil {
		return fmt.Errorf("probe finalized video duration: %w", err)
	}
	if !duration.HasTimestampAnomaly() {
		return nil
	}

	log.Warn().
		Str("video_id", video.ID.String()).
		Float64("format_duration", duration.FormatDuration).
		Float64("stream_duration", duration.LongestStreamDuration).
		Msg("detected anomalous container timestamps; normalizing each stream")

	if err := runPostProcessVideoFFmpeg(ctx, video, normalizedPostProcessVideoFFmpegArgs(video, configFfmpegArgs, retagHevc), output); err != nil {
		return fmt.Errorf("normalize finalized video timestamps: %w", err)
	}

	duration, err = ProbeMediaDuration(ctx, video.TmpVideoConvertPath)
	if err != nil {
		return fmt.Errorf("probe timestamp-normalized video duration: %w", err)
	}
	if duration.HasTimestampAnomaly() {
		return fmt.Errorf(
			"timestamp normalization did not repair container duration: format=%f stream=%f",
			duration.FormatDuration,
			duration.LongestStreamDuration,
		)
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

func postProcessVideoFFmpegArgs(video ent.Vod, configFfmpegArgs string, retagHevcAsHvc1 bool) []string {
	return buildPostProcessVideoFFmpegArgs(video, configFfmpegArgs, retagHevcAsHvc1, false)
}

func normalizedPostProcessVideoFFmpegArgs(video ent.Vod, configFfmpegArgs string, retagHevcAsHvc1 bool) []string {
	return buildPostProcessVideoFFmpegArgs(video, configFfmpegArgs, retagHevcAsHvc1, true)
}

// shouldTagHevcAsHvc1 reports whether a source-derived hvc1 tag may be applied
// to the conversion the configured arguments describe.
//
// hev1 is the mov/mp4 muxer's default tag for HEVC, and a copy inherits
// whatever tag a downloaded MP4 already carried; for Twitch it is hev1 either
// way. AVFoundation refuses such a file outright, so the recording never loads
// in Safari or on iOS - the refusal is at the container level, not the
// decoder's, which handles hev1 samples happily once something feeds them.
//
// Getting the answer wrong in one direction is far worse than the other: the
// mp4 muxer refuses to write a file whose tag does not match its codec, which
// fails the whole archive, while a missing tag only leaves the recording as
// unplayable as it already was. Every case that cannot be decided from the
// arguments alone therefore answers no, and the predicates below say no more
// about that.
//
// One cost applies to live captures only. Their download is MPEG-TS, whose
// samples carry the parameter sets in band; writing them under an hvc1 sample
// entry drops them, and only the set already in hvcC - the first one - is left.
// The remainder of a capture whose parameter sets change midway is then decoded
// with the first segment's parameters, which ranges from the wrong geometry to
// VideoToolbox refusing the picture outright. ffmpeg does not warn. A downloaded
// MP4 is unaffected: its samples pass through byte for byte. Measured on the
// FFmpeg 9.0 this project ships; 8.x strips as well, 7.1 and earlier did not.
//
// This muxer writes one sample entry, so it cannot express such a capture as
// both conforming hvc1 and intact - hence archive.tag_hevc_as_hvc1, which lets
// an installation choose the other side.
func shouldTagHevcAsHvc1(sourceVideoCodec, configFfmpegArgs string) bool {
	return sourceVideoCodec == "hevc" &&
		videoStreamIsCopied(configFfmpegArgs) &&
		!configSelectsStreams(configFfmpegArgs)
}

// codecOptionScope reports how an option relates to the first video stream,
// which is the only one that matters here: it is the stream the source codec is
// probed from, and the stream the tag addresses.
type codecOptionScope int

const (
	// scopeOther: the option leaves the first video stream alone. Option values
	// and every option that is not a codec option land here too, which is how
	// the scan skips them.
	scopeOther codecOptionScope = iota
	// scopeVideo: the option applies to the first video stream.
	scopeVideo
	// scopeUnknown: which streams the option applies to cannot be told without
	// resolving the output layout.
	scopeUnknown
)

// scopeOfCodecOption classifies a single argument. A codec option is decided
// when its specifier is absent, names a non-video type, or names video without
// narrowing it further ("v", "V", "v:0"). It is undecidable when a selector
// narrows video down, because "v:m:language:eng" may match no stream at all
// while an earlier re-encode still stands, and for every other selector form -
// a bare index, a program, a stream id, a stream group - which needs the
// resolved output layout: -dn and -ignore_unknown already shift the numbering.
//
// "V" is treated as "v" although it means video excluding attached pictures.
// The two differ only when the first video stream is itself an attached
// picture, in which case the probe reported that picture's codec and no tag is
// added anyway.
func scopeOfCodecOption(option string) codecOptionScope {
	name, spec, hasSpec := strings.Cut(option, ":")
	switch name {
	case "-vcodec":
		// The legacy spelling takes no stream specifier.
		return scopeVideo
	case "-c", "-codec":
	default:
		return scopeOther
	}
	if !hasSpec || spec == "" {
		return scopeVideo
	}
	switch spec[0] {
	case 'v', 'V':
		if rest := spec[1:]; rest != "" && rest != ":0" {
			return scopeUnknown
		}
		return scopeVideo
	case 'a', 's', 'd', 't':
		return scopeOther
	}
	return scopeUnknown
}

// videoStreamIsCopied reports whether the configured arguments leave the first
// video stream untouched. An encoder name says nothing about the codec it
// produces, so a sample entry tag derived from the source may only be applied
// to a copy. ffmpeg applies the last codec option that matches a stream, so
// later arguments win here too, except that an option whose scope cannot be
// resolved short-circuits the whole answer.
func videoStreamIsCopied(configFfmpegArgs string) bool {
	// The caller's own arguments start with "-c copy".
	videoCodec := "copy"
	arr := strings.Fields(configFfmpegArgs)
	for i := 0; i < len(arr)-1; i++ {
		switch scopeOfCodecOption(arr[i]) {
		case scopeUnknown:
			// Answering no is the safe direction, as above.
			return false
		case scopeVideo:
			videoCodec = arr[i+1]
		}
	}
	return videoCodec == "copy"
}

// configSelectsStreams reports whether the configured arguments contain a -map.
// A filtergraph can introduce output streams too, but combined with the copy
// this conversion performs ffmpeg refuses the command outright, so -map is the
// only form that reaches the muxer. The tag addresses the first video stream of the output,
// which is the stream the source codec was probed from as long as ffmpeg's own
// -map 0 decides the layout. Extra -map options accumulate onto that one, so a
// positive map cannot displace the probed stream, but a negative map can drop
// it and promote another one, which then carries a tag its codec does not match
// and ffmpeg refuses the output. Telling the two apart means resolving the
// mapping, so the presence of any -map is taken as reason enough to leave the
// tag off.
func configSelectsStreams(configFfmpegArgs string) bool {
	return slices.Contains(strings.Fields(configFfmpegArgs), "-map")
}

func buildPostProcessVideoFFmpegArgs(video ent.Vod, configFfmpegArgs string, retagHevcAsHvc1 bool, normalizeTimestamps bool) []string {
	arr := strings.Fields(configFfmpegArgs)
	ffmpegArgs := []string{"-y", "-hide_banner", "-fflags", "+genpts", "-i", video.TmpVideoDownloadPath, "-map", "0", "-dn", "-ignore_unknown", "-c", "copy", "-f", "mp4"}
	if !normalizeTimestamps {
		ffmpegArgs = append(ffmpegArgs, "-bsf:a", "aac_adtstoasc")
	}
	ffmpegArgs = append(ffmpegArgs, "-movflags", "+faststart")

	// See shouldTagHevcAsHvc1 for why a copied HEVC stream is retagged at all.
	//
	// The tag names the stream the codec was probed from. An unqualified -tag:v
	// would also reach a second video stream, such as a thumbnail embedded by
	// yt-dlp, and the mp4, mov and matroska muxers then refuse to write the
	// header at all (mpegts ignores a mismatched tag instead). Setting it here
	// rather than patching the sample entry of a finished file is what makes
	// the result conforming: hvc1 promises the parameter sets are in hvcC, and
	// only the muxer can keep that promise. Configured arguments follow and can
	// therefore still override the tag.
	if retagHevcAsHvc1 {
		ffmpegArgs = append(ffmpegArgs, "-tag:v:0", "hvc1")
	}

	ffmpegArgs = append(ffmpegArgs, "-metadata", "title="+video.Title)
	ffmpegArgs = append(ffmpegArgs, arr...)
	if normalizeTimestamps {
		// Apply one bitstream-filter instance per output stream. STARTDTS is
		// therefore local to each stream, repairing large audio/video start
		// offsets without decoding or re-encoding the media.
		ffmpegArgs = append(ffmpegArgs, "-bsf", "setts=ts=TS-STARTDTS")
	}
	ffmpegArgs = append(ffmpegArgs, video.TmpVideoConvertPath)

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
