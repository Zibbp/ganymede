package http

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/notification"
)

// TestNotification godoc
//
//	@Summary		Test notification
//	@Description	Test notification
//	@Tags			notification
//	@Accept			json
//	@Produce		json
//	@Param			type	query		string	true	"Type of notification to test"
//	@Success		200		{object}	string
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/notification/test [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) TestNotification(c echo.Context) error {

	notificationType := c.QueryParam("type")
	if notificationType == "" {
		return ErrorResponse(c, http.StatusBadRequest, "type is required")
	}

	testChannel := ent.Channel{
		ID:          uuid.New(),
		ExtID:       "1234456789",
		DisplayName: "Test Channel",
	}
	testVod := ent.Vod{
		ID:         uuid.New(),
		ExtID:      "987654321",
		Platform:   "twitch",
		Type:       "archive",
		Title:      "Demo Notification Title",
		Duration:   100,
		Views:      4510,
		Resolution: "best",
		StreamedAt: time.Now(),
		CreatedAt:  time.Now(),
	}
	testQueue := ent.Queue{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
	}
	failedTask := "video_download"

	switch notificationType {
	case "video_success":
		notification.SendVideoArchiveSuccessNotification(&testChannel, &testVod, &testQueue)
	case "live_success":
		notification.SendLiveArchiveSuccessNotification(&testChannel, &testVod, &testQueue)
	case "error":
		notification.SendErrorNotification(&testChannel, &testVod, &testQueue, failedTask)
	case "is_live":
		notification.SendLiveNotification(&testChannel, &testVod, &testQueue, "Demo Game")
	default:
		return ErrorResponse(c, http.StatusBadRequest, "type is invalid")
	}

	return SuccessResponse(c, "", "sent")
}
