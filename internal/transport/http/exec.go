package http

import (
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/exec"
	"net/http"
)

type ExecService interface {
	GetFfprobeData(path string) (map[string]interface{}, error)
}

type GetFfprobeDataRequest struct {
	Path string `json:"path" validate:"required"`
}

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
