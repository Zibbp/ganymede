package utils

import "testing"

func TestCreateKickURL(t *testing.T) {
	tests := []struct {
		name        string
		videoId     string
		videoType   VodType
		channelName string
		want        string
	}{
		{
			name:        "Clip type",
			videoId:     "clip_12345",
			videoType:   Clip,
			channelName: "testchannel",
			want:        "https://kick.com/testchannel/clips/clip_12345",
		},
		{
			name:        "Live type",
			videoId:     "liveid",
			videoType:   Live,
			channelName: "livechannel",
			want:        "https://kick.com/livechannel",
		},
		{
			name:        "Default type",
			videoId:     "vod456",
			videoType:   VodType("999"), // Unknown type
			channelName: "vodchannel",
			want:        "https://kick.com/vodchannel/videos/vod456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateKickURL(tt.videoId, tt.videoType, tt.channelName)
			if got != tt.want {
				t.Errorf("createKickURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
