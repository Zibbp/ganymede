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
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entNotification "github.com/zibbp/ganymede/ent/notification"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
)

var (
	templateVariableRegex = regexp.MustCompile(`\{{([^}]+)\}}`)
)

const (
	notificationMaxAttempts = 3
	notificationRetryDelay  = 2 * time.Second
)

// redactURL masks the path and query of a URL to avoid leaking secrets (e.g. webhook tokens).
// Returns "scheme://host/***" or the first 12 characters if parsing fails.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		if len(raw) > 12 {
			return raw[:12] + "***"
		}
		return "***"
	}
	return u.Scheme + "://" + u.Host + "/***"
}

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
		SetIsLiveTemplate(n.IsLiveTemplate).
		SetAppriseUrls(n.AppriseUrls).
		SetAppriseTitle(n.AppriseTitle).
		SetAppriseTag(n.AppriseTag)

	// Enum fields: only set when non-empty so the DB default applies on create
	if n.AppriseType != "" {
		builder.SetAppriseType(entNotification.AppriseType(n.AppriseType))
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

// MigrateFromLegacyConfig migrates old config.json notification settings to the database.
// It groups triggers by webhook URL — if multiple triggers share the same URL, they become
// one notification config. This is idempotent: it skips migration if any notification configs
// already exist in the database.
func (s *Service) MigrateFromLegacyConfig(ctx context.Context, legacy *config.LegacyNotification) error {
	// Run the entire migration in a transaction so the count check and all
	// creates are atomic — a partial failure won't leave orphaned rows that
	// cause subsequent runs to skip migration.
	tx, err := s.Store.Client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("error starting migration transaction: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// Skip if there are already notification configs in the database
	count, err := tx.Notification.Query().Count(ctx)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("error checking existing notifications: %w", err)
	}
	if count > 0 {
		_ = tx.Rollback()
		log.Debug().Msg("notification configs already exist, skipping legacy migration")
		return nil
	}

	// Group triggers by webhook URL
	type triggerGroup struct {
		name                 string
		triggerVideoSuccess  bool
		videoSuccessTemplate string
		triggerLiveSuccess   bool
		liveSuccessTemplate  string
		triggerError         bool
		errorTemplate        string
		triggerIsLive        bool
		isLiveTemplate       string
	}

	groups := make(map[string]*triggerGroup)

	addTrigger := func(url string, label string, apply func(g *triggerGroup)) {
		if url == "" {
			return
		}
		g, ok := groups[url]
		if !ok {
			g = &triggerGroup{name: label}
			groups[url] = g
		}
		apply(g)
	}

	if legacy.VideoSuccessEnabled && legacy.VideoSuccessWebhookUrl != "" {
		addTrigger(legacy.VideoSuccessWebhookUrl, "Video Success", func(g *triggerGroup) {
			g.triggerVideoSuccess = true
			g.videoSuccessTemplate = legacy.VideoSuccessTemplate
		})
	}
	if legacy.LiveSuccessEnabled && legacy.LiveSuccessWebhookUrl != "" {
		addTrigger(legacy.LiveSuccessWebhookUrl, "Live Success", func(g *triggerGroup) {
			g.triggerLiveSuccess = true
			g.liveSuccessTemplate = legacy.LiveSuccessTemplate
		})
	}
	if legacy.ErrorEnabled && legacy.ErrorWebhookUrl != "" {
		addTrigger(legacy.ErrorWebhookUrl, "Error", func(g *triggerGroup) {
			g.triggerError = true
			g.errorTemplate = legacy.ErrorTemplate
		})
	}
	if legacy.IsLiveEnabled && legacy.IsLiveWebhookUrl != "" {
		addTrigger(legacy.IsLiveWebhookUrl, "Is Live", func(g *triggerGroup) {
			g.triggerIsLive = true
			g.isLiveTemplate = legacy.IsLiveTemplate
		})
	}

	if len(groups) == 0 {
		_ = tx.Rollback()
		log.Debug().Msg("no legacy notifications to migrate")
		return nil
	}

	for url, g := range groups {
		// Build a descriptive name
		name := "Migrated Webhook"
		if len(groups) > 1 {
			name = "Migrated: " + g.name
		}

		builder := tx.Notification.Create().
			SetName(name).
			SetEnabled(true).
			SetType(entNotification.TypeWebhook).
			SetURL(url).
			SetTriggerVideoSuccess(g.triggerVideoSuccess).
			SetTriggerLiveSuccess(g.triggerLiveSuccess).
			SetTriggerError(g.triggerError).
			SetTriggerIsLive(g.triggerIsLive)

		if g.videoSuccessTemplate != "" {
			builder.SetVideoSuccessTemplate(g.videoSuccessTemplate)
		}
		if g.liveSuccessTemplate != "" {
			builder.SetLiveSuccessTemplate(g.liveSuccessTemplate)
		}
		if g.errorTemplate != "" {
			builder.SetErrorTemplate(g.errorTemplate)
		}
		if g.isLiveTemplate != "" {
			builder.SetIsLiveTemplate(g.isLiveTemplate)
		}

		if _, err := builder.Save(ctx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("error creating migrated notification for %s: %w", redactURL(url), err)
		}

		log.Info().Str("name", name).Str("url", redactURL(url)).Msg("migrated legacy notification to database")
	}

	log.Info().Int("count", len(groups)).Msg("legacy notification migration complete")
	return tx.Commit()
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
		if err := s.send(ctx, n, body, variableMap); err != nil {
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
		if err := s.send(ctx, n, body, variableMap); err != nil {
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
		if err := s.send(ctx, n, body, variableMap); err != nil {
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
		if err := s.send(ctx, n, body, variableMap); err != nil {
			log.Error().Err(err).Str("notification_id", n.ID.String()).Str("name", n.Name).Msg("error sending is-live notification")
		}
	}
}

// SendTestNotification sends a test notification using the config's own templates with dummy data.
func (s *Service) SendTestNotification(ctx context.Context, n *ent.Notification, eventType string) error {
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
	return s.send(ctx, n, body, variableMap)
}

// --- Internal ---

// send dispatches a notification based on its provider type.
// variableMap is optional — when provided, it is used to render Apprise title templates dynamically.
func (s *Service) send(ctx context.Context, n *ent.Notification, body string, variableMap map[string]interface{}) error {
	switch n.Type {
	case entNotification.TypeWebhook:
		return s.sendWebhook(ctx, n.URL, body)
	case entNotification.TypeApprise:
		return s.sendAppriseWithTitle(ctx, n, body, variableMap)
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
func (s *Service) sendWebhook(ctx context.Context, url string, body string) error {
	payload := webhookRequestBody{
		Content: body,
		Body:    body,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling webhook request body: %w", err)
	}

	return s.postJSONWithRetry(ctx, url, jsonBody, "webhook")
}

// postJSONWithRetry sends a POST request with JSON content and retries on failures.
func (s *Service) postJSONWithRetry(ctx context.Context, targetURL string, jsonBody []byte, provider string) error {
	var lastErr error

	for attempt := 1; attempt <= notificationMaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return fmt.Errorf("%s request canceled before attempt %d: %w", provider, attempt, ctx.Err())
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(jsonBody))
		if err != nil {
			return fmt.Errorf("error creating %s request: %w", provider, err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("error sending %s request: %w", provider, err)
		} else {
			func() {
				defer func() {
					_, _ = io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}()

				if resp.StatusCode >= 400 {
					lastErr = fmt.Errorf("%s returned status %d", provider, resp.StatusCode)
					return
				}

				lastErr = nil
			}()

			if lastErr == nil {
				return nil
			}
		}

		if attempt == notificationMaxAttempts {
			break
		}

		retryDelay := notificationRetryDelay * time.Duration(attempt)
		log.Warn().
			Err(lastErr).
			Str("provider", provider).
			Str("url", redactURL(targetURL)).
			Int("attempt", attempt).
			Int("max_attempts", notificationMaxAttempts).
			Dur("retry_in", retryDelay).
			Msg("notification request failed, retrying")

		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return fmt.Errorf("%s request canceled while waiting to retry: %w", provider, ctx.Err())
		case <-timer.C:
		}
	}

	return fmt.Errorf("%s request failed after %d attempts: %w", provider, notificationMaxAttempts, lastErr)
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
func (s *Service) sendAppriseWithTitle(ctx context.Context, n *ent.Notification, body string, variableMap map[string]interface{}) error {
	payload := appriseRequestBody{
		Body: body,
	}

	if n.AppriseUrls != "" {
		if variableMap != nil {
			payload.URLs = renderTemplate(n.AppriseUrls, variableMap)
		} else {
			payload.URLs = n.AppriseUrls
		}
	}
	if n.AppriseTitle != "" {
		if variableMap != nil {
			payload.Title = renderTemplate(n.AppriseTitle, variableMap)
		} else {
			payload.Title = n.AppriseTitle
		}
	}
	if n.AppriseType != "" {
		payload.Type = string(n.AppriseType)
	}
	if n.AppriseTag != "" {
		if variableMap != nil {
			payload.Tag = renderTemplate(n.AppriseTag, variableMap)
		} else {
			payload.Tag = n.AppriseTag
		}
	}
	if n.AppriseFormat != "" {
		payload.Format = string(n.AppriseFormat)
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling apprise request body: %w", err)
	}

	return s.postJSONWithRetry(ctx, n.URL, jsonBody, "apprise")
}

// --- Template rendering ---

// renderTemplate replaces all {{variable}} placeholders in the template with values from the variable map.
// Each placeholder is resolved exactly once in a single left-to-right pass — replacement values
// that happen to contain {{...}} are never re-processed.
// Unknown or nil variables are left untouched in the output.
func renderTemplate(tmpl string, variableMap map[string]interface{}) string {
	return templateVariableRegex.ReplaceAllStringFunc(tmpl, func(match string) string {
		// match is guaranteed to be "{{...}}" by the regex — strip delimiters and trim whitespace
		variableName := strings.TrimSpace(match[2 : len(match)-2])
		variableValue, ok := variableMap[variableName]
		if !ok || variableValue == nil {
			return match
		}
		return fmt.Sprintf("%v", variableValue)
	})
}

// formatTime formats a time.Time to RFC3339 for template rendering.
// Zero times are returned as an empty string to avoid printing Go zero-value timestamps.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// getVariableMap builds a map of template variables from the provided entities.
// All known keys are always initialized with type-appropriate defaults (empty strings for
// string/time fields, 0 for numeric fields) so that renderTemplate can replace them even
// when entities are missing.
func getVariableMap(channelItem *ent.Channel, vodItem *ent.Vod, qItem *ent.Queue, failedTask string, category *string) map[string]interface{} {
	categoryValue := ""
	if category != nil {
		categoryValue = *category
	}

	// Initialize all keys with type-appropriate defaults
	variables := map[string]interface{}{
		// Error
		"failed_task": failedTask,
		// Live stream
		"category": categoryValue,
		// Channel (strings)
		"channel_id":           "",
		"channel_ext_id":       "",
		"channel_display_name": "",
		// Vod (strings, numeric, times)
		"vod_id":          "",
		"vod_ext_id":      "",
		"vod_platform":    "",
		"vod_type":        "",
		"vod_title":       "",
		"vod_duration":    0,
		"vod_views":       0,
		"vod_resolution":  "",
		"vod_streamed_at": "",
		"vod_created_at":  "",
		// Queue
		"queue_id":         "",
		"queue_created_at": "",
	}

	// Overwrite with real values when entities are present
	if channelItem != nil {
		variables["channel_id"] = channelItem.ID
		variables["channel_ext_id"] = channelItem.ExtID
		variables["channel_display_name"] = channelItem.DisplayName
	}

	if vodItem != nil {
		variables["vod_id"] = vodItem.ID
		variables["vod_ext_id"] = vodItem.ExtID
		variables["vod_platform"] = vodItem.Platform
		variables["vod_type"] = vodItem.Type
		variables["vod_title"] = vodItem.Title
		variables["vod_duration"] = vodItem.Duration
		variables["vod_views"] = vodItem.Views
		variables["vod_resolution"] = vodItem.Resolution
		variables["vod_streamed_at"] = formatTime(vodItem.StreamedAt)
		variables["vod_created_at"] = formatTime(vodItem.CreatedAt)
	}

	if qItem != nil {
		variables["queue_id"] = qItem.ID
		variables["queue_created_at"] = formatTime(qItem.CreatedAt)
	}

	return variables
}

// getTestVariableMap builds a variable map with dummy test data.
// Note: "failed_task" and "category" are left empty here — SendTestNotification
// overwrites them with test values for the relevant event types.
func getTestVariableMap() map[string]interface{} {
	now := formatTime(time.Now())
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
		"vod_streamed_at":      now,
		"vod_created_at":       now,
		"queue_id":             uuid.New(),
		"queue_created_at":     now,
		"failed_task":          "",
		"category":             "",
	}
}
