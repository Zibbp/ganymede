package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/kv"
	"golang.org/x/oauth2"
)

type OAuthClaims struct {
	jwt.RegisteredClaims
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

func (s *Service) OAuthRedirect(c echo.Context) error {
	state, err := randString(32)
	if err != nil {
		return err
	}
	nonce, err := randString(32)
	if err != nil {
		return err
	}
	setCallbackCookie(c, "oauth_state", state)
	setCallbackCookie(c, "oauth_nonce", nonce)

	url := s.OAuth.Config.AuthCodeURL(state, oidc.Nonce(nonce))
	err = c.Redirect(http.StatusTemporaryRedirect, url)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) OAuthCallback(c echo.Context) error {
	state, err := c.Cookie("oauth_state")
	if err != nil {
		return fmt.Errorf("state cookie not found: %w", err)
	}
	// Validate state
	if state.Value != c.QueryParam("state") {
		return fmt.Errorf("invalid oauth state, expected '%s', got '%s'", state.Value, c.QueryParam("state"))
	}
	// Exchange code for token
	oauth2Token, err := s.OAuth.Config.Exchange(c.Request().Context(), c.QueryParam("code"))
	if err != nil {
		return fmt.Errorf("failed to exchange token: %w", err)
	}

	verifier := s.OAuth.Provider.Verifier(&oidc.Config{ClientID: s.OAuth.Config.ClientID})

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return fmt.Errorf("no id_token field in oauth2 token")
	}

	idToken, err := verifier.Verify(c.Request().Context(), rawIDToken)
	if err != nil {
		return fmt.Errorf("failed to verify ID token: %w", err)
	}
	nonce, err := c.Cookie("oauth_nonce")
	if err != nil {
		return fmt.Errorf("nonce cookie not found: %w", err)
	}

	if idToken.Nonce != nonce.Value {
		return fmt.Errorf("invalid nonce, expected '%s', got '%s'", nonce.Value, idToken.Nonce)
	}

	resp := struct {
		OAuth2Token   *oauth2.Token
		IDTokenClaims *json.RawMessage
	}{oauth2Token, new(json.RawMessage)}
	if err := idToken.Claims(&resp.IDTokenClaims); err != nil {
		return fmt.Errorf("failed to decode ID token claims: %w", err)
	}

	// Do we need to verify the token??
	// Verify
	//err = idToken.VerifyAccessToken(oauth2Token.AccessToken)
	//if err != nil {
	//	return fmt.Errorf("failed to verify access token: %w", err)
	//}

	// User check
	var userInfo UserInfo
	err = idToken.Claims(&userInfo)

	// some providers don't return nickname
	if userInfo.NickName == "" {
		userInfo.NickName = userInfo.PreferredUsername
	}
	if userInfo.Sub == "" || userInfo.NickName == "" {
		return fmt.Errorf("invalid user info: %w", err)
	}

	err = s.OAuthUserCheck(c, userInfo)
	if err != nil {
		return fmt.Errorf("failed to check user: %w", err)
	}

	// Get access token expiry
	accessTokenExpire := time.Now().Add(time.Duration(oauth2Token.Expiry.Unix()-time.Now().Unix()) * time.Second)
	// Get refresh token expiry
	refreshTokenExpire := time.Now().Add(30 * 24 * time.Hour)

	// Set cookies
	setOauthCookie(c, "oauth_access_token", oauth2Token.AccessToken, accessTokenExpire)
	setOauthCookie(c, "oauth_refresh_token", oauth2Token.RefreshToken, refreshTokenExpire)

	return nil
}

func (s *Service) OAuthTokenRefresh(c echo.Context, refreshToken string) error {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	newToken, err := s.OAuth.Config.TokenSource(c.Request().Context(), token).Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	accessTokenExpire := time.Now().Add(time.Duration(newToken.Expiry.Unix()-time.Now().Unix()) * time.Second)
	refreshTokenExpire := time.Now().Add(30 * 24 * time.Hour)
	setOauthCookie(c, "oauth_access_token", newToken.AccessToken, accessTokenExpire)
	setOauthCookie(c, "oauth_refresh_token", newToken.RefreshToken, refreshTokenExpire)
	return nil
}

