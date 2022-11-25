package http

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/notification"
)

func (h *Handler) TestNotification(c echo.Context) error {

	notificationType := c.QueryParam("type")
	if notificationType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "type is required")
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
		notification.SendLiveNotification(&testChannel, &testVod, &testQueue)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "type is invalid")
	}

	return c.JSON(http.StatusOK, "ok")
}
