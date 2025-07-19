package utils

import "testing"

func TestCreateTwitchURL(t *testing.T) {
	tests := []struct {
		name        string
		videoId     string
		videoType   VodType
		channelName string
		want        string
	}{
		{
			name:        "Clip type",
			videoId:     "abc123",
			videoType:   Clip,
			channelName: "testchannel",
			want:        "https://twitch.tv/testchannel/clip/abc123",
		},
		{
			name:        "Live type",
			videoId:     "liveid",
			videoType:   Live,
			channelName: "livechannel",
			want:        "https://twitch.tv/livechannel",
		},
		{
			name:        "Default type",
			videoId:     "vod456",
			videoType:   VodType("999"), // Unknown type
			channelName: "vodchannel",
			want:        "https://twitch.tv/videos/vod456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateTwitchURL(tt.videoId, tt.videoType, tt.channelName)
			if got != tt.want {
				t.Errorf("createTwitchURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
