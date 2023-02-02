package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/enttest"
	"github.com/zibbp/ganymede/internal/database"
	httpHandler "github.com/zibbp/ganymede/internal/transport/http"
	"github.com/zibbp/ganymede/internal/user"
	"github.com/zibbp/ganymede/internal/utils"
)

// * TestGetUsers tests the GetUsers function
// Gets all users
func TestGetUsers(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			UserService: user.NewService(&database.Database{Client: client}),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a user
	dbUser, err := client.User.Create().SetUsername("test").SetPassword("test").Save(context.Background())
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)

	if assert.NoError(t, h.GetUsers(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response []map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(response))
		assert.Equal(t, dbUser.ID.String(), response[0]["id"])

	}
}

// * TestGetUser tests the GetUser function
// Gets a user by id
func TestGetUser(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			UserService: user.NewService(&database.Database{Client: client}),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a user
	dbUser, err := client.User.Create().SetUsername("test").SetPassword("test").Save(context.Background())
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/user/%s", dbUser.ID.String()), nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(dbUser.ID.String())

	if assert.NoError(t, h.GetUser(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, dbUser.ID.String(), response["id"])

	}
}

// * TestUpdateUser tests the UpdateUser function
// Update a user
func TestUpdateUser(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			UserService: user.NewService(&database.Database{Client: client}),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a user
	dbUser, err := client.User.Create().SetUsername("test").SetPassword("test").Save(context.Background())
	assert.NoError(t, err)

	updateUserJson := `{
		"username": "test2",
		"role": "admin"
	}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/user/%s", dbUser.ID.String()), strings.NewReader(updateUserJson))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(dbUser.ID.String())

	if assert.NoError(t, h.UpdateUser(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "test2", response["username"])

	}
}

// * TestDeleteUser tests the DeleteUser function
// Delete a user
func TestDeleteUser(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			UserService: user.NewService(&database.Database{Client: client}),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a user
	dbUser, err := client.User.Create().SetUsername("test").SetPassword("test").Save(context.Background())
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/user/%s", dbUser.ID.String()), nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(dbUser.ID.String())

	if assert.NoError(t, h.DeleteUser(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check if user is deleted
		user, err := client.User.Get(context.Background(), dbUser.ID)
		assert.Error(t, err)
		assert.Nil(t, user)

	}
}
