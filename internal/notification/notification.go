// Package notification provides a database-backed notification service
// that supports multiple named notification configurations with webhook and Apprise providers.
package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entNotification "github.com/zibbp/ganymede/ent/notification"
	"github.com/zibbp/ganymede/internal/database"
)

var (
	templateVariableRegex = regexp.MustCompile(`\{{([^}]+)\}}`)
)

// Service provides CRUD operations and dispatch logic for notifications.
type Service struct {
	Store      *database.Database
	httpClient *http.Client
}

// NewService creates a new notification service.
func NewService(store *database.Database) *Service {
	return &Service{
		Store: store,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// --- CRUD ---

// CreateNotification creates a new notification configuration.
func (s *Service) CreateNotification(ctx context.Context, n *ent.Notification) (*ent.Notification, error) {
	builder := s.Store.Client.Notification.Create().
		SetName(n.Name).
		SetEnabled(n.Enabled).
		SetType(n.Type).
		SetURL(n.URL).
		SetTriggerVideoSuccess(n.TriggerVideoSuccess).
		SetTriggerLiveSuccess(n.TriggerLiveSuccess).
		SetTriggerError(n.TriggerError).
		SetTriggerIsLive(n.TriggerIsLive).
		SetVideoSuccessTemplate(n.VideoSuccessTemplate).
		SetLiveSuccessTemplate(n.LiveSuccessTemplate).
		SetErrorTemplate(n.ErrorTemplate).
		SetIsLiveTemplate(n.IsLiveTemplate)

	// Set Apprise-specific fields
	if n.AppriseUrls != "" {
		builder.SetAppriseUrls(n.AppriseUrls)
	}
	if n.AppriseTitle != "" {
		builder.SetAppriseTitle(n.AppriseTitle)
	}
	if n.AppriseType != "" {
		builder.SetAppriseType(entNotification.AppriseType(n.AppriseType))
	}
	if n.AppriseTag != "" {
		builder.SetAppriseTag(n.AppriseTag)
	}
	if n.AppriseFormat != "" {
		builder.SetAppriseFormat(entNotification.AppriseFormat(n.AppriseFormat))
	}

	return builder.Save(ctx)
}

// GetNotification retrieves a single notification configuration by ID.
func (s *Service) GetNotification(ctx context.Context, id uuid.UUID) (*ent.Notification, error) {
	return s.Store.Client.Notification.Get(ctx, id)
}

// GetNotifications retrieves all notification configurations.
func (s *Service) GetNotifications(ctx context.Context) ([]*ent.Notification, error) {
	return s.Store.Client.Notification.Query().
		Order(ent.Asc(entNotification.FieldCreatedAt)).
		All(ctx)
}

// UpdateNotification updates an existing notification configuration.
func (s *Service) UpdateNotification(ctx context.Context, id uuid.UUID, n *ent.Notification) (*ent.Notification, error) {
	builder := s.Store.Client.Notification.UpdateOneID(id).
		SetName(n.Name).
		SetEnabled(n.Enabled).
		SetType(n.Type).
		SetURL(n.URL).
		SetTriggerVideoSuccess(n.TriggerVideoSuccess).
		SetTriggerLiveSuccess(n.TriggerLiveSuccess).
		SetTriggerError(n.TriggerError).
		SetTriggerIsLive(n.TriggerIsLive).
		SetVideoSuccessTemplate(n.VideoSuccessTemplate).
		SetLiveSuccessTemplate(n.LiveSuccessTemplate).
		SetErrorTemplate(n.ErrorTemplate).
		SetIsLiveTemplate(n.IsLiveTemplate).
		SetAppriseUrls(n.AppriseUrls).
		SetAppriseTitle(n.AppriseTitle).
		SetAppriseTag(n.AppriseTag)

	if n.AppriseType != "" {
		builder.SetAppriseType(entNotification.AppriseType(n.AppriseType))
	}
	if n.AppriseFormat != "" {
		builder.SetAppriseFormat(entNotification.AppriseFormat(n.AppriseFormat))
	}

	return builder.Save(ctx)
}

// DeleteNotification deletes a notification configuration by ID.
func (s *Service) DeleteNotification(ctx context.Context, id uuid.UUID) error {
	return s.Store.Client.Notification.DeleteOneID(id).Exec(ctx)
}

// --- Dispatch ---

// SendVideoArchiveSuccess sends notifications to all enabled configs with trigger_video_success.
func (s *Service) SendVideoArchiveSuccess(ctx context.Context, channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue) {
	notifications, err := s.Store.Client.Notification.Query().
		Where(
			entNotification.EnabledEQ(true),
			entNotification.TriggerVideoSuccessEQ(true),
		).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error querying video success notifications")
		return
	}

	variableMap := getVariableMap(channelItem, vodItem, qItem, "", nil)

	for _, n := range notifications {
		body := renderTemplate(n.VideoSuccessTemplate, variableMap)
		if err := s.send(n, body, variableMap); err != nil {
			log.Error().Err(err).Str("notification_id", n.ID.String()).Str("name", n.Name).Msg("error sending video success notification")
		}
	}
}

// SendLiveArchiveSuccess sends notifications to all enabled configs with trigger_live_success.
func (s *Service) SendLiveArchiveSuccess(ctx context.Context, channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue) {
	notifications, err := s.Store.Client.Notification.Query().
		Where(
			entNotification.EnabledEQ(true),
			entNotification.TriggerLiveSuccessEQ(true),
		).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error querying live success notifications")
		return
	}

	variableMap := getVariableMap(channelItem, vodItem, qItem, "", nil)

	for _, n := range notifications {
		body := renderTemplate(n.LiveSuccessTemplate, variableMap)
		if err := s.send(n, body, variableMap); err != nil {
			log.Error().Err(err).Str("notification_id", n.ID.String()).Str("name", n.Name).Msg("error sending live success notification")
		}
	}
}

