package archive_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/tests"
)

// TestGetFolderName tests the GetFolderName function with various templates and inputs.
func TestGetFolderName(t *testing.T) {
	// Setup the application
	_, err := tests.Setup(t)
	assert.NoError(t, err)

	// Setup test input
	testUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	input := archive.StorageTemplateInput{
		UUID:               testUUID,
		ID:                 "testid",
		Channel:            "testchannel",
		ChannelID:          "12345",
		ChannelDisplayName: "TestChannel",
		Title:              "Test Title",
		Type:               "video",
		Date:               "2025-08-04",
		YYYY:               "2025",
		MM:                 "08",
		DD:                 "04",
		HH:                 "12",
	}

	tests := []struct {
		name        string
		template    string
		expected    string
		expectError bool
	}{
		{
			name:        "default template",
			template:    "{{date}}-{{id}}-{{type}}-{{uuid}}",
			expected:    "2025-08-04-testid-video-123e4567-e89b-12d3-a456-426614174000",
			expectError: false,
		},
		{
			name:        "custom template with variables",
			template:    "{{channel}}_{{type}}_{{id}}_{{uuid}}",
			expected:    "testchannel_video_testid_123e4567-e89b-12d3-a456-426614174000",
			expectError: false,
		},
		{
			name:        "custom template with granular date",
			template:    "s{{YYYY}}{{MM}}-{{DD}}{{HH}} - {{title}}",
			expected:    "s202508-0412 - Test_Title",
			expectError: false,
		},
		{
			name:        "template with channel_id variable",
			template:    "{{channel_id}}-{{id}}-{{uuid}}",
			expected:    "12345-testid-123e4567-e89b-12d3-a456-426614174000",
			expectError: false,
		},
		{
			name:        "template with channel_display_name variable",
			template:    "{{channel_display_name}}-{{date}}-{{id}}",
			expected:    "TestChannel-2025-08-04-testid",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update config with the template
			if tt.template != "" {
				c := config.Get()
				c.StorageTemplates.FolderTemplate = tt.template
				assert.NoError(t, config.UpdateConfig(c), "failed to update config with template")
			}
			result, err := archive.GetFolderName(testUUID, input)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("GetFolderName() = %q, expected %q", result, tt.expected)
				}
			}
		})
	}
}

// TestGetFileName tests the GetFileName function with various templates and inputs.
func TestGetFileName(t *testing.T) {
	// Setup the application
	_, err := tests.Setup(t)
	assert.NoError(t, err)

	// Setup test input
	testUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	input := archive.StorageTemplateInput{
		UUID:               testUUID,
		ID:                 "testid",
		Channel:            "testchannel",
		ChannelID:          "12345",
		ChannelDisplayName: "TestChannel",
		Title:              "Test Title",
		Type:               "video",
		Date:               "2025-08-04",
		YYYY:               "2025",
		MM:                 "08",
		DD:                 "04",
		HH:                 "12",
	}

	tests := []struct {
		name        string
		template    string
		expected    string
		expectError bool
	}{
		{
			name:        "default template",
			template:    "{{id}}",
			expected:    "testid",
			expectError: false,
		},
		{
			name:        "custom template with variables",
			template:    "{{channel}}_{{type}}_{{id}}_{{uuid}}.mp4",
			expected:    "testchannel_video_testid_123e4567-e89b-12d3-a456-426614174000.mp4",
			expectError: false,
		},
		{
			name:        "custom template with granular date",
			template:    "s{{YYYY}}{{MM}}-{{DD}}{{HH}} - {{title}}.mp4",
			expected:    "s202508-0412 - Test_Title.mp4",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.template != "" {
				c := config.Get()
				c.StorageTemplates.FileTemplate = tt.template
				assert.NoError(t, config.UpdateConfig(c), "failed to update config with template")
			}
			result, err := archive.GetFileName(testUUID, input)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("GetFileName() = %q, expected %q", result, tt.expected)
				}
			}
		})
	}
}

// TestGetChannelFolderName tests the GetChannelFolderName function with various templates and inputs.
func TestGetChannelFolderName(t *testing.T) {
	// Setup the application
	_, err := tests.Setup(t)
	assert.NoError(t, err)

	// Setup test input
	channelInput := archive.ChannelTemplateInput{
		ChannelName:        "testchannel",
		ChannelID:          "12345",
		ChannelDisplayName: "TestChannel",
	}

	tests := []struct {
		name        string
		template    string
		expected    string
		expectError bool
	}{
		{
			name:        "default template (channel login name)",
			template:    "{{channel}}",
			expected:    "testchannel",
			expectError: false,
		},
		{
			name:        "channel ID template",
			template:    "{{channel_id}}",
			expected:    "12345",
			expectError: false,
		},
		{
			name:        "channel display name template",
			template:    "{{channel_display_name}}",
			expected:    "TestChannel",
			expectError: false,
		},
		{
			name:        "mixed template with ID and name",
			template:    "{{channel_id}}_{{channel}}",
			expected:    "12345_testchannel",
			expectError: false,
		},
		{
			name:        "invalid variable",
			template:    "{{invalid_var}}",
			expected:    "",
			expectError: true,
		},
		{
			name:        "path traversal in template literal is sanitized",
			template:    "../../etc",
			expected:    "etc",
			expectError: false,
		},
		{
			name:        "path separator in template literal is sanitized",
			template:    "foo/bar",
			expected:    "foo_bar",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.template != "" {
				c := config.Get()
				c.StorageTemplates.ChannelFolderTemplate = tt.template
				assert.NoError(t, config.UpdateConfig(c), "failed to update config with template")
			}
			result, err := archive.GetChannelFolderName(channelInput)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("GetChannelFolderName() = %q, expected %q", result, tt.expected)
				}
			}
		})
	}

	// Test with unsafe display name containing path traversal characters
	t.Run("display name with path traversal is sanitized", func(t *testing.T) {
		c := config.Get()
		c.StorageTemplates.ChannelFolderTemplate = "{{channel_display_name}}"
		assert.NoError(t, config.UpdateConfig(c), "failed to update config with template")

		unsafeInput := archive.ChannelTemplateInput{
			ChannelName:        "testchannel",
			ChannelID:          "12345",
			ChannelDisplayName: "../../etc/passwd",
		}
		result, err := archive.GetChannelFolderName(unsafeInput)
		assert.NoError(t, err)
		assert.NotContains(t, result, "..")
		assert.NotContains(t, result, "/")
	})
}
