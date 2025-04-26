package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/kv"
	"golang.org/x/oauth2"
)

type OAuthClaims struct {
	jwt.RegisteredClaims
}

type OIDCCLaims struct {
	Sub               string   `json:"sub"`
	Nonce             string   `json:"nonce"`
	Name              string   `json:"name"`
	GivenName         string   `json:"given_name"`
	PreferredUsername string   `json:"preferred_username"`
	Nickname          string   `json:"nickname"`
	Groups            []string `json:"groups"`
}

type UserInfo struct {
	Sub               string   `json:"sub"`
	Exp               int64    `json:"exp"`
	Iat               int64    `json:"iat"`
	Name              string   `json:"name"`
	GivenName         string   `json:"given_name"`
	FamilyName        string   `json:"family_name"`
	PreferredUsername string   `json:"preferred_username"`
	NickName          string   `json:"nickname"`
	Groups            []string `json:"groups"`
}

type OAuthResponse struct {
	OAuth2Token *oauth2.Token
	UserInfo    UserInfo
}

func generateSecureRandomString() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (s *Service) OAuthRedirect(c echo.Context) error {
	// generate state and nonce
	state, err := generateSecureRandomString()
	if err != nil {
		return err
	}
	nonce, err := generateSecureRandomString()
	if err != nil {
		return err
	}

	stateCookie := new(http.Cookie)
	stateCookie.Name = "oidc_state"
	stateCookie.Value = state
	stateCookie.Path = "/"
	stateCookie.HttpOnly = true
	stateCookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(stateCookie)

	nonceCookie := new(http.Cookie)
	nonceCookie.Name = "oidc_nonce"
	nonceCookie.Value = nonce
	nonceCookie.Path = "/"
	nonceCookie.HttpOnly = true
	nonceCookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(nonceCookie)

	authURL := s.OAuth.Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.AccessTypeOffline,
	)
	err = c.Redirect(http.StatusTemporaryRedirect, authURL)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) OAuthCallback(c echo.Context) (*ent.User, error) {
	// Retrieve and validate state
	stateCookie, err := c.Cookie("oidc_state")
	if err != nil {
		return nil, fmt.Errorf("state cookie not found: %w", err)
	}

	// Retrieve and validate nonce
	nonceCookie, err := c.Cookie("oidc_nonce")
	if err != nil {
		return nil, fmt.Errorf("nonce cookie not found: %w", err)
	}

	// Verify state to prevent CSRF
	urlState := c.QueryParam("state")
	if urlState != stateCookie.Value {
		return nil, fmt.Errorf("invalid state parameter")
	}

	// Exchange authorization code for tokens
	oauth2Token, err := s.OAuth.Config.Exchange(c.Request().Context(), c.QueryParam("code"))
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	// Extract the ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no ID token")
	}

	// Verify ID token
	verifier := s.OAuth.Provider.Verifier(&oidc.Config{ClientID: s.OAuth.Config.ClientID})
	idToken, err := verifier.Verify(c.Request().Context(), rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	var claims OIDCCLaims

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// Debug claims in dev
	if s.EnvConfig.Development {
		err := debugOidcClaims(idToken)
		if err != nil {
			return nil, err
		}
	}

	// Verify nonce to prevent replay attack
	if claims.Nonce != nonceCookie.Value {
		return nil, fmt.Errorf("invalid nonce")
	}

	// Clear cookies
	clearCookie(c, "oidc_state")
	clearCookie(c, "oidc_nonce")

	// assert required fields
	if claims.PreferredUsername == "" {
		return nil, fmt.Errorf("preferred_username required, ensure your idP is returning this claim")
	}

	// create or update user
	user, err := s.OAuthUserCheck(c.Request().Context(), claims)
	if err != nil {
		return nil, fmt.Errorf("error creating or updating users: %v", err)
	}

	return user, nil
}

func debugOidcClaims(idToken *oidc.IDToken) error {
	var claimsDebug map[string]interface{}

	// Extract all claims into the map
	if err := idToken.Claims(&claimsDebug); err != nil {
		return fmt.Errorf("failed to parse claims: %v", err)
	}

	// Pretty print all claims
	fmt.Println("=== DEBUG: ALL TOKEN CLAIMS ===")

	// Use JSON marshaling for a nice, readable output
	prettyJSON, err := json.MarshalIndent(claimsDebug, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format claims: %v", err)
	}

	fmt.Println(string(prettyJSON))
	fmt.Println("=== END OF CLAIMS ===")

	return nil
}

func clearCookie(c echo.Context, name string) {
	cookie := new(http.Cookie)
	cookie.Name = name
	cookie.Value = ""
	cookie.MaxAge = -1
	cookie.Path = "/"
	c.SetCookie(cookie)
}

func FetchJWKS(ctx context.Context) error {
	env := config.GetEnvConfig()
	providerURL := env.OAuthProviderURL
	provider, err := oidc.NewProvider(context.Background(), providerURL)
	if err != nil {
		return err
	}

	// Get JWKS uri
	var claims struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := provider.Claims(&claims); err != nil {
		return fmt.Errorf("failed to decode provider claims: %w", err)
	}

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", claims.JWKSURI, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	jwksResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer func() {
		if err := jwksResp.Body.Close(); err != nil {
			fmt.Printf("failed to close body: %v\n", err)
		}
	}()
	body, err := io.ReadAll(jwksResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}
	var jwks jose.JSONWebKeySet
	err = json.Unmarshal(body, &jwks)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JWKS: %w", err)
	}

	// jwks to string
	jwksString, err := json.Marshal(jwks)
	if err != nil {
		return fmt.Errorf("failed to marshal JWKS: %w", err)
	}

	kv.DB().Set("jwks", string(jwksString))

	log.Debug().Msg("fetched jwks")

	return nil
}
