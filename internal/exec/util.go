package exec

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/utils"
)

// killYtDlp attempts to gracefully terminate the yt-dlp process by sending a SIGINT to its process group.
func killYtDlp(pid int) error {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get process group ID for PID %d", pid)
		return fmt.Errorf("failed to get process group ID for PID %d: %w", pid, err)
	}
	log.Debug().Msgf("Process group ID for PID %d is %d", pid, pgid)
	return syscall.Kill(-pgid, syscall.SIGINT)
}

func testProxyServer(proxyURL string, testURL string, header string, proxyType utils.ProxyType) bool {
	switch proxyType {
	case utils.ProxyTypeTwitchHLS:
		return testTwitchHLSProxy(proxyURL, testURL, header)
	case utils.ProxyTypeHTTP:
		return testHTTPProxy(proxyURL, testURL, header)
	default:
		log.Error().Msgf("Unknown proxy type: %s", proxyType)
		return false
	}
}

func testTwitchHLSProxy(proxyURL string, testURL string, header string) bool {
	log.Debug().Msgf("testing Twitch HLS proxy server: %s", proxyURL)
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("error creating request for Twitch HLS proxy server test")
		return false
	}
	if header != "" {
		log.Debug().Msgf("adding header %s to Twitch HLS proxy server test", header)
		splitHeader := strings.SplitN(header, ":", 2)
		req.Header.Add(splitHeader[0], splitHeader[1])
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("error making request for Twitch HLS proxy server test")
		return false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing response body for Twitch HLS proxy server test")
		}
	}()
	if resp.StatusCode != 200 {
		log.Error().Msgf("Twitch HLS proxy server test returned status code %d", resp.StatusCode)
		return false
	}
	log.Debug().Msg("Twitch HLS proxy server test successful")
	return true
}

func testHTTPProxy(proxyURL string, testURL string, header string) bool {
	log.Debug().Msgf("testing HTTP proxy server: %s", proxyURL)
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		log.Error().Err(err).Msg("error parsing HTTP proxy URL")
		return false
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(parsedURL),
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("error creating request for HTTP proxy server test")
		return false
	}

	if header != "" {
		log.Debug().Msgf("adding header %s to HTTP proxy server test", header)
		splitHeader := strings.SplitN(header, ":", 2)
		req.Header.Add(splitHeader[0], splitHeader[1])
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("error making request for HTTP proxy server test")
		return false
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing response body for HTTP proxy server test")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("HTTP proxy server test returned status code %d", resp.StatusCode)
		return false
	}

	log.Debug().Msg("HTTP proxy server test successful")
	return true
}
