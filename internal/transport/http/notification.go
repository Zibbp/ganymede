package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entNotification "github.com/zibbp/ganymede/ent/notification"
)

// NotificationService defines the interface for notification operations.
type NotificationService interface {
	CreateNotification(ctx context.Context, n *ent.Notification) (*ent.Notification, error)
	GetNotification(ctx context.Context, id uuid.UUID) (*ent.Notification, error)
	GetNotifications(ctx context.Context) ([]*ent.Notification, error)
	UpdateNotification(ctx context.Context, id uuid.UUID, n *ent.Notification) (*ent.Notification, error)
	DeleteNotification(ctx context.Context, id uuid.UUID) error
	SendTestNotification(n *ent.Notification, eventType string) error
}

// CreateNotificationRequest is the request body for creating a notification.
type CreateNotificationRequest struct {
	Name                 string `json:"name" validate:"required"`
	Enabled              bool   `json:"enabled"`
	Type                 string `json:"type" validate:"required,oneof=webhook apprise"`
	URL                  string `json:"url" validate:"required,http_url"`
	TriggerVideoSuccess  bool   `json:"trigger_video_success"`
	TriggerLiveSuccess   bool   `json:"trigger_live_success"`
	TriggerError         bool   `json:"trigger_error"`
	TriggerIsLive        bool   `json:"trigger_is_live"`
	VideoSuccessTemplate string `json:"video_success_template"`
	LiveSuccessTemplate  string `json:"live_success_template"`
	ErrorTemplate        string `json:"error_template"`
	IsLiveTemplate       string `json:"is_live_template"`
	AppriseUrls          string `json:"apprise_urls"`
	AppriseTitle         string `json:"apprise_title"`
	AppriseType          string `json:"apprise_type" validate:"omitempty,oneof=info success warning failure"`
	AppriseTag           string `json:"apprise_tag"`
	AppriseFormat        string `json:"apprise_format" validate:"omitempty,oneof=text html markdown"`
}

// UpdateNotificationRequest is the request body for updating a notification.
type UpdateNotificationRequest struct {
	Name                 string `json:"name" validate:"required"`
	Enabled              bool   `json:"enabled"`
	Type                 string `json:"type" validate:"required,oneof=webhook apprise"`
	URL                  string `json:"url" validate:"required,http_url"`
	TriggerVideoSuccess  bool   `json:"trigger_video_success"`
	TriggerLiveSuccess   bool   `json:"trigger_live_success"`
	TriggerError         bool   `json:"trigger_error"`
	TriggerIsLive        bool   `json:"trigger_is_live"`
	VideoSuccessTemplate string `json:"video_success_template"`
	LiveSuccessTemplate  string `json:"live_success_template"`
	ErrorTemplate        string `json:"error_template"`
	IsLiveTemplate       string `json:"is_live_template"`
	AppriseUrls          string `json:"apprise_urls"`
	AppriseTitle         string `json:"apprise_title"`
	AppriseType          string `json:"apprise_type" validate:"omitempty,oneof=info success warning failure"`
	AppriseTag           string `json:"apprise_tag"`
	AppriseFormat        string `json:"apprise_format" validate:"omitempty,oneof=text html markdown"`
}

// NotificationResponse is a DTO that avoids the ent-generated omitempty on bool fields,
// ensuring false values are always included in JSON responses.
type NotificationResponse struct {
	ID                   uuid.UUID `json:"id"`
	Name                 string    `json:"name"`
	Enabled              bool      `json:"enabled"`
	Type                 string    `json:"type"`
	URL                  string    `json:"url"`
	TriggerVideoSuccess  bool      `json:"trigger_video_success"`
	TriggerLiveSuccess   bool      `json:"trigger_live_success"`
	TriggerError         bool      `json:"trigger_error"`
	TriggerIsLive        bool      `json:"trigger_is_live"`
	VideoSuccessTemplate string    `json:"video_success_template"`
	LiveSuccessTemplate  string    `json:"live_success_template"`
	ErrorTemplate        string    `json:"error_template"`
	IsLiveTemplate       string    `json:"is_live_template"`
	AppriseUrls          string    `json:"apprise_urls"`
	AppriseTitle         string    `json:"apprise_title"`
	AppriseType          string    `json:"apprise_type"`
	AppriseTag           string    `json:"apprise_tag"`
	AppriseFormat        string    `json:"apprise_format"`
	UpdatedAt            time.Time `json:"updated_at"`
	CreatedAt            time.Time `json:"created_at"`
}