// SendError sends notifications to all enabled configs with trigger_error.
func (s *Service) SendError(ctx context.Context, channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue, failedTask string) {
	notifications, err := s.Store.Client.Notification.Query().
		Where(
			entNotification.EnabledEQ(true),
			entNotification.TriggerErrorEQ(true),
		).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error querying error notifications")
		return
	}

	variableMap := getVariableMap(channelItem, vodItem, qItem, failedTask, nil)

	for _, n := range notifications {
		body := renderTemplate(n.ErrorTemplate, variableMap)
		if err := s.send(n, body, variableMap); err != nil {
			log.Error().Err(err).Str("notification_id", n.ID.String()).Str("name", n.Name).Msg("error sending error notification")
		}
	}
}

// SendLive sends notifications to all enabled configs with trigger_is_live.
func (s *Service) SendLive(ctx context.Context, channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue, category string) {
	notifications, err := s.Store.Client.Notification.Query().
		Where(
			entNotification.EnabledEQ(true),
			entNotification.TriggerIsLiveEQ(true),
		).All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error querying is-live notifications")
		return
	}

	variableMap := getVariableMap(channelItem, vodItem, qItem, "", &category)

	for _, n := range notifications {
		body := renderTemplate(n.IsLiveTemplate, variableMap)
		if err := s.send(n, body, variableMap); err != nil {
			log.Error().Err(err).Str("notification_id", n.ID.String()).Str("name", n.Name).Msg("error sending is-live notification")
		}
	}
}

// SendTestNotification sends a test notification using the config's own templates with dummy data.
func (s *Service) SendTestNotification(n *ent.Notification, eventType string) error {
	variableMap := getTestVariableMap()

	var tmpl string
	switch eventType {
	case "video_success":
		tmpl = n.VideoSuccessTemplate
	case "live_success":
		tmpl = n.LiveSuccessTemplate
	case "error":
		variableMap["failed_task"] = "video_download"
		tmpl = n.ErrorTemplate
	case "is_live":
		variableMap["category"] = "Demo Game"
		tmpl = n.IsLiveTemplate
	default:
		return fmt.Errorf("unknown test notification event type: %s", eventType)
	}

	body := renderTemplate(tmpl, variableMap)
	return s.send(n, body, variableMap)
}

// --- Internal ---

// send dispatches a notification based on its provider type.
// variableMap is optional — when provided, it is used to render Apprise title templates dynamically.
func (s *Service) send(n *ent.Notification, body string, variableMap map[string]interface{}) error {
	switch n.Type {
	case entNotification.TypeWebhook:
		return s.sendWebhook(n.URL, body)
	case entNotification.TypeApprise:
		return s.sendAppriseWithTitle(n, body, variableMap)
	default:
		return fmt.Errorf("unknown notification provider type: %s", string(n.Type))
	}
}

// webhookRequestBody is the JSON payload for simple webhook notifications.
type webhookRequestBody struct {
	Content string `json:"content"`
	Body    string `json:"body"`
}

