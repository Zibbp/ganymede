package exec

import (
	"testing"

	"github.com/zibbp/ganymede/internal/utils"
)

func TestCreateTwitchURL(t *testing.T) {
	tests := []struct {
		name        string
		videoId     string
		videoType   utils.VodType
		channelName string
		want        string
	}{
		{
			name:        "Clip type",
			videoId:     "abc123",
			videoType:   utils.Clip,
			channelName: "testchannel",
			want:        "https://twitch.tv/testchannel/clip/abc123",
		},
		{
			name:        "Live type",
			videoId:     "liveid",
			videoType:   utils.Live,
			channelName: "livechannel",
			want:        "https://twitch.tv/livechannel",
		},
		{
			name:        "Default type",
			videoId:     "vod456",
			videoType:   utils.VodType("999"), // Unknown type
			channelName: "vodchannel",
			want:        "https://twitch.tv/videos/vod456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createTwitchURL(tt.videoId, tt.videoType, tt.channelName)
			if got != tt.want {
				t.Errorf("createTwitchURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
