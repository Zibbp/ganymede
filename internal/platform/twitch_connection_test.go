package platform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func withTwitchTestServers(t *testing.T, authHandler http.HandlerFunc, apiHandler http.HandlerFunc) {
	t.Helper()

	authServer := httptest.NewServer(authHandler)
	apiServer := httptest.NewServer(apiHandler)
	t.Cleanup(func() {
		authServer.Close()
		apiServer.Close()
	})

	previousAuthURL := TwitchAuthUrl
	previousAPIURL := TwitchApiUrl
	TwitchAuthUrl = authServer.URL
	TwitchApiUrl = apiServer.URL
	t.Cleanup(func() {
		TwitchAuthUrl = previousAuthURL
		TwitchApiUrl = previousAPIURL
	})
}

func writeAuthResponse(t *testing.T, w http.ResponseWriter, token string, expiresIn int) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(AuthTokenResponse{
		AccessToken: token,
		ExpiresIn:   expiresIn,
		TokenType:   "bearer",
	}); err != nil {
		t.Fatalf("failed to write auth response: %v", err)
	}
}

func TestTwitchConnectionAuthenticateStoresTokenAndExpiry(t *testing.T) {
	withTwitchTestServers(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST auth request, got %s", r.Method)
		}
		if got := r.URL.Query().Get("client_id"); got != "client-id" {
			t.Fatalf("expected client_id query, got %q", got)
		}
		if got := r.URL.Query().Get("client_secret"); got != "client-secret" {
			t.Fatalf("expected client_secret query, got %q", got)
		}
		if got := r.URL.Query().Get("grant_type"); got != "client_credentials" {
			t.Fatalf("expected grant_type query, got %q", got)
		}
		writeAuthResponse(t, w, "stored-token", 3600)
	}, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected Helix request")
	})

	conn := &TwitchConnection{
		ClientId:     "client-id",
		ClientSecret: "client-secret",
	}
	before := time.Now()

	info, err := conn.Authenticate(context.Background())
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}

	if info.AccessToken != "stored-token" {
		t.Fatalf("expected connection info token stored-token, got %q", info.AccessToken)
	}

	conn.mu.RLock()
	defer conn.mu.RUnlock()

	if conn.AccessToken != "stored-token" {
		t.Fatalf("expected stored token, got %q", conn.AccessToken)
	}
	if !conn.tokenExpiresAt.After(before.Add(59 * time.Minute)) {
		t.Fatalf("expected expiry about an hour in the future, got %s", conn.tokenExpiresAt)
	}
}

func TestTwitchMakeHTTPRequestUsesCurrentToken(t *testing.T) {
	withTwitchTestServers(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected auth request")
	}, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Client-ID"); got != "client-id" {
			t.Fatalf("expected Client-ID client-id, got %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer current-token" {
			t.Fatalf("expected current token authorization, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	})

	conn := &TwitchConnection{
		ClientId:       "client-id",
		ClientSecret:   "client-secret",
		AccessToken:    "current-token",
		tokenExpiresAt: time.Now().Add(time.Hour),
	}

	body, err := conn.twitchMakeHTTPRequest(context.Background(), http.MethodGet, "users", url.Values{"login": []string{"test"}}, nil)
	if err != nil {
		t.Fatalf("twitchMakeHTTPRequest returned error: %v", err)
	}
	if string(body) != `{"data":[]}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestTwitchMakeHTTPRequestRefreshesNearExpiredToken(t *testing.T) {
	var authRequests atomic.Int32

	withTwitchTestServers(t, func(w http.ResponseWriter, r *http.Request) {
		authRequests.Add(1)
		writeAuthResponse(t, w, "fresh-token", 3600)
	}, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer fresh-token" {
			t.Fatalf("expected fresh token authorization, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	})

	conn := &TwitchConnection{
		ClientId:       "client-id",
		ClientSecret:   "client-secret",
		AccessToken:    "stale-token",
		tokenExpiresAt: time.Now().Add(time.Minute),
	}

	if _, err := conn.twitchMakeHTTPRequest(context.Background(), http.MethodGet, "users", nil, nil); err != nil {
		t.Fatalf("twitchMakeHTTPRequest returned error: %v", err)
	}
	if got := authRequests.Load(); got != 1 {
		t.Fatalf("expected 1 auth request, got %d", got)
	}
}

func TestTwitchMakeHTTPRequestRefreshesAndRetriesAfterUnauthorized(t *testing.T) {
	var authRequests atomic.Int32
	var apiRequests atomic.Int32

	withTwitchTestServers(t, func(w http.ResponseWriter, r *http.Request) {
		request := authRequests.Add(1)
		if request == 1 {
			writeAuthResponse(t, w, "initial-token", 3600)
			return
		}
		writeAuthResponse(t, w, "retry-token", 3600)
	}, func(w http.ResponseWriter, r *http.Request) {
		request := apiRequests.Add(1)
		switch request {
		case 1:
			if got := r.Header.Get("Authorization"); got != "Bearer initial-token" {
				t.Fatalf("expected initial token authorization, got %q", got)
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		case 2:
			if got := r.Header.Get("Authorization"); got != "Bearer retry-token" {
				t.Fatalf("expected retry token authorization, got %q", got)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
		default:
			t.Fatalf("unexpected API request %d", request)
		}
	})

	conn := &TwitchConnection{
		ClientId:     "client-id",
		ClientSecret: "client-secret",
	}

	body, err := conn.twitchMakeHTTPRequest(context.Background(), http.MethodGet, "users", nil, nil)
	if err != nil {
		t.Fatalf("twitchMakeHTTPRequest returned error: %v", err)
	}
	if string(body) != `{"data":[]}` {
		t.Fatalf("unexpected body: %s", body)
	}
	if got := authRequests.Load(); got != 2 {
		t.Fatalf("expected 2 auth requests, got %d", got)
	}
	if got := apiRequests.Load(); got != 2 {
		t.Fatalf("expected 2 API requests, got %d", got)
	}
}
