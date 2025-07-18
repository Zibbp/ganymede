package exec

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zibbp/ganymede/internal/utils"
)

// mockProxyType is a helper to create a ProxyType for testing.
type mockProxyType string

func Test_testTwitchHLSProxy_Success(t *testing.T) {
	// Start a test server that always returns 200 OK
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test-Header") != "value" {
			t.Errorf("expected header X-Test-Header to be set")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ok := testTwitchHLSProxy("", ts.URL, "X-Test-Header:value")
	if !ok {
		t.Errorf("expected testTwitchHLSProxy to return true on 200 OK")
	}
}

func Test_testTwitchHLSProxy_FailStatus(t *testing.T) {
	// Start a test server that returns 500
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	ok := testTwitchHLSProxy("", ts.URL, "")
	if ok {
		t.Errorf("expected testTwitchHLSProxy to return false on non-200 status")
	}
}

func Test_testTwitchHLSProxy_BadURL(t *testing.T) {
	ok := testTwitchHLSProxy("", "http://[::1]:namedport", "")
	if ok {
		t.Errorf("expected testTwitchHLSProxy to return false on bad URL")
	}
}

func Test_testHTTPProxy_Success(t *testing.T) {
	// Start a test server that always returns 200 OK
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test-Header") != "value" {
			t.Errorf("expected header X-Test-Header to be set")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Use empty proxy (no proxy)
	ok := testHTTPProxy("", ts.URL, "X-Test-Header:value")
	if !ok {
		t.Errorf("expected testHTTPProxy to return true on 200 OK")
	}
}

func Test_testHTTPProxy_BadProxyURL(t *testing.T) {
	ok := testHTTPProxy("http://[::1]:namedport", "http://example.com", "")
	if ok {
		t.Errorf("expected testHTTPProxy to return false on bad proxy URL")
	}
}

func Test_testHTTPProxy_BadTestURL(t *testing.T) {
	ok := testHTTPProxy("", "http://[::1]:namedport", "")
	if ok {
		t.Errorf("expected testHTTPProxy to return false on bad test URL")
	}
}

func Test_testHTTPProxy_FailStatus(t *testing.T) {
	// Start a test server that returns 404
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	ok := testHTTPProxy("", ts.URL, "")
	if ok {
		t.Errorf("expected testHTTPProxy to return false on non-200 status")
	}
}

func Test_testProxyServer_TwitchHLS(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ok := testProxyServer("", ts.URL, "", utils.ProxyTypeTwitchHLS)
	if !ok {
		t.Errorf("expected testProxyServer to return true for TwitchHLS")
	}
}

func Test_testProxyServer_HTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ok := testProxyServer("", ts.URL, "", utils.ProxyTypeHTTP)
	if !ok {
		t.Errorf("expected testProxyServer to return true for HTTP")
	}
}

func Test_testProxyServer_UnknownType(t *testing.T) {
	ok := testProxyServer("", "http://example.com", "", utils.ProxyType("unknown"))
	if ok {
		t.Errorf("expected testProxyServer to return false for unknown proxy type")
	}
}