func (s *Service) OAuthLogout(c echo.Context) error {
	// Session end
	// https://openid.net/specs/openid-connect-session-1_0.html#RPLogout

	var endpoints struct {
		RevocationEndpoint string `json:"revocation_endpoint"`
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	err := s.OAuth.Provider.Claims(&endpoints)
	if err != nil {
		return fmt.Errorf("failed to get endpoints: %w", err)
	}

	clearCookie(c, "oauth_access_token")
	clearCookie(c, "oauth_refresh_token")
	clearCookie(c, "oauth_state")
	clearCookie(c, "oauth_nonce")

	return nil
}

func clearCookie(c echo.Context, name string) {
	cookie := new(http.Cookie)
	cookie.Name = name
	cookie.Value = ""
	cookie.Expires = time.Now().Add(-1 * time.Hour)
	cookie.Path = "/"
	c.SetCookie(cookie)
}

func CheckOAuthAccessToken(c echo.Context, accessToken string) (*UserInfo, error) {
	env := config.GetEnvConfig()
	// Get JWKS from KV store
	jwksString := kv.DB().Get("jwks")
	if jwksString == "" {
		return nil, fmt.Errorf("jwks not found")
	}
	// Parse JWKS
	jwks, err := keyfunc.NewJSON(json.RawMessage(jwksString))
	if err != nil {
		return nil, fmt.Errorf("failed to parse jwks: %w", err)
	}
	// Remove new line characters from access token
	newAccessToken := strings.Replace(accessToken, "\n", "", -1)

	// Parse token - this will also verify the signature
	token, err := jwt.Parse(newAccessToken, jwks.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse access token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid access token")
	}

	// Check aud
	audClaim, ok := token.Claims.(jwt.MapClaims)["aud"]
	if !ok {
		return nil, fmt.Errorf("missing aud claim")
	}

	validAud := false

	switch v := audClaim.(type) {
	case string:
		if v == env.OAuthClientID {
			validAud = true
		}
	case []interface{}:
		for _, a := range v {
			if s, ok := a.(string); ok && s == env.OAuthClientID {
				validAud = true
				break
			}
		}
	case []string:
		for _, a := range v {
			if a == env.OAuthClientID {
				validAud = true
				break
			}
		}
	case nil:
		return nil, fmt.Errorf("aud claim is nil")
	default:
		return nil, fmt.Errorf("unexpected aud claim type: %T", v)
	}

	if !validAud {
		return nil, fmt.Errorf("invalid aud claim")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to parse claims")
	}

	userInfo := UserInfo{
		Sub:      claims["sub"].(string),
		NickName: claims["nickname"].(string),
	}

	return &userInfo, nil
}

func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setCallbackCookie(c echo.Context, name, value string) {
	env := config.GetEnvConfig()
	cookieDomain := env.CookieDomain
	cookie := new(http.Cookie)
	cookie.Name = name
	cookie.Value = value
	cookie.Expires = time.Now().Add(1 * time.Hour)
	cookie.Path = "/"
	// Http-only helps mitigate the risk of client side script accessing the protected cookie.
	cookie.HttpOnly = false
	cookie.SameSite = http.SameSiteLaxMode
	if cookieDomain != "" {
		cookie.Domain = cookieDomain
	}

	c.SetCookie(cookie)
}

func setOauthCookie(c echo.Context, name, value string, time time.Time) {
	env := config.GetEnvConfig()
	cookieDomain := env.CookieDomain
	cookie := new(http.Cookie)
	cookie.Name = name
	cookie.Value = value
	cookie.Expires = time
	cookie.Path = "/"
	// Http-only helps mitigate the risk of client side script accessing the protected cookie.
	cookie.HttpOnly = false
	cookie.SameSite = http.SameSiteLaxMode
	if cookieDomain != "" {
		cookie.Domain = cookieDomain
	}

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
	defer jwksResp.Body.Close()
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
