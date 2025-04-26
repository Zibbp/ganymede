package http

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/utils"
)

type ArchiveService interface {
	ArchiveChannel(ctx context.Context, channelName string) (*ent.Channel, error)
	ArchiveVideo(ctx context.Context, input archive.ArchiveVideoInput) error
	ArchiveLivestream(ctx context.Context, input archive.ArchiveVideoInput) error
	ArchiveClip(ctx context.Context, input archive.ArchiveClipInput) error
}

type ArchiveChannelRequest struct {
	ChannelName string `json:"channel_name" validate:"required"`
}
type ArchiveVideoRequest struct {
	VideoId     string           `json:"video_id"`
	ChannelId   string           `json:"channel_id"`
	Quality     utils.VodQuality `json:"quality" validate:"required,oneof=best 1080p 720p 480p 360p 160p audio"`
	ArchiveChat bool             `json:"archive_chat"`
	RenderChat  bool             `json:"render_chat"`
}

// CheckIDType checks if the provided ID is a video id (numeric) or clip (alphanumeric)
func CheckIDType(id string) string {
	// Try to parse as a number
	if _, err := strconv.ParseUint(id, 10, 64); err == nil {
		return "numeric"
	}

	// Check for alphanumeric pattern
	alphanumericRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if alphanumericRegex.MatchString(id) {
		return "alphanumeric"
	}

	return "unknown"
}

// ArchiveChannel godoc
//
//	@Summary		Archive a twitch channel
//	@Description	Archive a twitch channel (creates channel in database and download profile image)
//	@Tags			archive
//	@Accept			json
//	@Produce		json
//	@Param			channel	body		ArchiveChannelRequest	true	"Channel"
//	@Success		200		{object}	ent.Channel
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/archive/channel [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) ArchiveChannel(c echo.Context) error {
	body := new(ArchiveChannelRequest)
	if err := c.Bind(body); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(body); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	channel, err := h.Service.ArchiveService.ArchiveChannel(c.Request().Context(), body.ChannelName)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, channel, "twitch channel created")
}

// ArchiveVideo godoc
//
//	@Summary		Archive a twitch vod
//	@Description	Archive a twitch vod
//	@Tags			archive
//	@Accept			json
//	@Produce		json
//	@Param			vod	body		ArchiveVodRequest	true	"Vod"
//	@Success		200	{object}	archive.TwitchVodResponse
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/archive/vod [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) ArchiveVideo(c echo.Context) error {
	body := new(ArchiveVideoRequest)
	if err := c.Bind(body); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(body); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	if body.VideoId == "" && body.ChannelId == "" {
		return ErrorResponse(c, http.StatusBadRequest, "either channel_id or video_id must be set")
	}

	if body.VideoId != "" && body.ChannelId != "" {
		return ErrorResponse(c, http.StatusBadRequest, "either channel_id or video_id must be set")
	}

	if body.ChannelId != "" {
		// validate channel id
		parsedChannelId, err := uuid.Parse(body.ChannelId)
		if err != nil {
			return ErrorResponse(c, http.StatusBadRequest, err.Error())
		}

		err = h.Service.ArchiveService.ArchiveLivestream(c.Request().Context(), archive.ArchiveVideoInput{
			ChannelId:   parsedChannelId,
			Quality:     body.Quality,
			ArchiveChat: body.ArchiveChat,
			RenderChat:  body.RenderChat,
		})
		if err != nil {
			return ErrorResponse(c, http.StatusInternalServerError, err.Error())
		}
	} else if body.VideoId != "" {
		idType := CheckIDType(body.VideoId)

		switch idType {
		case "numeric":
			err := h.Service.ArchiveService.ArchiveVideo(c.Request().Context(), archive.ArchiveVideoInput{
				VideoId:     body.VideoId,
				Quality:     body.Quality,
				ArchiveChat: body.ArchiveChat,
				RenderChat:  body.RenderChat,
			})
			if err != nil {
				return ErrorResponse(c, http.StatusInternalServerError, err.Error())
			}

		case "alphanumeric":
			err := h.Service.ArchiveService.ArchiveClip(c.Request().Context(), archive.ArchiveClipInput{
				ID:          body.VideoId,
				Quality:     body.Quality,
				ArchiveChat: body.ArchiveChat,
				RenderChat:  body.RenderChat,
			})
			if err != nil {
				return ErrorResponse(c, http.StatusInternalServerError, err.Error())
			}

		default:
			return ErrorResponse(c, http.StatusBadRequest, "Unknown Video ID")
		}

	}

	return SuccessResponse(c, "", "archive started")
}

// debug route to test converting chat files
func (h *Handler) ConvertTwitchChat(c echo.Context) error {
	type Body struct {
		LiveChatPath      string `json:"live_chat_path"`
		ChannelName       string `json:"channel_name"`
		VideoID           string `json:"video_id"`
		VideoExternalID   string `json:"video_external_id"`
		ChannelID         int    `json:"channel_id"`
		PreviousVideoID   string `json:"previous_video_id"`
		FirstMessageEpoch string `json:"first_message_epoch"`
	}
	body := new(Body)
	if err := c.Bind(body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	epoch, err := strconv.Atoi(body.FirstMessageEpoch)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	epochMicroseconds := int64(epoch)
	seconds := epochMicroseconds / 1_000_000
	nanoseconds := (epochMicroseconds % 1_000_000) * 1_000

	t := time.Unix(seconds, nanoseconds)

	envConfig := config.GetEnvConfig()
	outPath := fmt.Sprintf("%s/%s-chat-convert.json", envConfig.TempDir, body.VideoID)

	err = utils.ConvertTwitchLiveChatToTDLChat(body.LiveChatPath, outPath, body.ChannelName, body.VideoID, body.VideoExternalID, body.ChannelID, t, body.PreviousVideoID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusOK)
}