// toNotificationResponse converts an ent.Notification to a NotificationResponse DTO.
func toNotificationResponse(n *ent.Notification) NotificationResponse {
	return NotificationResponse{
		ID:                   n.ID,
		Name:                 n.Name,
		Enabled:              n.Enabled,
		Type:                 string(n.Type),
		URL:                  n.URL,
		TriggerVideoSuccess:  n.TriggerVideoSuccess,
		TriggerLiveSuccess:   n.TriggerLiveSuccess,
		TriggerError:         n.TriggerError,
		TriggerIsLive:        n.TriggerIsLive,
		VideoSuccessTemplate: n.VideoSuccessTemplate,
		LiveSuccessTemplate:  n.LiveSuccessTemplate,
		ErrorTemplate:        n.ErrorTemplate,
		IsLiveTemplate:       n.IsLiveTemplate,
		AppriseUrls:          n.AppriseUrls,
		AppriseTitle:         n.AppriseTitle,
		AppriseType:          string(n.AppriseType),
		AppriseTag:           n.AppriseTag,
		AppriseFormat:        string(n.AppriseFormat),
		UpdatedAt:            n.UpdatedAt,
		CreatedAt:            n.CreatedAt,
	}
}

// toNotificationResponses converts a slice of ent.Notification to a slice of NotificationResponse DTOs.
func toNotificationResponses(notifications []*ent.Notification) []NotificationResponse {
	result := make([]NotificationResponse, len(notifications))
	for i, n := range notifications {
		result[i] = toNotificationResponse(n)
	}
	return result
}

// validateNotificationRequest performs custom validation beyond struct tags.
func validateNotificationRequest(notifType string, triggerVideoSuccess, triggerLiveSuccess, triggerError, triggerIsLive bool, videoSuccessTemplate, liveSuccessTemplate, errorTemplate, isLiveTemplate, appriseUrls, appriseTag string) error {
	// At least one trigger must be enabled
	if !triggerVideoSuccess && !triggerLiveSuccess && !triggerError && !triggerIsLive {
		return fmt.Errorf("at least one trigger must be enabled")
	}

	// Enabled triggers must have a non-whitespace template
	if triggerVideoSuccess && strings.TrimSpace(videoSuccessTemplate) == "" {
		return fmt.Errorf("video success template is required when video success trigger is enabled")
	}
	if triggerLiveSuccess && strings.TrimSpace(liveSuccessTemplate) == "" {
		return fmt.Errorf("live success template is required when live success trigger is enabled")
	}
	if triggerError && strings.TrimSpace(errorTemplate) == "" {
		return fmt.Errorf("error template is required when error trigger is enabled")
	}
	if triggerIsLive && strings.TrimSpace(isLiveTemplate) == "" {
		return fmt.Errorf("is live template is required when is live trigger is enabled")
	}

	// Apprise requires at least one of urls or tag
	if notifType == "apprise" && strings.TrimSpace(appriseUrls) == "" && strings.TrimSpace(appriseTag) == "" {
		return fmt.Errorf("apprise notifications require either apprise_urls (stateless) or apprise_tag (stateful)")
	}

	return nil
}

// TestNotificationRequest is the request body for testing a notification.
type TestNotificationRequest struct {
	EventType string `json:"event_type" validate:"required,oneof=video_success live_success error is_live"`
}

// GetNotifications returns all notification configurations.
func (h *Handler) GetNotifications(c echo.Context) error {
	notifications, err := h.Service.NotificationService.GetNotifications(c.Request().Context())
	if err != nil {
		log.Error().Err(err).Msg("error getting notifications")
		return ErrorResponse(c, http.StatusInternalServerError, "error getting notifications")
	}
	return SuccessResponse(c, toNotificationResponses(notifications), "notifications")
}

// GetNotification returns a single notification configuration.
func (h *Handler) GetNotification(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid id")
	}

	n, err := h.Service.NotificationService.GetNotification(c.Request().Context(), id)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrorResponse(c, http.StatusNotFound, "notification not found")
		}
		log.Error().Err(err).Str("id", id.String()).Msg("error getting notification")
		return ErrorResponse(c, http.StatusInternalServerError, "error getting notification")
	}
	return SuccessResponse(c, toNotificationResponse(n), "notification")
}

