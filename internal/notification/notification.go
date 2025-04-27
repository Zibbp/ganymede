package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/config"
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

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing response body")
		}
	}()

	return nil
}

func SendVideoArchiveSuccessNotification(channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue) {
	// Get notification settings
	videoSuccessWebhookUrl := config.Get().Notification.VideoSuccessWebhookUrl
	videoSuccessTemplate := config.Get().Notification.VideoSuccessTemplate
	videoSuccessEnabled := config.Get().Notification.VideoSuccessEnabled

	if (!videoSuccessEnabled) || (videoSuccessWebhookUrl == "") || (videoSuccessTemplate == "") {
		log.Debug().Msg("Video archive success notification is disabled")
		return
	}

	variableMap := getVariableMap(channelItem, vodItem, qItem, "", nil)

	res := notificationVariableRegex.FindAllStringSubmatch(videoSuccessTemplate, -1)
	for _, match := range res {
		// Get variable name
		variableName := match[1]
		// Get variable value
		variableValue := variableMap[variableName]
		// Replace variable in template
		variableValueString := fmt.Sprintf("%v", variableValue)
		videoSuccessTemplate = strings.ReplaceAll(videoSuccessTemplate, match[0], variableValueString)
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
	liveSuccessWebhookUrl := config.Get().Notification.LiveSuccessWebhookUrl
	liveSuccessTemplate := config.Get().Notification.LiveSuccessTemplate
	liveSuccessEnabled := config.Get().Notification.LiveSuccessEnabled

	if (!liveSuccessEnabled) || (liveSuccessWebhookUrl == "") || (liveSuccessTemplate == "") {
		log.Debug().Msg("Live archive success notification is disabled")
		return
	}

	variableMap := getVariableMap(channelItem, vodItem, qItem, "", nil)

	res := notificationVariableRegex.FindAllStringSubmatch(liveSuccessTemplate, -1)
	for _, match := range res {
		// Get variable name
		variableName := match[1]
		// Get variable value
		variableValue := variableMap[variableName]
		// Replace variable in template
		variableValueString := fmt.Sprintf("%v", variableValue)
		liveSuccessTemplate = strings.ReplaceAll(liveSuccessTemplate, match[0], variableValueString)

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
	errorWebhookUrl := config.Get().Notification.ErrorWebhookUrl
	errorTemplate := config.Get().Notification.ErrorTemplate
	errorEnabled := config.Get().Notification.ErrorEnabled

	if (!errorEnabled) || (errorWebhookUrl == "") || (errorTemplate == "") {
		log.Debug().Msg("Error notification is disabled")
		return
	}

	variableMap := getVariableMap(channelItem, vodItem, qItem, failedTask, nil)

	res := notificationVariableRegex.FindAllStringSubmatch(errorTemplate, -1)
	for _, match := range res {
		// Get variable name
		variableName := match[1]
		// Get variable value
		variableValue := variableMap[variableName]
		// Replace variable in template
		variableValueString := fmt.Sprintf("%v", variableValue)
		errorTemplate = strings.ReplaceAll(errorTemplate, match[0], variableValueString)

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

func SendLiveNotification(channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue, category string) {
	// Get notification settings
	liveWebhookUrl := config.Get().Notification.IsLiveWebhookUrl
	liveTemplate := config.Get().Notification.IsLiveTemplate
	liveEnabled := config.Get().Notification.IsLiveEnabled

	if (!liveEnabled) || (liveWebhookUrl == "") || (liveTemplate == "") {
		log.Debug().Msg("Live notification is disabled")
		return
	}

	variableMap := getVariableMap(channelItem, vodItem, qItem, "", &category)

	res := notificationVariableRegex.FindAllStringSubmatch(liveTemplate, -1)
	for _, match := range res {
		// Get variable name
		variableName := match[1]
		// Get variable value
		variableValue := variableMap[variableName]
		// Replace variable in template
		variableValueString := fmt.Sprintf("%v", variableValue)
		liveTemplate = strings.ReplaceAll(liveTemplate, match[0], variableValueString)

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

func getVariableMap(channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue, failedTask string, category *string) map[string]interface{} {
	categoryValue := ""
	if category != nil {
		categoryValue = *category
	}
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
		// Live stream
		"category": categoryValue,
	}
	return variables
}
