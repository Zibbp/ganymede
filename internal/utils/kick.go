package utils

import "fmt"

// CreateKickURL generates a Kick URL based on the video ID, video type, and channel name.
func CreateKickURL(videoId string, videoType VodType, channelName string) string {
	var url string
	switch videoType {
	case Clip:
		url = fmt.Sprintf("https://kick.com/%s/clips/%s", channelName, videoId)
	case Live:
		url = fmt.Sprintf("https://kick.com/%s", channelName)
	default:
		url = fmt.Sprintf("https://kick.com/%s/videos/%s", channelName, videoId)

	}
	return url
}
