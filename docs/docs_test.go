package docs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSwaggerInfoRendersRuntimeDocument(t *testing.T) {
	var document struct {
		Info struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"info"`
		SecurityDefinitions map[string]json.RawMessage `json:"securityDefinitions"`
	}

	rendered := SwaggerInfo.ReadDoc()
	require.NoError(t, json.Unmarshal([]byte(rendered), &document))
	require.Equal(t, "Ganymede API", document.Info.Title)
	require.Contains(t, document.Info.Description, "programmatic API clients with scoped API keys")
	require.Contains(t, document.Info.Description, "not entered in the Swagger Authorize dialog")
	require.Len(t, document.SecurityDefinitions, 1)
	require.Contains(t, document.SecurityDefinitions, "ApiKeyAuth")
	require.NotContains(t, document.SecurityDefinitions, "BrowserSessionAuth")
	require.NotContains(t, document.SecurityDefinitions, "ApiKeyCookieAuth")
	require.NotContains(t, document.SecurityDefinitions, "SessionCookieAuth")
}
