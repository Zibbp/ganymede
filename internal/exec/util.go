package exec

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/grafov/m3u8"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/utils"
)

func tryProxyServer(proxyURL string, testURL string, header string, proxyType utils.ProxyType) (*m3u8.MasterPlaylist, bool) {
	switch proxyType {
	case utils.ProxyTypeTwitchHLS:
		return tryTwitchHLSProxy(proxyURL, testURL, header)
	case utils.ProxyTypeHTTP:
		return tryHTTPProxy(proxyURL, testURL, header)
	default:
		log.Error().Msgf("Unknown proxy type: %s", proxyType)
		return nil, false
	}
}

func tryTwitchHLSProxy(proxyURL string, testURL string, header string) (*m3u8.MasterPlaylist, bool) {
	log.Debug().Msgf("testing Twitch HLS proxy server: %s", proxyURL)
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("error creating request for Twitch HLS proxy server test")
		return nil, false
	}
	if header != "" {
		log.Debug().Msgf("adding header %s to Twitch HLS proxy server test", header)
		splitHeader := strings.SplitN(header, ":", 2)
		req.Header.Add(splitHeader[0], splitHeader[1])
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("error making request for Twitch HLS proxy server test")
		return nil, false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing response body for Twitch HLS proxy server test")
		}
	}()
	if resp.StatusCode != 200 {
		log.Error().Msgf("Twitch HLS proxy server test returned status code %d", resp.StatusCode)
		return nil, false
	}

	playlist, _, err := m3u8.DecodeFrom(resp.Body, false)
	if err != nil {
		log.Error().Err(err).Msg("error decoding m3u8 response body for Twitch HLS proxy server test")
		return nil, false
	}

	masterPlaylist, ok := playlist.(*m3u8.MasterPlaylist)
	if !ok {
		log.Error().Msg("error casting playlist to a master playlist for Twitch HLS proxy server test")
		return nil, false
	}

	log.Debug().Msg("Twitch HLS proxy server test successful")
	return masterPlaylist, true
}

func tryHTTPProxy(proxyURL string, testURL string, header string) (*m3u8.MasterPlaylist, bool) {
	log.Debug().Msgf("testing HTTP proxy server: %s", proxyURL)
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		log.Error().Err(err).Msg("error parsing HTTP proxy URL")
		return nil, false
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
		return nil, false
	}

	if header != "" {
		log.Debug().Msgf("adding header %s to HTTP proxy server test", header)
		splitHeader := strings.SplitN(header, ":", 2)
		req.Header.Add(splitHeader[0], splitHeader[1])
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("error making request for HTTP proxy server test")
		return nil, false
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing response body for HTTP proxy server test")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("HTTP proxy server test returned status code %d", resp.StatusCode)
		return nil, false
	}

	playlist, _, err := m3u8.DecodeFrom(resp.Body, false)
	if err != nil {
		log.Error().Err(err).Msg("error decoding m3u8 response body for HTTP proxy server test")
		return nil, false
	}

	masterPlaylist, ok := playlist.(*m3u8.MasterPlaylist)
	if !ok {
		log.Error().Msg("error casting playlist to a master playlist for HTTP proxy server test")
		return nil, false
	}

	log.Debug().Msg("HTTP proxy server test successful")
	return masterPlaylist, true
}
