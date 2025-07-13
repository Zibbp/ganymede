package exec

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
	tests_shared "github.com/zibbp/ganymede/tests/shared"
)

// TestGetVideoQualities tests the GetVideoQualities function for various video types
func TestGetVideoQualities(t *testing.T) {
	testCases := []struct {
		videoId     string
		expectedLen int
		description string
		videoType   utils.VodType
		channelName string
	}{
		{
			videoId:     tests_shared.TestTwitchVideoId1,
			expectedLen: 6,
			description: "Standard archive VOD 1",
			videoType:   utils.Archive,
			channelName: tests_shared.TestTwitchVideoChannelName1,
		},
		{
			videoId:     tests_shared.TestTwitchClipId1,
			expectedLen: 7,
			description: "Clip",
			videoType:   utils.Clip,
			channelName: tests_shared.TestTwitchClipChannelName1,
		},
		{
			videoId:     tests_shared.TestTwitchVideoId2,
			expectedLen: 7,
			description: "Standard archive VOD 2",
			videoType:   utils.Archive,
			channelName: tests_shared.TestTwitchVideoChannelName2,
		},
		{
			videoId:     tests_shared.TestTwitchClipId2,
			expectedLen: 4,
			description: "Clip",
			videoType:   utils.Clip,
			channelName: tests_shared.TestTwitchClipChannelName2,
		}}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Setup VOD object
			v := ent.Vod{
				ExtID: tc.videoId,
				Type:  tc.videoType,
			}
			c := ent.Channel{}
			v.Edges.Channel = &c
			v.Edges.Channel.Name = tc.channelName

			qualities, err := GetVideoQualities(context.Background(), v)
			assert.NoError(t, err)

			t.Logf("Qualities for VOD %s: %v", tc.videoId, qualities)

			assert.Equal(t, tc.expectedLen, len(qualities),
				"Expected %d quality options for video %s, got %d",
				tc.expectedLen, tc.videoId, len(qualities))
		})
	}
}

// TestYtDlpGetVideoInfo tests the YtDlpGetVideoInfo function for various video types
func TestYtDlpGetVideoInfo(t *testing.T) {
	testCases := []struct {
		name          string
		video         ent.Vod
		expectedID    string
		expectedField string
	}{
		{
			name: "Twitch Archive VOD",
			video: ent.Vod{
				ExtID: tests_shared.TestTwitchVideoId1,
				Type:  utils.Archive,
			},
			expectedID:    fmt.Sprintf("v%s", tests_shared.TestTwitchVideoId1), // YtDlp prefixes Twitch VOD IDs with 'v'
			expectedField: "ID",
		},
		{
			name: "Twitch Clip",
			video: ent.Vod{
				ExtID: tests_shared.TestTwitchClipId1,
				Type:  utils.Clip,
				Edges: ent.VodEdges{Channel: &ent.Channel{Name: tests_shared.TestTwitchClipChannelName1}},
			},
			expectedID:    tests_shared.TestTwitchClipId1,
			expectedField: "DisplayID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := YtDlpGetVideoInfo(t.Context(), tc.video)
			assert.NoError(t, err, "Expected no error getting video info")
			assert.NotNil(t, info, "Expected video info to be non-nil")
			if tc.expectedField == "ID" {
				assert.Equal(t, tc.expectedID, info.ID, "Expected video ID to match")
			} else {
				assert.Equal(t, tc.expectedID, info.DisplayID, "Expected video DisplayID to match")
			}
			assert.Greater(t, info.Duration, int64(0), "Expected video duration to be greater than 0")
			assert.Greater(t, len(info.Formats), int(0), "Expected video formats to be greater than 0")
		})
	}
}