// CreateNotification creates a new notification configuration.
func (h *Handler) CreateNotification(c echo.Context) error {
	var req CreateNotificationRequest
	if err := c.Bind(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid request body")
	}
	if err := h.Server.Validator.Validate(req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := validateNotificationRequest(
		req.Type,
		req.TriggerVideoSuccess, req.TriggerLiveSuccess, req.TriggerError, req.TriggerIsLive,
		req.VideoSuccessTemplate, req.LiveSuccessTemplate, req.ErrorTemplate, req.IsLiveTemplate,
		req.AppriseUrls, req.AppriseTag,
	); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	n := &ent.Notification{
		Name:                 req.Name,
		Enabled:              req.Enabled,
		Type:                 entNotification.Type(req.Type),
		URL:                  req.URL,
		TriggerVideoSuccess:  req.TriggerVideoSuccess,
		TriggerLiveSuccess:   req.TriggerLiveSuccess,
		TriggerError:         req.TriggerError,
		TriggerIsLive:        req.TriggerIsLive,
		VideoSuccessTemplate: req.VideoSuccessTemplate,
		LiveSuccessTemplate:  req.LiveSuccessTemplate,
		ErrorTemplate:        req.ErrorTemplate,
		IsLiveTemplate:       req.IsLiveTemplate,
		AppriseUrls:          req.AppriseUrls,
		AppriseTitle:         req.AppriseTitle,
		AppriseType:          entNotification.AppriseType(req.AppriseType),
		AppriseTag:           req.AppriseTag,
		AppriseFormat:        entNotification.AppriseFormat(req.AppriseFormat),
	}

	created, err := h.Service.NotificationService.CreateNotification(c.Request().Context(), n)
	if err != nil {
		log.Error().Err(err).Msg("error creating notification")
		return ErrorResponse(c, http.StatusInternalServerError, "error creating notification")
	}
	return SuccessResponse(c, toNotificationResponse(created), "notification created")
}

// UpdateNotification updates an existing notification configuration.
func (h *Handler) UpdateNotification(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid id")
	}

	var req UpdateNotificationRequest
	if err := c.Bind(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid request body")
	}
	if err := h.Server.Validator.Validate(req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := validateNotificationRequest(
		req.Type,
		req.TriggerVideoSuccess, req.TriggerLiveSuccess, req.TriggerError, req.TriggerIsLive,
		req.VideoSuccessTemplate, req.LiveSuccessTemplate, req.ErrorTemplate, req.IsLiveTemplate,
		req.AppriseUrls, req.AppriseTag,
	); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	n := &ent.Notification{
		Name:                 req.Name,
		Enabled:              req.Enabled,
		Type:                 entNotification.Type(req.Type),
		URL:                  req.URL,
		TriggerVideoSuccess:  req.TriggerVideoSuccess,
		TriggerLiveSuccess:   req.TriggerLiveSuccess,
		TriggerError:         req.TriggerError,
		TriggerIsLive:        req.TriggerIsLive,
		VideoSuccessTemplate: req.VideoSuccessTemplate,
		LiveSuccessTemplate:  req.LiveSuccessTemplate,
		ErrorTemplate:        req.ErrorTemplate,
		IsLiveTemplate:       req.IsLiveTemplate,
		AppriseUrls:          req.AppriseUrls,
		AppriseTitle:         req.AppriseTitle,
		AppriseType:          entNotification.AppriseType(req.AppriseType),
		AppriseTag:           req.AppriseTag,
		AppriseFormat:        entNotification.AppriseFormat(req.AppriseFormat),
	}

	updated, err := h.Service.NotificationService.UpdateNotification(c.Request().Context(), id, n)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrorResponse(c, http.StatusNotFound, "notification not found")
		}
		log.Error().Err(err).Str("id", id.String()).Msg("error updating notification")
		return ErrorResponse(c, http.StatusInternalServerError, "error updating notification")
	}
	return SuccessResponse(c, toNotificationResponse(updated), "notification updated")
}

// DeleteNotification deletes a notification configuration.
func (h *Handler) DeleteNotification(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid id")
	}

	if err := h.Service.NotificationService.DeleteNotification(c.Request().Context(), id); err != nil {
		if ent.IsNotFound(err) {
			return ErrorResponse(c, http.StatusNotFound, "notification not found")
		}
		log.Error().Err(err).Str("id", id.String()).Msg("error deleting notification")
		return ErrorResponse(c, http.StatusInternalServerError, "error deleting notification")
	}
	return SuccessResponse(c, nil, "notification deleted")
}

// TestNotification tests a specific notification configuration with dummy data.
func (h *Handler) TestNotification(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid id")
	}

	var req TestNotificationRequest
	if err := c.Bind(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid request body")
	}
	if err := h.Server.Validator.Validate(req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	n, err := h.Service.NotificationService.GetNotification(c.Request().Context(), id)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrorResponse(c, http.StatusNotFound, "notification not found")
		}
		log.Error().Err(err).Str("id", id.String()).Msg("error getting notification for test")
		return ErrorResponse(c, http.StatusInternalServerError, "error getting notification")
	}

	if err := h.Service.NotificationService.SendTestNotification(n, req.EventType); err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("error sending test notification")
		return ErrorResponse(c, http.StatusInternalServerError, "error sending test notification")
	}
	return SuccessResponse(c, nil, "test notification sent")
}
