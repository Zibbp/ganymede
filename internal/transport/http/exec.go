package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/exec"
)

type ExecService interface {
	GetFfprobeData(path string) (map[string]interface{}, error)
}

type GetFfprobeDataRequest struct {
	Path string `json:"path" validate:"required"`
}

// GetFfprobeData godoc
//
//	@Summary		Get ffprobe data
//	@Description	Get ffprobe data
//	@Tags			exec
//	@Accept			json
//	@Produce		json
//	@Param			body	body		GetFfprobeDataRequest	true	"GetFfprobeDataRequest"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/exec/ffprobe [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetFfprobeData(c echo.Context) error {
	gfd := new(GetFfprobeDataRequest)
	if err := c.Bind(gfd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(gfd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ffprobeData, err := exec.GetFfprobeData(gfd.Path)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, ffprobeData)
}
