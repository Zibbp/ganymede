package playlist_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/chapter"
	"github.com/zibbp/ganymede/internal/playlist"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
	"github.com/zibbp/ganymede/tests"
)

// setupAppAndSeed initializes the application and seeds it with a playlist, channel, and video.
func setupAppAndSeed(t *testing.T) (*server.Application, *ent.Playlist, *ent.Channel, *ent.Vod) {
	app, err := tests.Setup(t)
	assert.NoError(t, err, "Failed to setup test application")

	// Create a playlist
	playlist, err := app.PlaylistService.CreatePlaylist(t.Context(), playlist.Playlist{
		Name:        "Test Playlist",
		Description: "Playlist for testing",
	})
	assert.NoError(t, err, "Failed to create playlist")
	assert.NotNil(t, playlist, "Playlist should not be nil")

	// Create a channel
	channel, err := app.ChannelService.CreateChannel(channel.Channel{
		ExtID: "123456789",
		Name:  "TestChannel",
	})
	assert.NoError(t, err, "Failed to create channel")
	assert.NotNil(t, channel, "Channel should not be nil")

	// Create a video
	video, err := app.VodService.CreateVod(vod.Vod{
		ExtID:      "123456789",
		Platform:   utils.PlatformTwitch,
		Type:       utils.Archive,
		Title:      "Test video - Playing Baldur's Gate 3",
		Duration:   3600,
		Views:      1000,
		Resolution: string(utils.Best),
	}, channel.ID)
	assert.NoError(t, err, "Failed to create video")
	assert.NotNil(t, video, "Video should not be nil")

	// Add chapters to the video
	_, err = app.ChapterService.CreateChapter(chapter.Chapter{
		Type:  string(utils.ChapterTypeGameChange),
		Title: "Baldur's Gate 3",
		Start: 0,
		End:   3600,
	}, video.ID)
	assert.NoError(t, err, "Failed to create chapter")

	return app, playlist, channel, video
}

