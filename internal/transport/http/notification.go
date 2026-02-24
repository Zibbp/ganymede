package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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
	SendTestNotification(n *ent.Notification, eventType string)
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

// validateNotificationRequest performs custom validation beyond struct tags.
func validateNotificationRequest(triggerVideoSuccess, triggerLiveSuccess, triggerError, triggerIsLive bool, videoSuccessTemplate, liveSuccessTemplate, errorTemplate, isLiveTemplate string) error {
	// At least one trigger must be enabled
	if !triggerVideoSuccess && !triggerLiveSuccess && !triggerError && !triggerIsLive {
		return fmt.Errorf("at least one trigger must be enabled")
	}

	// Enabled triggers must have a non-empty template
	if triggerVideoSuccess && videoSuccessTemplate == "" {
		return fmt.Errorf("video success template is required when video success trigger is enabled")
	}
	if triggerLiveSuccess && liveSuccessTemplate == "" {
		return fmt.Errorf("live success template is required when live success trigger is enabled")
	}
	if triggerError && errorTemplate == "" {
		return fmt.Errorf("error template is required when error trigger is enabled")
	}
	if triggerIsLive && isLiveTemplate == "" {
		return fmt.Errorf("is live template is required when is live trigger is enabled")
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
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, notifications, "notifications")
}

// GetNotification returns a single notification configuration.
func (h *Handler) GetNotification(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid id")
	}

	n, err := h.Service.NotificationService.GetNotification(c.Request().Context(), id)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, n, "notification")
}

// CreateNotification creates a new notification configuration.
func (h *Handler) CreateNotification(c echo.Context) error {
	var req CreateNotificationRequest
	if err := c.Bind(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := h.Server.Validator.Validate(req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := validateNotificationRequest(
		req.TriggerVideoSuccess, req.TriggerLiveSuccess, req.TriggerError, req.TriggerIsLive,
		req.VideoSuccessTemplate, req.LiveSuccessTemplate, req.ErrorTemplate, req.IsLiveTemplate,
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
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, created, "notification created")
}

// UpdateNotification updates an existing notification configuration.
func (h *Handler) UpdateNotification(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid id")
	}

	var req UpdateNotificationRequest
	if err := c.Bind(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := h.Server.Validator.Validate(req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := validateNotificationRequest(
		req.TriggerVideoSuccess, req.TriggerLiveSuccess, req.TriggerError, req.TriggerIsLive,
		req.VideoSuccessTemplate, req.LiveSuccessTemplate, req.ErrorTemplate, req.IsLiveTemplate,
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
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, updated, "notification updated")
}

// DeleteNotification deletes a notification configuration.
func (h *Handler) DeleteNotification(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid id")
	}

	if err := h.Service.NotificationService.DeleteNotification(c.Request().Context(), id); err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
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
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := h.Server.Validator.Validate(req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	n, err := h.Service.NotificationService.GetNotification(c.Request().Context(), id)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	h.Service.NotificationService.SendTestNotification(n, req.EventType)
	return SuccessResponse(c, nil, "test notification sent")
}
