package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/api_key"
	"github.com/zibbp/ganymede/internal/utils"
)

// statusForApiKeyServiceError maps service-layer errors to HTTP
// statuses: validation failures (unknown scope, empty list) → 400;
// the row not existing (or being revoked) → 404; everything else is
// an internal failure → 500.
//
// Both api_key.ErrNotFound (returned by the conditional-update path)
// and ent.IsNotFound (returned by Only) map to 404. The string form
// of the error is preserved in the response body so admins still see
// "unknown scope: \"bogus:read\"" rather than a generic "bad request".
func statusForApiKeyServiceError(err error) int {
	switch {
	case errors.Is(err, api_key.ErrInvalidScope), errors.Is(err, api_key.ErrEmptyScopes):
		return http.StatusBadRequest
	case errors.Is(err, api_key.ErrNotFound), ent.IsNotFound(err):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// ApiKeyService is the surface the HTTP handlers need from the api_key
// package. It mirrors the methods on *api_key.Service that handlers call;
// kept narrow so handler tests can stub it without spinning up the full
// service.
type ApiKeyService interface {
	Create(ctx context.Context, name, description string, scopes []utils.ApiKeyScope, createdBy uuid.UUID) (*ent.ApiKey, string, error)
	Update(ctx context.Context, id uuid.UUID, name, description string, scopes []utils.ApiKeyScope) (*ent.ApiKey, error)
	List(ctx context.Context, includeRevoked bool) ([]*ent.ApiKey, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

// CreateApiKeyRequest is the JSON body for POST /admin/api-keys.
//
// Each entry in Scopes must be a known utils.ApiKeyScope (resource:tier);
// validation runs in the service layer (rather than via a struct tag) so
// 400 responses can name the offending scope rather than dumping the full
// catalog of 45+ valid strings.
type CreateApiKeyRequest struct {
	Name        string   `json:"name"        validate:"required,min=3,max=50"`
	Description string   `json:"description" validate:"max=500"`
	Scopes      []string `json:"scopes"      validate:"required,min=1"`
}

// UpdateApiKeyRequest is the JSON body for PUT /admin/api-keys/:id.
// Same shape as CreateApiKeyRequest but applies to an existing key —
// the server replaces all editable fields (name, description, scopes)
// in one call. The prefix and secret are immutable; rotating a key
// still means revoke + create.
type UpdateApiKeyRequest struct {
	Name        string   `json:"name"        validate:"required,min=3,max=50"`
	Description string   `json:"description" validate:"max=500"`
	Scopes      []string `json:"scopes"      validate:"required,min=1"`
}

// apiKeyDTO is the JSON shape returned in list and create responses. We
// build this explicitly rather than marshalling *ent.ApiKey so the
// hashed_secret column never leaves the server, even by accident.
type apiKeyDTO struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Prefix      string    `json:"prefix"`
	// Scopes is the raw list of resource:tier strings granted to the key.
	// Frontend converts these to badges; clients use them to predict what
	// requests will succeed.
	Scopes []string `json:"scopes"`
	// CreatedByID is the UUID of the admin who minted this key. Null
	// for keys created before the audit edge was added. Frontend uses
	// it to show "minted by <username>" in the list and detail views.
	CreatedByID *uuid.UUID `json:"created_by_id"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	// RevokedAt is null for active keys; set when an admin revokes
	// the key. Surfaced in list responses when ?include_revoked=true
	// is passed so admins can audit historical state.
	RevokedAt *time.Time `json:"revoked_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// createApiKeyResponse is the response body for POST /admin/api-keys. The
// secret is included exactly once, here at creation time. Subsequent GETs
// only return the prefix.
type createApiKeyResponse struct {
	ApiKey apiKeyDTO `json:"api_key"`
	Secret string    `json:"secret"`
}

func toAPIKeyDTO(k *ent.ApiKey) apiKeyDTO {
	scopes := k.Scopes
	if scopes == nil {
		// Avoid serialising a null when the column happens to be empty.
		scopes = []string{}
	}
	return apiKeyDTO{
		ID:          k.ID,
		Name:        k.Name,
		Description: k.Description,
		Prefix:      k.Prefix,
		Scopes:      scopes,
		CreatedByID: k.CreatedByID,
		LastUsedAt:  k.LastUsedAt,
		RevokedAt:   k.RevokedAt,
		CreatedAt:   k.CreatedAt,
	}
}

// ListApiKeys godoc
//
//	@Summary		List API keys
//	@Description	Returns API keys, newest first. By default only active (non-revoked) keys are returned; pass ?include_revoked=true for a full audit listing. Secrets are never returned by this endpoint.
//	@Tags			admin
//	@Produce		json
//	@Param			include_revoked	query		bool	false	"include revoked keys"
//	@Success		200				{object}	[]apiKeyDTO
//	@Failure		500				{object}	utils.ErrorResponse
//	@Router			/admin/api-keys [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) ListApiKeys(c echo.Context) error {
	if h.Service.ApiKeyService == nil {
		return ErrorResponse(c, http.StatusInternalServerError, "api key service not configured")
	}
	// Accept the common boolean string forms ("true", "1") for the
	// include_revoked toggle. strconv.ParseBool covers both.
	includeRevoked := false
	if raw := c.QueryParam("include_revoked"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			includeRevoked = v
		}
	}
	keys, err := h.Service.ApiKeyService.List(c.Request().Context(), includeRevoked)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("error listing api keys: %v", err))
	}
	out := make([]apiKeyDTO, 0, len(keys))
	for _, k := range keys {
		out = append(out, toAPIKeyDTO(k))
	}
	return SuccessResponse(c, out, "api keys")
}

