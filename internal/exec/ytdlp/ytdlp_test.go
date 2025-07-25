package ytdlp

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCreateYtDlpCookiesFile tests the createYtDlpCookiesFile function.
func TestCreateYtDlpCookiesFile(t *testing.T) {
	ctx := context.Background()
	cookies := []YtDlpCookie{
		{
			Domain: ".twitch.tv",
			Name:   "auth-token",
			Value:  "test-token",
		},
		{
			Domain: ".youtube.com",
			Name:   "SID",
			Value:  "test-sid",
		},
	}

	file, err := createYtDlpCookiesFile(ctx, cookies)
	assert.NoError(t, err)
	defer assert.NoError(t, file.Close())

	data, err := os.ReadFile(file.Name())
	assert.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "# Netscape HTTP Cookie File")
	assert.Contains(t, content, ".twitch.tv")
	assert.Contains(t, content, "auth-token")
	assert.Contains(t, content, "test-token")
	assert.Contains(t, content, ".youtube.com")
	assert.Contains(t, content, "SID")
	assert.Contains(t, content, "test-sid")
}

// TestYtDlpService_CreateCommand tests the YtDlpService.CreateCommand method.
func TestYtDlpService_CreateCommand(t *testing.T) {
	ctx := context.Background()
	inputArgs := []string{"--version"}
	cookies := []YtDlpCookie{
		{
			Domain: ".example.com",
			Name:   "sessionid",
			Value:  "abc123",
		},
	}
	svc := NewYtDlpService(YtDlpOptions{
		Cookies: cookies,
	})

	cmd, cookiesFile, err := svc.CreateCommand(ctx, inputArgs, true)
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.NotNil(t, cookiesFile)

	args := cmd.Args
	assert.Contains(t, args, "--version")
	assert.Contains(t, args, "--cookies")
	assert.Contains(t, args, cookiesFile.Name())

	data, err := os.ReadFile(cookiesFile.Name())
	assert.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, ".example.com")
	assert.Contains(t, content, "sessionid")
	assert.Contains(t, content, "abc123")
}

// TestYtDlpService_CreateCommand_NoCookies tests CreateCommand with no cookies.
func TestYtDlpService_CreateCommand_NoCookies(t *testing.T) {
	ctx := context.Background()
	inputArgs := []string{"--help"}
	svc := NewYtDlpService(YtDlpOptions{})

	cmd, cookiesFile, err := svc.CreateCommand(ctx, inputArgs, true)
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Nil(t, cookiesFile)

	args := cmd.Args
	assert.Contains(t, args, "--help")
	assert.NotContains(t, args, "--cookies")
}

// TestYTDLPVideoInfo_CreateQualityOption tests the CreateQualityOption method on YTDLPVideoInfo.
func TestYTDLPVideoInfo_CreateQualityOption(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"best", "best"},
		{"audio", "bestaudio"},
		{"audio_only", "bestaudio"},
		{"1440p60__source_", "best[height=1440]/best"},
		{"1080p60__source_", "best[height=1080]/best"},
		{"720p60", "best[height=720]/best"},
		{"1080p30", "best[height=1080]/best"},
		{"720", "best[height=720]/best"},
		{"1080", "best[height=1080]/best"},
		{"480", "best[height=480]/best"},
		{"foo", "best[height<=?foo]/best"},
		{"360p", "best[height=360]/best"},
		{"", "best[height<=?]/best"},
	}

	svc := NewYtDlpService(YtDlpOptions{})

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := svc.CreateQualityOption(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
