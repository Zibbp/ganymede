package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/ent"
)

var (
	notificationVariableRegex = regexp.MustCompile(`\{{([^}]+)\}}`)
)

type WebhookRequestBody struct {
	Content string `json:"content"`
	Body    string `json:"body"`
}

func sendWebhook(url string, body []byte) error {

	client := &http.Client{}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))

	if err != nil {
		return fmt.Errorf("error creating webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("error sending webhook request: %w", err)
	}

	defer resp.Body.Close()

	return nil
}

func SendVideoArchiveSuccessNotification(channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue) {
	// Get notification settings
	videoSuccessWebhookUrl := viper.GetString("notifications.video_success_webhook_url")
	videoSuccessTemplate := viper.GetString("notifications.video_success_template")
	videoSuccessEnabled := viper.GetBool("notifications.video_success_enabled")

	if (!videoSuccessEnabled) || (videoSuccessWebhookUrl == "") || (videoSuccessTemplate == "") {
		log.Debug().Msg("Video archive success notification is disabled")
		return
	}

	variableMap := getVariableMap(channelItem, vodItem, qItem, "")

	res := notificationVariableRegex.FindAllStringSubmatch(videoSuccessTemplate, -1)
	for _, match := range res {
		// Get variable name
		variableName := match[1]
		// Get variable value
		variableValue := variableMap[variableName]
		// Replace variable in template
		variableValueString := fmt.Sprintf("%v", variableValue)
		videoSuccessTemplate = strings.Replace(videoSuccessTemplate, match[0], variableValueString, -1)

	}

	var webhookRequestBody = WebhookRequestBody{
		Content: videoSuccessTemplate,
		Body:    videoSuccessTemplate,
	}

	body, err := json.Marshal(webhookRequestBody)
	if err != nil {
		log.Error().Err(err).Msg("error marshalling webhook request body")
		return
	}

	err = sendWebhook(videoSuccessWebhookUrl, body)
	if err != nil {
		log.Error().Err(err).Msg("error sending webhook")
		return
	}

}

func SendLiveArchiveSuccessNotification(channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue) {
	// Get notification settings
	liveSuccessWebhookUrl := viper.GetString("notifications.live_success_webhook_url")
	liveSuccessTemplate := viper.GetString("notifications.live_success_template")
	liveSuccessEnabled := viper.GetBool("notifications.live_success_enabled")

	if (!liveSuccessEnabled) || (liveSuccessWebhookUrl == "") || (liveSuccessTemplate == "") {
		log.Debug().Msg("Live archive success notification is disabled")
		return
	}

	variableMap := getVariableMap(channelItem, vodItem, qItem, "")

	res := notificationVariableRegex.FindAllStringSubmatch(liveSuccessTemplate, -1)
	for _, match := range res {
		// Get variable name
		variableName := match[1]
		// Get variable value
		variableValue := variableMap[variableName]
		// Replace variable in template
		variableValueString := fmt.Sprintf("%v", variableValue)
		liveSuccessTemplate = strings.Replace(liveSuccessTemplate, match[0], variableValueString, -1)

	}

	var webhookRequestBody = WebhookRequestBody{
		Content: liveSuccessTemplate,
		Body:    liveSuccessTemplate,
	}

	body, err := json.Marshal(webhookRequestBody)
	if err != nil {
		log.Error().Err(err).Msg("error marshalling webhook request body")
		return
	}

	err = sendWebhook(liveSuccessWebhookUrl, body)
	if err != nil {
		log.Error().Err(err).Msg("error sending webhook")
		return
	}

}

func SendErrorNotification(channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue, failedTask string) {
	// Get notification settings
	errorWebhookUrl := viper.GetString("notifications.error_webhook_url")
	errorTemplate := viper.GetString("notifications.error_template")
	errorEnabled := viper.GetBool("notifications.error_enabled")

	if (!errorEnabled) || (errorWebhookUrl == "") || (errorTemplate == "") {
		log.Debug().Msg("Error notification is disabled")
		return
	}

	variableMap := getVariableMap(channelItem, vodItem, qItem, failedTask)

	res := notificationVariableRegex.FindAllStringSubmatch(errorTemplate, -1)
	for _, match := range res {
		// Get variable name
		variableName := match[1]
		// Get variable value
		variableValue := variableMap[variableName]
		// Replace variable in template
		variableValueString := fmt.Sprintf("%v", variableValue)
		errorTemplate = strings.Replace(errorTemplate, match[0], variableValueString, -1)

	}

	var webhookRequestBody = WebhookRequestBody{
		Content: errorTemplate,
		Body:    errorTemplate,
	}

	body, err := json.Marshal(webhookRequestBody)
	if err != nil {
		log.Error().Err(err).Msg("error marshalling webhook request body")
		return
	}

	err = sendWebhook(errorWebhookUrl, body)
	if err != nil {
		log.Error().Err(err).Msg("error sending webhook")
		return
	}

}

func SendLiveNotification(channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue) {
	// Get notification settings
	liveWebhookUrl := viper.GetString("notifications.is_live_webhook_url")
	liveTemplate := viper.GetString("notifications.is_live_template")
	liveEnabled := viper.GetBool("notifications.is_live_enabled")

	if (!liveEnabled) || (liveWebhookUrl == "") || (liveTemplate == "") {
		log.Debug().Msg("Live notification is disabled")
		return
	}

	variableMap := getVariableMap(channelItem, vodItem, qItem, "")

	res := notificationVariableRegex.FindAllStringSubmatch(liveTemplate, -1)
	for _, match := range res {
		// Get variable name
		variableName := match[1]
		// Get variable value
		variableValue := variableMap[variableName]
		// Replace variable in template
		variableValueString := fmt.Sprintf("%v", variableValue)
		liveTemplate = strings.Replace(liveTemplate, match[0], variableValueString, -1)

	}

	var webhookRequestBody = WebhookRequestBody{
		Content: liveTemplate,
		Body:    liveTemplate,
	}

	body, err := json.Marshal(webhookRequestBody)
	if err != nil {
		log.Error().Err(err).Msg("error marshalling webhook request body")
		return
	}

	err = sendWebhook(liveWebhookUrl, body)
	if err != nil {
		log.Error().Err(err).Msg("error sending webhook")
		return
	}

}

func getVariableMap(channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue, failedTask string) map[string]interface{} {
	variables := map[string]interface{}{
		// Channel variables
		"channel_id":           channelItem.ID,
		"channel_ext_id":       channelItem.ExtID,
		"channel_display_name": channelItem.DisplayName,
		// Vod variables
		"vod_id":          vodItem.ID,
		"vod_ext_id":      vodItem.ExtID,
		"vod_platform":    vodItem.Platform,
		"vod_type":        vodItem.Type,
		"vod_title":       vodItem.Title,
		"vod_duration":    vodItem.Duration,
		"vod_views":       vodItem.Views,
		"vod_resolution":  vodItem.Resolution,
		"vod_streamed_at": vodItem.StreamedAt,
		"vod_created_at":  vodItem.CreatedAt,
		// Queue variables
		"queue_id":         qItem.ID,
		"queue_created_at": qItem.CreatedAt,
		// Error
		"failed_task": failedTask,
	}
	return variables
}
