package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/storage"
)

type StorageService interface {
	ListFindings(ctx context.Context) (storage.ListFindingsResponse, error)
	DeleteFindings(ctx context.Context, ids []uuid.UUID, username string) (storage.ActionResult, error)
	ImportFindings(ctx context.Context, ids []uuid.UUID, username string) (storage.ActionResult, error)
}

type StorageFindingsRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,max=100,dive,required"`
}

// ListStorageFindings godoc
//
//	@Summary		List storage findings
//	@Description	List directories that hold the files of a video that is not in the database, and the state of the reconciliation task
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	storage.ListFindingsResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/admin/storage-findings [get]
//	@Security		ApiKeyCookieAuth
//	@Security		ApiKeyAuth
func (h *Handler) ListStorageFindings(c echo.Context) error {
	resp, err := h.Service.StorageService.ListFindings(c.Request().Context())
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("Error retrieving storage findings: %v", err))
	}
	return SuccessResponse(c, resp, "Storage Findings")
}

// DeleteStorageFindings godoc
//
//	@Summary		Delete storage findings
//	@Description	Delete the directories of the given findings from disk
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			body	body		StorageFindingsRequest	true	"Findings to delete"
//	@Success		200		{object}	storage.ActionResult
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/admin/storage-findings/delete [post]
//	@Security		ApiKeyCookieAuth
//	@Security		ApiKeyAuth
func (h *Handler) DeleteStorageFindings(c echo.Context) error {
	var req StorageFindingsRequest
	if err := c.Bind(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := h.Server.Validator.Validate(req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	result, err := h.Service.StorageService.DeleteFindings(c.Request().Context(), req.IDs, usernameFromContext(c))
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, result, "Storage Findings Deleted")
}

// ImportStorageFindings godoc
//
//	@Summary		Import storage findings
//	@Description	Turn the directories of the given findings back into videos
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			body	body		StorageFindingsRequest	true	"Findings to import"
//	@Success		200		{object}	storage.ActionResult
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/admin/storage-findings/import [post]
//	@Security		ApiKeyCookieAuth
//	@Security		ApiKeyAuth
func (h *Handler) ImportStorageFindings(c echo.Context) error {
	var req StorageFindingsRequest
	if err := c.Bind(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := h.Server.Validator.Validate(req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	result, err := h.Service.StorageService.ImportFindings(c.Request().Context(), req.IDs, usernameFromContext(c))
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, result, "Storage Findings Imported")
}

// usernameFromContext returns the name of the acting user for the log, or the API key marker.
func usernameFromContext(c echo.Context) string {
	if user := userFromContext(c); user != nil {
		return user.Username
	}
	return "api-key"
}
