package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/exec/ytdlp"
	"github.com/zibbp/ganymede/internal/utils"
)

const (
	kickWebsocketURL = "wss://ws-us2.pusher.com/app/32cbd69e4b950bf97679"
	maxBackoff       = 32 * time.Second
	initialBack      = 1 * time.Second
)

func DownloadKickLiveVideo(ctx context.Context, video ent.Vod, channel ent.Channel, startChat chan bool) error {
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

	proxyEnabled := false                 // Whether to use a proxy
	proxyFound := false                   // Whether a proxy was found
	proxyType := utils.ProxyTypeTwitchHLS // The type of proxy to use, default is Twitch HLS
	proxyHeader := ""                     // The header to use for the proxy if found
	proxyHTTPUrl := ""                    // The http proxy URL to use if found

	url := utils.CreateKickURL(video.ExtID, video.Type, channel.Name)

	// Handle proxy setting
	proxyEnabled = config.Get().Livestream.ProxyEnabled
	whitelistedChannels := config.Get().Livestream.ProxyWhitelist // list of channels that are whitelisted from using proxy
	if proxyEnabled {
		if utils.Contains(whitelistedChannels, channel.Name) {
			log.Debug().Str("channel_name", channel.Name).Msg("channel is whitelisted, not using proxy")
		} else {
			// proxyParams := config.Get().Livestream.ProxyParameters
			proxyList := config.Get().Livestream.Proxies

			log.Debug().Str("proxy_list", fmt.Sprintf("%v", proxyList)).Msg("proxy list")

			// Test proxies - the first one that works will be used
			for _, proxy := range proxyList {
				// proxyUrl is url that will be sent to yt-dlp for download
				// this can be a direct URL or a proxy URL
				proxyUrl := url
				if proxy.ProxyType == utils.ProxyTypeTwitchHLS {
					log.Info().Str("channel_name", channel.Name).Str("proxy_url", proxy.URL).Msg("cannot use TwitchHLS proxy with Kick, skipping")
					continue
				}
				// Test the proxy server
				if testProxyServer(proxy.URL, proxyUrl, proxy.Header, proxy.ProxyType) {
					log.Debug().Str("channel_name", channel.Name).Str("proxy_url", proxy.URL).Msg("proxy found")
					proxyFound = true
					url = proxyUrl
					proxyHeader = proxy.Header
					proxyType = proxy.ProxyType
					if proxy.ProxyType == utils.ProxyTypeHTTP {
						proxyHTTPUrl = proxy.URL
					}
					break
				}
			}
		}
	}

	// Create yt-dlp service
	ytdlpSvc := ytdlp.NewYtDlpService(ytdlp.YtDlpOptions{})

	qualities, err := ytdlpSvc.GetVideoQualities(ctx, video)
	if err != nil {
		return fmt.Errorf("error getting video quality options: %w", err)
	}

	closestQuality := utils.SelectClosestQuality(video.Resolution, qualities)
	log.Info().Str("requested_quality", video.Resolution).Msgf("selected closest quality %s", closestQuality)

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
		"--no-part",
		"--no-warnings", "--progress", "--newline", "--no-check-certificate",
		"--hls-use-mpegts",
	)

	// Set proxy header if enabled
	if proxyHeader != "" {
		cmdArgs = append(cmdArgs, "--add-headers", proxyHeader)
	}

	// Set HTTP proxy if enabled
	if proxyFound && proxyType == utils.ProxyTypeHTTP {
		cmdArgs = append(cmdArgs, "--proxy", proxyHTTPUrl)
	}

	// Sanitize config args before appending
	for _, arg := range configYtDlpArgsArr {
		if strings.TrimSpace(arg) != "" {
			cmdArgs = append(cmdArgs, arg)
		}
	}

	// Create yt-dlp command
	// Only enable cookies if proxy is found - cookies are not set if proxy is used!
	// This means the quality requested may not be the one downloaded because of the proxy
	cmd, cookieFile, err := ytdlpSvc.CreateCommand(ctx, cmdArgs, !proxyFound)
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
			err := killYtDlp(cmd.Process.Pid)
			if err != nil {
				return fmt.Errorf("failed to kill yt-dlp process: %v", err)
			}
		}
		_, err := cmd.Process.Wait()
		if err != nil {
			log.Debug().Err(err).Msg("error waiting for yt-dlp process")
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

// connectToKickChatWebSocket connects to the Kick chat WebSocket and subscribes to the necessary channels.
func connnectToKickChatWebSocket(subscriptions []map[string]interface{}) (*websocket.Conn, error) {
	c, _, err := websocket.DefaultDialer.Dial(kickWebsocketURL, nil)
	if err != nil {
		return nil, err
	}
	for _, msg := range subscriptions {
		if err := c.WriteJSON(msg); err != nil {
			c.Close()
			return nil, fmt.Errorf("subscribe error: %w", err)
		}
	}
	return c, nil
}

// DownloadKickLiveChat downloads the live chat from Kick using WebSocket and saves it to a JSON file.
func DownloadKickLiveChat(ctx context.Context, video ent.Vod, channel ent.Channel, queue ent.Queue, chatRoomID string) error {
	env := config.GetEnvConfig()

	wsSubscriptions := []map[string]interface{}{
		{"event": "pusher:subscribe", "data": map[string]string{"auth": "", "channel": fmt.Sprintf("chatroom_%s", chatRoomID)}},
		{"event": "pusher:subscribe", "data": map[string]string{"auth": "", "channel": fmt.Sprintf("chatrooms.%s.v2", chatRoomID)}},
		{"event": "pusher:subscribe", "data": map[string]string{"auth": "", "channel": fmt.Sprintf("channel_%s", video.ExtID)}},
		{"event": "pusher:subscribe", "data": map[string]string{"auth": "", "channel": fmt.Sprintf("channel.%s", video.ExtID)}},
		{"event": "pusher:subscribe", "data": map[string]string{"auth": "", "channel": fmt.Sprintf("chatrooms.%s", chatRoomID)}},
	}
	log.Debug().Str("video_id", video.ID.String()).Msgf("subscribing to Kick WebSocket channels: %v", wsSubscriptions)

	// Record start time
	chatStart := time.Now()
	if _, err := queue.Update().SetChatStart(chatStart).Save(ctx); err != nil {
		return err
	}

	// Open log file
	logPath := fmt.Sprintf("%s/%s-chat.log", env.LogsDir, video.ID.String())
	lf, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer lf.Close()

	// Open JSON chat file
	jf, err := os.OpenFile(video.TmpLiveChatDownloadPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open json file: %w", err)
	}
	defer jf.Close()

	backoff := initialBack

outer:
	for {
		// Stop early if context is done
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Connect to Kick WebSocket
		conn, err := connnectToKickChatWebSocket(wsSubscriptions)
		if err != nil {
			log.Error().Err(err).Msg("connect error, retrying")
			fmt.Fprintf(lf, "connect error: %v\n", err)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}
		log.Info().Msgf("connected to Kick WebSocket for chat download, channel: %s", channel.Name)
		fmt.Fprintf(lf, "connected to Kick WebSocket\n")
		backoff = initialBack

		// Close connection when context is done
		done := make(chan struct{})
		go func() {
			<-ctx.Done()
			conn.Close()
			close(done)
		}()

		for {
			type result struct {
				msg []byte
				err error
			}
			ch := make(chan result, 1)
			go func() {
				_, m, e := conn.ReadMessage()
				ch <- result{m, e}
			}()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-done:
				return ctx.Err()
			case res := <-ch:
				if res.err != nil {
					log.Error().Err(res.err).Msg("read error, reconnecting")
					fmt.Fprintf(lf, "read error: %v\n", res.err)
					conn.Close()
					continue outer
				}
				var raw utils.KickWebSocketRawMsg
				if err := json.Unmarshal(res.msg, &raw); err != nil || raw.Event != "App\\Events\\ChatMessageEvent" {
					continue
				}
				// Parse the actual chat message
				blob, _ := json.Marshal(raw)
				jf.Write(blob)
				jf.Write([]byte("\n"))
			}
		}
	}
}
