package utils

import "fmt"

// createTwitchURL generates a Twitch URL based on the video ID, video type, and channel name.
func CreateTwitchURL(videoId string, videoType VodType, channelName string) string {
	var url string
	switch videoType {
	case Clip:
		url = fmt.Sprintf("https://twitch.tv/%s/clip/%s", channelName, videoId)
	case Live:
		url = fmt.Sprintf("https://twitch.tv/%s", channelName)
	default:
		url = fmt.Sprintf("https://twitch.tv/videos/%s", videoId)

	}
	return url
}
