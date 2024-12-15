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

	return c.String(http.StatusOK, webVtt)
}
