package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
)

type ChapterService interface {
	GetVideoChapters(videoId uuid.UUID) ([]*ent.Chapter, error)
	CreateWebVtt(chapters []*ent.Chapter) (string, error)
}

// GetVideoChapters godoc
//
//	@Summary		Get video chapters
//	@Description	Get chapters for a video
//	@Tags			chapter
//	@Accept			json
//	@Produce		json
//	@Param			videoId	path		string	true	"Video ID"
//	@Success		200		{array}		ent.Chapter
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/chapter/video/{videoId} [get]
func (h *Handler) GetVideoChapters(c echo.Context) error {
	videoId := c.Param("videoId")

	// parse uuid
	vid, err := uuid.Parse(videoId)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	chapters, err := h.Service.ChapterService.GetVideoChapters(vid)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, chapters, "video chapters")
}

// GetWebVTTChapters godoc
//
//	@Summary		Get video chapters as WebVTT
//	@Description	Get chapters for a video in WebVTT format
//	@Tags			chapter
//	@Accept			json
//	@Produce		text/vtt
//	@Param			videoId	path		string	true	"Video ID"
//	@Success		200		{object}	string
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/chapter/video/{videoId}/webvtt [get]
func (h *Handler) GetWebVTTChapters(c echo.Context) error {
	videoId := c.Param("videoId")

	// parse uuid
	vid, err := uuid.Parse(videoId)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	chapters, err := h.Service.ChapterService.GetVideoChapters(vid)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	webVtt, err := h.Service.ChapterService.CreateWebVtt(chapters)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	// Native <track> elements require the WebVTT MIME type; text/plain
	// risks browsers rejecting the cues.
	return c.Blob(http.StatusOK, "text/vtt", []byte(webVtt))
}
