package exec

import (
	"fmt"

	"github.com/zibbp/ganymede/internal/utils"
)

// createTwitchURL generates a Twitch URL based on the video ID, video type, and channel name.
func createTwitchURL(videoId string, videoType utils.VodType, channelName string) string {
	var url string
	switch videoType {
	case utils.Clip:
		url = fmt.Sprintf("https://twitch.tv/%s/clip/%s", channelName, videoId)
	case utils.Live:
		url = fmt.Sprintf("https://twitch.tv/%s", channelName)
	default:
		url = fmt.Sprintf("https://twitch.tv/videos/%s", videoId)

	}
	return url
}
