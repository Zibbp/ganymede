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
		UUID:    testUUID,
		ID:      "testid",
		Channel: "testchannel",
		Title:   "Test Title",
		Type:    "video",
		Date:    "2025-08-04",
		YYYY:    "2025",
		MM:      "08",
		DD:      "04",
		HH:      "12",
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
		UUID:    testUUID,
		ID:      "testid",
		Channel: "testchannel",
		Title:   "Test Title",
		Type:    "video",
		Date:    "2025-08-04",
		YYYY:    "2025",
		MM:      "08",
		DD:      "04",
		HH:      "12",
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