// sendWebhook posts a JSON body to the webhook URL.
func (s *Service) sendWebhook(url string, body string) error {
	payload := webhookRequestBody{
		Content: body,
		Body:    body,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling webhook request body: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("error creating webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending webhook request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// appriseRequestBody is the JSON payload for Apprise API notifications.
type appriseRequestBody struct {
	URLs   string `json:"urls,omitempty"`
	Body   string `json:"body"`
	Title  string `json:"title,omitempty"`
	Type   string `json:"type,omitempty"`
	Tag    string `json:"tag,omitempty"`
	Format string `json:"format,omitempty"`
}

// sendAppriseWithTitle is used by dispatch methods when the variable map is available
// to render the Apprise title template dynamically.
func (s *Service) sendAppriseWithTitle(n *ent.Notification, body string, variableMap map[string]interface{}) error {
	payload := appriseRequestBody{
		Body: body,
	}

	if n.AppriseUrls != "" {
		payload.URLs = n.AppriseUrls
	}
	if n.AppriseTitle != "" {
		payload.Title = renderTemplate(n.AppriseTitle, variableMap)
	}
	if n.AppriseType != "" {
		payload.Type = string(n.AppriseType)
	}
	if n.AppriseTag != "" {
		payload.Tag = n.AppriseTag
	}
	if n.AppriseFormat != "" {
		payload.Format = string(n.AppriseFormat)
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling apprise request body: %w", err)
	}

	req, err := http.NewRequest("POST", n.URL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("error creating apprise request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending apprise request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("apprise returned status %d", resp.StatusCode)
	}

	return nil
}

// --- Template rendering ---

// renderTemplate replaces all {{variable}} placeholders in the template with values from the variable map.
// Unknown variables are left untouched in the output.
func renderTemplate(tmpl string, variableMap map[string]interface{}) string {
	res := templateVariableRegex.FindAllStringSubmatch(tmpl, -1)
	for _, match := range res {
		variableName := strings.TrimSpace(match[1])
		variableValue, ok := variableMap[variableName]
		if !ok || variableValue == nil {
			// Leave unknown or nil variables untouched
			continue
		}
		variableValueString := fmt.Sprintf("%v", variableValue)
		tmpl = strings.ReplaceAll(tmpl, match[0], variableValueString)
	}
	return tmpl
}

// getVariableMap builds a map of template variables from the provided entities.
// Nil entities are handled gracefully — their variables will be empty strings.
func getVariableMap(channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue, failedTask string, category *string) map[string]interface{} {
	categoryValue := ""
	if category != nil {
		categoryValue = *category
	}
	variables := map[string]interface{}{
		// Error
		"failed_task": failedTask,
		// Live stream
		"category": categoryValue,
	}

	// Channel variables
	if channelItem != nil {
		variables["channel_id"] = channelItem.ID
		variables["channel_ext_id"] = channelItem.ExtID
		variables["channel_display_name"] = channelItem.DisplayName
	}

	// Vod variables
	if vodItem != nil {
		variables["vod_id"] = vodItem.ID
		variables["vod_ext_id"] = vodItem.ExtID
		variables["vod_platform"] = vodItem.Platform
		variables["vod_type"] = vodItem.Type
		variables["vod_title"] = vodItem.Title
		variables["vod_duration"] = vodItem.Duration
		variables["vod_views"] = vodItem.Views
		variables["vod_resolution"] = vodItem.Resolution
		variables["vod_streamed_at"] = vodItem.StreamedAt
		variables["vod_created_at"] = vodItem.CreatedAt
	}

	// Queue variables
	if qItem != nil {
		variables["queue_id"] = qItem.ID
		variables["queue_created_at"] = qItem.CreatedAt
	}

	return variables
}

// getTestVariableMap builds a variable map with dummy test data.
func getTestVariableMap() map[string]interface{} {
	return map[string]interface{}{
		"channel_id":           uuid.New(),
		"channel_ext_id":       "1234456789",
		"channel_display_name": "Test Channel",
		"vod_id":               uuid.New(),
		"vod_ext_id":           "987654321",
		"vod_platform":         "twitch",
		"vod_type":             "archive",
		"vod_title":            "Demo Notification Title",
		"vod_duration":         100,
		"vod_views":            4510,
		"vod_resolution":       "best",
		"vod_streamed_at":      time.Now(),
		"vod_created_at":       time.Now(),
		"queue_id":             uuid.New(),
		"queue_created_at":     time.Now(),
		"failed_task":          "",
		"category":             "",
	}
}