// TestShouldBeInPlaylist_TableDriven tests the ShouldVideoBeInPlaylist function with various rule groups.
// It tests all evaluation paths for rules
func TestShouldBeInPlaylist_TableDriven(t *testing.T) {
	app, seedPlaylist, _, seedVideo := setupAppAndSeed(t)
	video, err := app.Database.Client.Vod.Query().Where(entVod.ID(seedVideo.ID)).WithChapters().Only(t.Context())
	assert.NoError(t, err, "Failed to query video with chapters")
	assert.NotNil(t, video, "Video should not be nil")

	tests := []struct {
		name       string
		ruleGroups []playlist.RuleGroupInput
		expected   bool
	}{
		{
			name: "Title contains BG3",
			ruleGroups: []playlist.RuleGroupInput{
				{
					Operator: "OR",
					Position: 0,
					Rules: []playlist.RuleInput{
						{
							Name:     "Title contains BG3",
							Field:    utils.FieldTitle,
							Operator: utils.OperatorContains,
							Value:    "Baldur's Gate 3",
							Position: 0,
							Enabled:  true,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Title regex BG3",
			ruleGroups: []playlist.RuleGroupInput{
				{
					Operator: "OR",
					Position: 0,
					Rules: []playlist.RuleInput{
						{
							Name:     "Title regex BG3",
							Field:    utils.FieldTitle,
							Operator: utils.OperatorRegex,
							Value:    "(?i)(baldur's|gate\\s*3|BG3)",
							Position: 0,
							Enabled:  true,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Title contains BG3 and type is Archive",
			ruleGroups: []playlist.RuleGroupInput{
				{
					Operator: "AND",
					Position: 0,
					Rules: []playlist.RuleInput{
						{
							Name:     "Title contains BG3",
							Field:    utils.FieldTitle,
							Operator: utils.OperatorContains,
							Value:    "Baldur's Gate 3",
							Position: 0,
							Enabled:  true,
						},
						{
							Name:     "Type is Archive",
							Field:    utils.FieldType,
							Operator: utils.OperatorEquals,
							Value:    string(utils.Archive),
							Position: 1,
							Enabled:  true,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Title contains BG3 and type is Live (should not match)",
			ruleGroups: []playlist.RuleGroupInput{
				{
					Operator: "AND",
					Position: 0,
					Rules: []playlist.RuleInput{
						{
							Name:     "Title contains BG3",
							Field:    utils.FieldTitle,
							Operator: utils.OperatorContains,
							Value:    "Baldur's Gate 3",
							Position: 0,
							Enabled:  true,
						},
						{
							Name:     "Type is Live",
							Field:    utils.FieldType,
							Operator: utils.OperatorEquals,
							Value:    string(utils.Live),
							Position: 1,
							Enabled:  true,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Title contains BG3 and channel_name is foobar (should not match)",
			ruleGroups: []playlist.RuleGroupInput{
				{
					Operator: "AND",
					Position: 0,
					Rules: []playlist.RuleInput{
						{
							Name:     "Title contains BG3",
							Field:    utils.FieldTitle,
							Operator: utils.OperatorContains,
							Value:    "Baldur's Gate 3",
							Position: 0,
							Enabled:  true,
						},
						{
							Name:     "Channel name is foobar",
							Field:    utils.FieldChannelName,
							Operator: utils.OperatorEquals,
							Value:    "foobar",
							Position: 1,
							Enabled:  true,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Category contains Baldur's Gate 3",
			ruleGroups: []playlist.RuleGroupInput{
				{
					Operator: "AND",
					Position: 0,
					Rules: []playlist.RuleInput{
						{
							Name:     "Category contains BG3",
							Field:    utils.FieldCategory,
							Operator: utils.OperatorContains,
							Value:    "Baldur's Gate 3",
							Position: 0,
							Enabled:  true,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Two rule group - Category contains Baldur's Gate 3 and Title contains test",
			ruleGroups: []playlist.RuleGroupInput{
				{
					Operator: "AND",
					Position: 0,
					Rules: []playlist.RuleInput{
						{
							Name:     "Category contains BG3",
							Field:    utils.FieldCategory,
							Operator: utils.OperatorContains,
							Value:    "Baldur's Gate 3",
							Position: 0,
							Enabled:  true,
						},
					},
				},
				{
					Operator: "AND",
					Position: 1,
					Rules: []playlist.RuleInput{
						{
							Name:     "Title contains BG3",
							Field:    utils.FieldTitle,
							Operator: utils.OperatorContains,
							Value:    "Baldur's Gate 3",
							Position: 0,
							Enabled:  true,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Two rule group - Category equals Baldur's Gate 3 or Title contains foobar (should match)",
			ruleGroups: []playlist.RuleGroupInput{
				{
					Operator: "AND",
					Position: 0,
					Rules: []playlist.RuleInput{
						{
							Name:     "Category contains BG3",
							Field:    utils.FieldCategory,
							Operator: utils.OperatorEquals,
							Value:    "Baldur's Gate 3",
							Position: 0,
							Enabled:  true,
						},
					},
				},
				{
					Operator: "AND",
					Position: 1,
					Rules: []playlist.RuleInput{
						{
							Name:     "Title contains foobar",
							Field:    utils.FieldTitle,
							Operator: utils.OperatorContains,
							Value:    "foobar",
							Position: 0,
							Enabled:  true,
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createdGroups, err := app.PlaylistService.SetPlaylistRules(t.Context(), seedPlaylist.ID, tt.ruleGroups)
			assert.NoError(t, err, "Failed to set playlist rules")
			assert.Len(t, createdGroups, len(tt.ruleGroups), "Unexpected number of rule groups created")

			ruleGroups, err := app.PlaylistService.GetPlaylistRules(t.Context(), seedPlaylist.ID)
			assert.NoError(t, err, "Failed to get playlist rules")
			assert.Len(t, ruleGroups, len(tt.ruleGroups), "Unexpected number of rule groups returned")

			result, err := app.PlaylistService.ShouldVideoBeInPlaylist(t.Context(), video.ID, seedPlaylist.ID)
			assert.NoError(t, err, "Failed to evaluate video against playlist rules")
			assert.Equal(t, tt.expected, result, "Unexpected playlist inclusion result")
		})
	}
}