// CreateApiKey godoc
//
//	@Summary		Create an API key
//	@Description	Mints a new admin-managed API key. The full secret is returned in the response and is the only time it is visible — store it securely.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CreateApiKeyRequest	true	"create api key payload"
//	@Success		201		{object}	createApiKeyResponse
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/admin/api-keys [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) CreateApiKey(c echo.Context) error {
	if h.Service.ApiKeyService == nil {
		return ErrorResponse(c, http.StatusInternalServerError, "api key service not configured")
	}
	var req CreateApiKeyRequest
	if err := c.Bind(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	scopes := utils.ApiKeyScopesFromStrings(req.Scopes)
	for _, s := range scopes {
		if !s.IsValid() {
			return ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("unknown scope: %q", s))
		}
	}

	// /admin/api-keys is session-only, so userFromContext is the
	// authenticated admin; we record their id for audit attribution.
	// Falls back to uuid.Nil if somehow absent — Create will skip the
	// FK rather than blow up.
	var createdBy uuid.UUID
	if u := userFromContext(c); u != nil {
		createdBy = u.ID
	}

	created, secret, err := h.Service.ApiKeyService.Create(
		c.Request().Context(),
		req.Name,
		req.Description,
		scopes,
		createdBy,
	)
	if err != nil {
		return ErrorResponse(c, statusForApiKeyServiceError(err), fmt.Sprintf("error creating api key: %v", err))
	}

	resp := createApiKeyResponse{
		ApiKey: toAPIKeyDTO(created),
		Secret: secret,
	}
	return c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    resp,
		Message: "api key created — store the secret now, it will not be shown again",
	})
}

// UpdateApiKey godoc
//
//	@Summary		Update an API key
//	@Description	Replaces an existing API key's editable fields (name, description, scopes). Prefix and secret are immutable — rotating a key still means revoke + create.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"api key id"
//	@Param			body	body		UpdateApiKeyRequest	true	"update api key payload"
//	@Success		200		{object}	apiKeyDTO
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		404		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/admin/api-keys/{id} [put]
//	@Security		ApiKeyCookieAuth
func (h *Handler) UpdateApiKey(c echo.Context) error {
	if h.Service.ApiKeyService == nil {
		return ErrorResponse(c, http.StatusInternalServerError, "api key service not configured")
	}
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid id")
	}

	var req UpdateApiKeyRequest
	if err := c.Bind(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	scopes := utils.ApiKeyScopesFromStrings(req.Scopes)
	for _, s := range scopes {
		if !s.IsValid() {
			return ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("unknown scope: %q", s))
		}
	}

	updated, err := h.Service.ApiKeyService.Update(
		c.Request().Context(),
		id,
		req.Name,
		req.Description,
		scopes,
	)
	if err != nil {
		// statusForApiKeyServiceError maps validation → 400, not-found
		// (revoked or missing) → 404, everything else → 500.
		status := statusForApiKeyServiceError(err)
		msg := fmt.Sprintf("error updating api key: %v", err)
		if status == http.StatusNotFound {
			msg = "api key not found"
		}
		return ErrorResponse(c, status, msg)
	}
	return SuccessResponse(c, toAPIKeyDTO(updated), "api key updated")
}

// DeleteApiKey godoc
//
//	@Summary		Revoke an API key
//	@Description	Soft-deletes an API key by setting revoked_at. The verification cache is flushed so the key stops authenticating immediately.
//	@Tags			admin
//	@Param			id	path	string	true	"api key id"
//	@Success		200
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/admin/api-keys/{id} [delete]
//	@Security		ApiKeyCookieAuth
func (h *Handler) DeleteApiKey(c echo.Context) error {
	if h.Service.ApiKeyService == nil {
		return ErrorResponse(c, http.StatusInternalServerError, "api key service not configured")
	}
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid id")
	}
	if err := h.Service.ApiKeyService.Revoke(c.Request().Context(), id); err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("error revoking api key: %v", err))
	}
	return SuccessResponse(c, nil, "api key revoked")
}
