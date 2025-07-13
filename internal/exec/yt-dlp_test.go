package exec

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zibbp/ganymede/internal/config"
)

// mockConfig and mockEnvConfig for testing
type mockParameters struct {
	TwitchToken string
}
type mockJSONConfig struct {
	Parameters mockParameters
}
type mockEnvConfig struct {
	TempDir string
}

// Patch config.Get and config.GetEnvConfig for testing
func setMockConfig(token string, tempDir string) (restore func()) {
	origGet := config.Get
	origGetEnv := config.GetEnvConfig

	config.Get = func() *config.JSONConfig {
		return &config.JSONConfig{
			Parameters: config.Parameters{
				TwitchToken: token,
			},
		}
	}
	config.GetEnvConfig = func() *config.EnvConfig {
		return &config.EnvConfig{
			TempDir: tempDir,
		}
	}
	return func() {
		config.Get = origGet
		config.GetEnvConfig = origGetEnv
	}
}

func TestCreateYtDlpCommand_NoToken(t *testing.T) {
	restore := setMockConfig("", "")
	defer restore()

	ctx := context.Background()
	args := []string{"--version"}
	cmd, err := createYtDlpCommand(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Path != "yt-dlp" {
		t.Errorf("expected yt-dlp command, got %s", cmd.Path)
	}
	if !strings.Contains(strings.Join(cmd.Args, " "), "--version") {
		t.Errorf("expected --version in args, got %v", cmd.Args)
	}
}

func TestCreateYtDlpCommand_WithToken(t *testing.T) {
	tempDir := t.TempDir()
	restore := setMockConfig("test-token", tempDir)
	defer restore()

	ctx := context.Background()
	args := []string{"--test-arg"}
	cmd, err := createYtDlpCommand(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for i, arg := range cmd.Args {
		if arg == "--cookies" && i+1 < len(cmd.Args) {
			cookieFile := cmd.Args[i+1]
			if !strings.HasPrefix(cookieFile, filepath.Join(tempDir, "cookies-")) {
				t.Errorf("cookie file not in temp dir: %s", cookieFile)
			}
			// Check file exists and content
			data, err := os.ReadFile(cookieFile)
			if err != nil {
				t.Errorf("failed to read cookie file: %v", err)
			}
			if !strings.Contains(string(data), "auth-token\ttest-token") {
				t.Errorf("cookie file missing token: %s", string(data))
			}
			found = true
		}
	}
	if !found {
		t.Errorf("--cookies argument not found in command args: %v", cmd.Args)
	}
}

func TestCreateYtDlpTwitchCookies(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()
	token := "my-token"
	file, err := createYtDlpTwitchCookies(ctx, tempDir, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(file.Name())

	data, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatalf("failed to read cookie file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, ".twitch.tv") {
		t.Errorf("cookie file missing .twitch.tv: %s", content)
	}
	if !strings.Contains(content, "auth-token\tmy-token") {
		t.Errorf("cookie file missing token: %s", content)
	}
}
